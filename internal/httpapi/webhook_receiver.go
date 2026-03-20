package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
)

var inboundWebhookKeyPattern = regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*$`)

const (
	inboundWebhookAcceptedStatusCode = http.StatusAccepted
)

var ticketRepoScopeWebhookTarget = inboundWebhookTarget{
	Connector: inboundWebhookKey("ticket-repo-scope"),
	Provider:  inboundWebhookKey("github"),
}

type inboundWebhookKey string

type inboundWebhookTarget struct {
	Connector inboundWebhookKey
	Provider  inboundWebhookKey
}

type inboundWebhookRequest struct {
	Target  inboundWebhookTarget
	Headers http.Header
	Payload []byte
}

type inboundWebhookSummary struct {
	Event      string
	DeliveryID string
	Action     string
	LogArgs    []any
}

type inboundWebhookDispatch struct {
	Summary inboundWebhookSummary
	Payload any
	Ignore  bool
}

type inboundWebhookEndpoint interface {
	Target() inboundWebhookTarget
	MaxPayloadBytes() int64
	VerifySignature(request inboundWebhookRequest) error
	ParseEvent(request inboundWebhookRequest) (inboundWebhookDispatch, error)
	Dispatch(ctx context.Context, dispatch inboundWebhookDispatch) error
}

type inboundWebhookReceiver struct {
	logger    *slog.Logger
	endpoints map[inboundWebhookTarget]inboundWebhookEndpoint
}

type inboundWebhookError struct {
	StatusCode int
	Code       string
	Message    string
}

type errInboundWebhookPayloadTooLarge struct {
	MaxPayloadBytes int64
}

func (e *inboundWebhookError) Error() string {
	if e == nil {
		return ""
	}

	return e.Message
}

func (e errInboundWebhookPayloadTooLarge) Error() string {
	return fmt.Sprintf("request body exceeds %d bytes", e.MaxPayloadBytes)
}

func newInboundWebhookReceiver(logger *slog.Logger, endpoints ...inboundWebhookEndpoint) *inboundWebhookReceiver {
	if logger == nil {
		logger = slog.Default()
	}

	receiver := &inboundWebhookReceiver{
		logger:    logger.With("component", "inbound-webhook-receiver"),
		endpoints: map[inboundWebhookTarget]inboundWebhookEndpoint{},
	}
	for _, endpoint := range endpoints {
		if endpoint == nil {
			continue
		}
		receiver.endpoints[endpoint.Target()] = endpoint
	}

	return receiver
}

func (s *Server) handleLegacyGitHubWebhook(c echo.Context) error {
	return s.handleInboundWebhookTarget(c, ticketRepoScopeWebhookTarget)
}

func (s *Server) handleInboundWebhook(c echo.Context) error {
	target, err := parseInboundWebhookTarget(c.Param("connector"), c.Param("provider"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_WEBHOOK_ROUTE", err.Error())
	}

	return s.handleInboundWebhookTarget(c, target)
}

func (s *Server) handleInboundWebhookTarget(c echo.Context, target inboundWebhookTarget) error {
	if s.inboundWebhooks == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "inbound webhook receiver unavailable")
	}

	return s.inboundWebhooks.Handle(c, target)
}

func (r *inboundWebhookReceiver) Handle(c echo.Context, target inboundWebhookTarget) error {
	if r == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "inbound webhook receiver unavailable")
	}

	endpoint, ok := r.endpoints[target]
	if !ok {
		return writeAPIError(
			c,
			http.StatusNotFound,
			"WEBHOOK_ROUTE_NOT_FOUND",
			fmt.Sprintf("no inbound webhook receiver registered for connector %q and provider %q", target.Connector, target.Provider),
		)
	}

	payload, err := readInboundWebhookPayload(c.Request(), endpoint.MaxPayloadBytes())
	if err != nil {
		var payloadTooLarge errInboundWebhookPayloadTooLarge
		if errors.As(err, &payloadTooLarge) {
			return writeAPIError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", err.Error())
		}

		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", fmt.Sprintf("read request body: %v", err))
	}

	request := inboundWebhookRequest{
		Target:  target,
		Headers: c.Request().Header.Clone(),
		Payload: payload,
	}

	if err := endpoint.VerifySignature(request); err != nil {
		return writeInboundWebhookError(c, err, http.StatusUnauthorized, "INVALID_SIGNATURE")
	}

	dispatch, err := endpoint.ParseEvent(request)
	if err != nil {
		return writeInboundWebhookError(c, err, http.StatusBadRequest, "INVALID_REQUEST")
	}

	logArgs := append(target.logArgs(), dispatch.Summary.LogArgs...)
	if dispatch.Ignore {
		r.logger.Info("webhook ignored", logArgs...)
		return c.NoContent(inboundWebhookAcceptedStatusCode)
	}

	r.logger.Info("webhook accepted", logArgs...)
	if err := endpoint.Dispatch(c.Request().Context(), dispatch); err != nil {
		r.logger.Error("webhook dispatch failed", append(logArgs, "error", err)...)
		return writeAPIError(c, http.StatusInternalServerError, "WEBHOOK_DISPATCH_FAILED", err.Error())
	}

	return c.NoContent(inboundWebhookAcceptedStatusCode)
}

func writeInboundWebhookError(c echo.Context, err error, fallbackStatus int, fallbackCode string) error {
	var apiErr *inboundWebhookError
	if errors.As(err, &apiErr) {
		return writeAPIError(c, apiErr.StatusCode, apiErr.Code, apiErr.Message)
	}

	return writeAPIError(c, fallbackStatus, fallbackCode, err.Error())
}

func readInboundWebhookPayload(request *http.Request, maxPayloadBytes int64) ([]byte, error) {
	if maxPayloadBytes <= 0 {
		maxPayloadBytes = 1 << 20
	}

	payload, err := io.ReadAll(io.LimitReader(request.Body, maxPayloadBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > maxPayloadBytes {
		return nil, errInboundWebhookPayloadTooLarge{MaxPayloadBytes: maxPayloadBytes}
	}

	return payload, nil
}

func parseInboundWebhookTarget(rawConnector string, rawProvider string) (inboundWebhookTarget, error) {
	connector, err := parseInboundWebhookKey("connector", rawConnector)
	if err != nil {
		return inboundWebhookTarget{}, err
	}

	provider, err := parseInboundWebhookKey("provider", rawProvider)
	if err != nil {
		return inboundWebhookTarget{}, err
	}

	return inboundWebhookTarget{
		Connector: connector,
		Provider:  provider,
	}, nil
}

func parseInboundWebhookKey(label string, raw string) (inboundWebhookKey, error) {
	key := inboundWebhookKey(strings.ToLower(strings.TrimSpace(raw)))
	if key == "" {
		return "", fmt.Errorf("%s must not be empty", label)
	}
	if !inboundWebhookKeyPattern.MatchString(string(key)) {
		return "", fmt.Errorf("%s must match %s", label, inboundWebhookKeyPattern.String())
	}

	return key, nil
}

func (t inboundWebhookTarget) logArgs() []any {
	return []any{
		"connector", string(t.Connector),
		"provider", string(t.Provider),
	}
}
