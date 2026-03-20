package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const (
	gitHubWebhookEventHeader        = "X-GitHub-Event"
	gitHubWebhookDeliveryIDHeader   = "X-GitHub-Delivery"
	gitHubWebhookSignatureHeader    = "X-Hub-Signature-256"
	gitHubWebhookSignaturePrefix    = "sha256="
	gitHubWebhookMaxPayloadBytes    = 1 << 20
	gitHubWebhookAcceptedStatusCode = http.StatusAccepted
)

type gitHubWebhookEvent string

const (
	gitHubWebhookEventPullRequest       gitHubWebhookEvent = "pull_request"
	gitHubWebhookEventPullRequestReview gitHubWebhookEvent = "pull_request_review"
)

type gitHubWebhookEnvelope struct {
	Event       gitHubWebhookEvent
	DeliveryID  string
	Action      string
	Repository  gitHubWebhookRepository
	PullRequest gitHubWebhookPullRequest
	Review      *gitHubWebhookReview
}

type gitHubWebhookRepository struct {
	CloneURL string
	FullName string
}

type gitHubWebhookPullRequest struct {
	Number int
	URL    string
	Branch string
	State  string
	Merged bool
}

type gitHubWebhookReview struct {
	State string
}

type rawGitHubWebhookPayload struct {
	Action      string                  `json:"action"`
	Number      int                     `json:"number"`
	Repository  rawGitHubWebhookRepo    `json:"repository"`
	PullRequest *rawGitHubWebhookPR     `json:"pull_request"`
	Review      *rawGitHubWebhookReview `json:"review"`
}

type rawGitHubWebhookRepo struct {
	CloneURL string `json:"clone_url"`
	FullName string `json:"full_name"`
}

type rawGitHubWebhookPR struct {
	Number  int                    `json:"number"`
	HTMLURL string                 `json:"html_url"`
	State   string                 `json:"state"`
	Merged  bool                   `json:"merged"`
	Head    rawGitHubWebhookPRHead `json:"head"`
}

type rawGitHubWebhookPRHead struct {
	Ref string `json:"ref"`
}

type rawGitHubWebhookReview struct {
	State string `json:"state"`
}

var errGitHubWebhookPayloadTooLarge = errors.New("request body exceeds 1048576 bytes")

func (s *Server) handleGitHubWebhook(c echo.Context) error {
	payload, err := readGitHubWebhookPayload(c.Request())
	if err != nil {
		if errors.Is(err, errGitHubWebhookPayloadTooLarge) {
			return writeAPIError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", err.Error())
		}

		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", fmt.Sprintf("read request body: %v", err))
	}

	event, err := parseGitHubWebhookEvent(c.Request().Header.Get(gitHubWebhookEventHeader))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_EVENT", err.Error())
	}

	if err := verifyGitHubWebhookSignature(s.github.WebhookSecret, payload, c.Request().Header.Get(gitHubWebhookSignatureHeader)); err != nil {
		return writeAPIError(c, http.StatusUnauthorized, "INVALID_SIGNATURE", err.Error())
	}

	deliveryID := strings.TrimSpace(c.Request().Header.Get(gitHubWebhookDeliveryIDHeader))
	if !event.isSupported() {
		s.logger.Info("github webhook ignored", "event", event, "delivery_id", deliveryID)
		return c.NoContent(gitHubWebhookAcceptedStatusCode)
	}

	delivery, err := parseGitHubWebhookEnvelope(event, deliveryID, payload)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	logArgs := []any{
		"event", delivery.Event,
		"delivery_id", delivery.DeliveryID,
		"action", delivery.Action,
		"repository", delivery.Repository.CloneURL,
		"pull_request_number", delivery.PullRequest.Number,
		"branch", delivery.PullRequest.Branch,
	}
	if delivery.Review != nil {
		logArgs = append(logArgs, "review_state", delivery.Review.State)
	}
	s.logger.Info("github webhook accepted", logArgs...)

	return c.NoContent(gitHubWebhookAcceptedStatusCode)
}

func readGitHubWebhookPayload(request *http.Request) ([]byte, error) {
	payload, err := io.ReadAll(io.LimitReader(request.Body, gitHubWebhookMaxPayloadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(payload) > gitHubWebhookMaxPayloadBytes {
		return nil, errGitHubWebhookPayloadTooLarge
	}

	return payload, nil
}

func parseGitHubWebhookEvent(raw string) (gitHubWebhookEvent, error) {
	event := gitHubWebhookEvent(strings.TrimSpace(raw))
	if event == "" {
		return "", errors.New("X-GitHub-Event header is required")
	}

	return event, nil
}

func (event gitHubWebhookEvent) isSupported() bool {
	return event == gitHubWebhookEventPullRequest || event == gitHubWebhookEventPullRequestReview
}

func verifyGitHubWebhookSignature(secret string, payload []byte, rawSignature string) error {
	trimmedSecret := strings.TrimSpace(secret)
	if trimmedSecret == "" {
		return nil
	}

	signature := strings.TrimSpace(rawSignature)
	if signature == "" {
		return fmt.Errorf("%s header is required", gitHubWebhookSignatureHeader)
	}
	if !strings.HasPrefix(signature, gitHubWebhookSignaturePrefix) {
		return fmt.Errorf("%s header must use %s format", gitHubWebhookSignatureHeader, gitHubWebhookSignaturePrefix)
	}

	providedDigest, err := hex.DecodeString(strings.TrimPrefix(signature, gitHubWebhookSignaturePrefix))
	if err != nil {
		return fmt.Errorf("%s header must contain a valid hexadecimal digest", gitHubWebhookSignatureHeader)
	}

	mac := hmac.New(sha256.New, []byte(trimmedSecret))
	mac.Write(payload)
	expectedDigest := mac.Sum(nil)
	if !hmac.Equal(providedDigest, expectedDigest) {
		return errors.New("webhook signature mismatch")
	}

	return nil
}

func parseGitHubWebhookEnvelope(
	event gitHubWebhookEvent,
	deliveryID string,
	payload []byte,
) (gitHubWebhookEnvelope, error) {
	var raw rawGitHubWebhookPayload
	if err := json.Unmarshal(payload, &raw); err != nil {
		return gitHubWebhookEnvelope{}, fmt.Errorf("invalid GitHub webhook payload: %w", err)
	}

	action := strings.TrimSpace(raw.Action)
	if action == "" {
		return gitHubWebhookEnvelope{}, errors.New("GitHub webhook payload action is required")
	}

	repoCloneURL := strings.TrimSpace(raw.Repository.CloneURL)
	if repoCloneURL == "" {
		return gitHubWebhookEnvelope{}, errors.New("GitHub webhook payload repository.clone_url is required")
	}

	if raw.PullRequest == nil {
		return gitHubWebhookEnvelope{}, errors.New("GitHub webhook payload pull_request is required")
	}

	prBranch := strings.TrimSpace(raw.PullRequest.Head.Ref)
	if prBranch == "" {
		return gitHubWebhookEnvelope{}, errors.New("GitHub webhook payload pull_request.head.ref is required")
	}

	prNumber := raw.Number
	if prNumber == 0 {
		prNumber = raw.PullRequest.Number
	}
	if prNumber == 0 {
		return gitHubWebhookEnvelope{}, errors.New("GitHub webhook payload number is required")
	}

	delivery := gitHubWebhookEnvelope{
		Event:      event,
		DeliveryID: strings.TrimSpace(deliveryID),
		Action:     action,
		Repository: gitHubWebhookRepository{
			CloneURL: repoCloneURL,
			FullName: strings.TrimSpace(raw.Repository.FullName),
		},
		PullRequest: gitHubWebhookPullRequest{
			Number: prNumber,
			URL:    strings.TrimSpace(raw.PullRequest.HTMLURL),
			Branch: prBranch,
			State:  strings.TrimSpace(raw.PullRequest.State),
			Merged: raw.PullRequest.Merged,
		},
	}

	if event == gitHubWebhookEventPullRequestReview {
		if raw.Review == nil {
			return gitHubWebhookEnvelope{}, errors.New("GitHub webhook payload review is required")
		}

		reviewState := strings.TrimSpace(raw.Review.State)
		if reviewState == "" {
			return gitHubWebhookEnvelope{}, errors.New("GitHub webhook payload review.state is required")
		}
		delivery.Review = &gitHubWebhookReview{State: reviewState}
	}

	return delivery, nil
}
