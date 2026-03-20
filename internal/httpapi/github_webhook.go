package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/BetterAndBetterII/openase/ent/ticketreposcope"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
)

const (
	gitHubWebhookEventHeader      = "X-GitHub-Event"
	gitHubWebhookDeliveryIDHeader = "X-GitHub-Delivery"
	gitHubWebhookSignatureHeader  = "X-Hub-Signature-256"
	gitHubWebhookSignaturePrefix  = "sha256="
	gitHubWebhookMaxPayloadBytes  = 1 << 20
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

type gitHubRepoScopeWebhookEndpoint struct {
	server *Server
}

func newGitHubRepoScopeWebhookEndpoint(server *Server) gitHubRepoScopeWebhookEndpoint {
	return gitHubRepoScopeWebhookEndpoint{server: server}
}

func (e gitHubRepoScopeWebhookEndpoint) Target() inboundWebhookTarget {
	return ticketRepoScopeWebhookTarget
}

func (e gitHubRepoScopeWebhookEndpoint) MaxPayloadBytes() int64 {
	return gitHubWebhookMaxPayloadBytes
}

func (e gitHubRepoScopeWebhookEndpoint) VerifySignature(request inboundWebhookRequest) error {
	if err := verifyGitHubWebhookSignature(
		e.server.github.WebhookSecret,
		request.Payload,
		request.Headers.Get(gitHubWebhookSignatureHeader),
	); err != nil {
		return &inboundWebhookError{
			StatusCode: 401,
			Code:       "INVALID_SIGNATURE",
			Message:    err.Error(),
		}
	}

	return nil
}

func (e gitHubRepoScopeWebhookEndpoint) ParseEvent(request inboundWebhookRequest) (inboundWebhookDispatch, error) {
	event, err := parseGitHubWebhookEvent(request.Headers.Get(gitHubWebhookEventHeader))
	if err != nil {
		return inboundWebhookDispatch{}, &inboundWebhookError{
			StatusCode: 400,
			Code:       "INVALID_EVENT",
			Message:    err.Error(),
		}
	}

	deliveryID := strings.TrimSpace(request.Headers.Get(gitHubWebhookDeliveryIDHeader))
	if !event.isSupported() {
		return inboundWebhookDispatch{
			Ignore: true,
			Summary: inboundWebhookSummary{
				Event:      string(event),
				DeliveryID: deliveryID,
				LogArgs: []any{
					"event", event,
					"delivery_id", deliveryID,
				},
			},
		}, nil
	}

	delivery, err := parseGitHubWebhookEnvelope(event, deliveryID, request.Payload)
	if err != nil {
		return inboundWebhookDispatch{}, &inboundWebhookError{
			StatusCode: 400,
			Code:       "INVALID_REQUEST",
			Message:    err.Error(),
		}
	}

	return inboundWebhookDispatch{
		Summary: delivery.summary(),
		Payload: delivery,
	}, nil
}

func (e gitHubRepoScopeWebhookEndpoint) Dispatch(ctx context.Context, dispatch inboundWebhookDispatch) error {
	delivery, ok := dispatch.Payload.(gitHubWebhookEnvelope)
	if !ok {
		return fmt.Errorf("github repo-scope webhook dispatch requires gitHubWebhookEnvelope payload")
	}

	return e.server.syncGitHubRepoScopeStatus(ctx, delivery)
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

func (delivery gitHubWebhookEnvelope) summary() inboundWebhookSummary {
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

	return inboundWebhookSummary{
		Event:      string(delivery.Event),
		DeliveryID: delivery.DeliveryID,
		Action:     delivery.Action,
		LogArgs:    logArgs,
	}
}

func (s *Server) syncGitHubRepoScopeStatus(ctx context.Context, delivery gitHubWebhookEnvelope) error {
	if s.ticketService == nil {
		return nil
	}

	input, ok := mapGitHubWebhookRepoScopeSyncInput(delivery)
	if !ok {
		return nil
	}

	matched, err := s.ticketService.SyncRepoScopePRStatus(ctx, input)
	if err != nil {
		return err
	}
	if !matched.Matched {
		s.logger.Info(
			"github webhook did not match ticket repo scope",
			"event", delivery.Event,
			"delivery_id", delivery.DeliveryID,
			"action", delivery.Action,
			"repository", delivery.Repository.CloneURL,
			"branch", delivery.PullRequest.Branch,
		)
	}
	if matched.Ticket != nil {
		eventType := ticketUpdatedEventType
		if matched.Outcome == ticketservice.RepoScopePRStatusSyncOutcomeFinished {
			eventType = ticketStatusEventType
		}
		if err := s.publishTicketEvent(ctx, eventType, *matched.Ticket); err != nil {
			return err
		}
	}

	return nil
}

func mapGitHubWebhookRepoScopeSyncInput(
	delivery gitHubWebhookEnvelope,
) (ticketservice.SyncRepoScopePRStatusInput, bool) {
	switch delivery.Event {
	case gitHubWebhookEventPullRequest:
		status, ok := mapGitHubPullRequestActionToStatus(delivery.Action, delivery.PullRequest.Merged)
		if !ok {
			return ticketservice.SyncRepoScopePRStatusInput{}, false
		}

		return ticketservice.SyncRepoScopePRStatusInput{
			RepositoryURL:      delivery.Repository.CloneURL,
			RepositoryFullName: delivery.Repository.FullName,
			BranchName:         delivery.PullRequest.Branch,
			PullRequestURL:     delivery.PullRequest.URL,
			PRStatus:           status,
		}, true
	case gitHubWebhookEventPullRequestReview:
		if delivery.Review == nil || !strings.EqualFold(strings.TrimSpace(delivery.Review.State), "changes_requested") {
			return ticketservice.SyncRepoScopePRStatusInput{}, false
		}

		return ticketservice.SyncRepoScopePRStatusInput{
			RepositoryURL:      delivery.Repository.CloneURL,
			RepositoryFullName: delivery.Repository.FullName,
			BranchName:         delivery.PullRequest.Branch,
			PullRequestURL:     delivery.PullRequest.URL,
			PRStatus:           ticketreposcope.PrStatusChangesRequested,
		}, true
	default:
		return ticketservice.SyncRepoScopePRStatusInput{}, false
	}
}

func mapGitHubPullRequestActionToStatus(action string, merged bool) (ticketreposcope.PrStatus, bool) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "opened", "reopened", "ready_for_review":
		return ticketreposcope.PrStatusOpen, true
	case "closed":
		if merged {
			return ticketreposcope.PrStatusMerged, true
		}
		return ticketreposcope.PrStatusClosed, true
	default:
		return "", false
	}
}
