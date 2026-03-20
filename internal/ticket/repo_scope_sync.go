package ticket

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	"github.com/BetterAndBetterII/openase/ent/ticketreposcope"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/google/uuid"
)

// SyncRepoScopePRStatusInput describes a repo-scope PR status update from an external system.
type SyncRepoScopePRStatusInput struct {
	RepositoryURL      string
	RepositoryFullName string
	BranchName         string
	PullRequestURL     string
	PRStatus           ticketreposcope.PrStatus
}

// RepoScopePRStatusSyncOutcome reports whether syncing changed ticket execution state.
type RepoScopePRStatusSyncOutcome string

const (
	// RepoScopePRStatusSyncOutcomeNone means no ticket status transition was triggered.
	RepoScopePRStatusSyncOutcomeNone RepoScopePRStatusSyncOutcome = ""
	// RepoScopePRStatusSyncOutcomeRetried means the ticket was moved back into retry.
	RepoScopePRStatusSyncOutcomeRetried RepoScopePRStatusSyncOutcome = "retried"
	// RepoScopePRStatusSyncOutcomeFinished means the ticket was finished from merged scopes.
	RepoScopePRStatusSyncOutcomeFinished RepoScopePRStatusSyncOutcome = "finished"
)

// RepoScopePRStatusSyncResult captures the ticket impact of a repo-scope PR sync.
type RepoScopePRStatusSyncResult struct {
	Matched bool
	Outcome RepoScopePRStatusSyncOutcome
	Ticket  *Ticket
}

// SyncRepoScopePRStatus reconciles a PR status update back into the owning ticket.
func (s *Service) SyncRepoScopePRStatus(ctx context.Context, input SyncRepoScopePRStatusInput) (RepoScopePRStatusSyncResult, error) {
	if s.client == nil {
		return RepoScopePRStatusSyncResult{}, ErrUnavailable
	}

	repositoryKey, err := normalizeGitHubRepositoryKey(input.RepositoryURL, input.RepositoryFullName)
	if err != nil {
		return RepoScopePRStatusSyncResult{}, err
	}

	branchName := strings.TrimSpace(input.BranchName)
	if branchName == "" {
		return RepoScopePRStatusSyncResult{}, fmt.Errorf("branch name must not be empty")
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return RepoScopePRStatusSyncResult{}, fmt.Errorf("start repo scope sync tx: %w", err)
	}
	defer rollback(tx)

	scopes, err := tx.TicketRepoScope.Query().
		Where(ticketreposcope.BranchNameEQ(branchName)).
		WithRepo().
		All(ctx)
	if err != nil {
		return RepoScopePRStatusSyncResult{}, fmt.Errorf("query ticket repo scopes: %w", err)
	}

	var matchedScope *ent.TicketRepoScope
	for _, scope := range scopes {
		scopeRepositoryKey, repoErr := normalizeGitHubRepositoryKey(scope.Edges.Repo.RepositoryURL, "")
		if repoErr != nil {
			return RepoScopePRStatusSyncResult{}, fmt.Errorf("normalize project repo %q: %w", scope.Edges.Repo.RepositoryURL, repoErr)
		}
		if scopeRepositoryKey != repositoryKey {
			continue
		}
		if matchedScope != nil {
			return RepoScopePRStatusSyncResult{}, fmt.Errorf("multiple ticket repo scopes matched repository %q and branch %q", repositoryKey, branchName)
		}
		matchedScope = scope
	}

	if matchedScope == nil {
		return RepoScopePRStatusSyncResult{}, nil
	}

	update := tx.TicketRepoScope.UpdateOneID(matchedScope.ID).
		SetPrStatus(input.PRStatus)
	if pullRequestURL := strings.TrimSpace(input.PullRequestURL); pullRequestURL != "" {
		update.SetPullRequestURL(pullRequestURL)
	}

	if _, err := update.Save(ctx); err != nil {
		return RepoScopePRStatusSyncResult{}, fmt.Errorf("update ticket repo scope: %w", err)
	}

	siblingScopes, err := tx.TicketRepoScope.Query().
		Where(ticketreposcope.TicketIDEQ(matchedScope.TicketID)).
		All(ctx)
	if err != nil {
		return RepoScopePRStatusSyncResult{}, fmt.Errorf("query sibling ticket repo scopes: %w", err)
	}

	result := RepoScopePRStatusSyncResult{Matched: true}
	allMerged := len(siblingScopes) > 0
	anyClosed := false
	for _, scope := range siblingScopes {
		if scope.PrStatus != ticketreposcope.PrStatusMerged {
			allMerged = false
		}
		if scope.PrStatus == ticketreposcope.PrStatusClosed {
			anyClosed = true
		}
	}

	switch {
	case anyClosed:
		if err := s.scheduleRepoScopeRetry(ctx, tx, matchedScope.TicketID); err != nil {
			return RepoScopePRStatusSyncResult{}, err
		}
		result.Outcome = RepoScopePRStatusSyncOutcomeRetried
	case allMerged:
		if err := s.finishTicketForMergedRepoScopes(ctx, tx, matchedScope.TicketID); err != nil {
			return RepoScopePRStatusSyncResult{}, err
		}
		result.Outcome = RepoScopePRStatusSyncOutcomeFinished
	}

	if err := tx.Commit(); err != nil {
		return RepoScopePRStatusSyncResult{}, fmt.Errorf("commit repo scope sync tx: %w", err)
	}

	if result.Outcome != RepoScopePRStatusSyncOutcomeNone {
		ticketItem, err := s.Get(ctx, matchedScope.TicketID)
		if err != nil {
			return RepoScopePRStatusSyncResult{}, err
		}
		result.Ticket = &ticketItem
	}

	return result, nil
}

func (s *Service) scheduleRepoScopeRetry(ctx context.Context, tx *ent.Tx, ticketID uuid.UUID) error {
	current, err := tx.Ticket.Get(ctx, ticketID)
	if err != nil {
		return s.mapTicketReadError("get ticket for repo scope retry", err)
	}

	nextAttemptCount := current.AttemptCount + 1
	update := tx.Ticket.UpdateOneID(current.ID).
		ClearAssignedAgentID().
		SetAttemptCount(nextAttemptCount).
		SetConsecutiveErrors(current.ConsecutiveErrors + 1).
		SetNextRetryAt(timeNowUTC().Add(ticketing.ComputeRetryBackoff(nextAttemptCount)))

	if ticketing.ShouldPauseForBudget(current.CostAmount, current.BudgetUsd) {
		update.SetRetryPaused(true).
			SetPauseReason(ticketing.PauseReasonBudgetExhausted.String())
	}

	if _, err := update.Save(ctx); err != nil {
		return s.mapTicketWriteError("update ticket repo scope retry", err)
	}

	return releaseTicketAgentClaim(ctx, tx, current)
}

func (s *Service) finishTicketForMergedRepoScopes(ctx context.Context, tx *ent.Tx, ticketID uuid.UUID) error {
	current, err := tx.Ticket.Get(ctx, ticketID)
	if err != nil {
		return s.mapTicketReadError("get ticket for repo scope finish", err)
	}
	if current.WorkflowID == nil {
		return fmt.Errorf("ticket %s has no workflow to finish", current.ID)
	}

	workflowItem, err := tx.Workflow.Get(ctx, *current.WorkflowID)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrWorkflowNotFound
		}
		return fmt.Errorf("get workflow for repo scope finish: %w", err)
	}
	if workflowItem.FinishStatusID == nil {
		return fmt.Errorf("workflow %s has no finish status configured", workflowItem.ID)
	}

	update := tx.Ticket.UpdateOneID(current.ID).
		SetStatusID(*workflowItem.FinishStatusID).
		SetCompletedAt(timeNowUTC()).
		ClearAssignedAgentID()
	if current.NextRetryAt != nil {
		update.ClearNextRetryAt()
	}
	if current.RetryPaused {
		update.SetRetryPaused(false)
	}
	if current.PauseReason != "" {
		update.ClearPauseReason()
	}

	if _, err := update.Save(ctx); err != nil {
		return s.mapTicketWriteError("finish ticket after repo scopes merged", err)
	}

	return releaseTicketAgentClaim(ctx, tx, current)
}

func timeNowUTC() time.Time {
	return time.Now().UTC()
}

func normalizeGitHubRepositoryKey(rawURL string, rawFullName string) (string, error) {
	if fullName := strings.TrimSpace(rawFullName); fullName != "" {
		return normalizeOwnerRepoPath(fullName)
	}

	repositoryURL := strings.TrimSpace(rawURL)
	if repositoryURL == "" {
		return "", fmt.Errorf("repository URL must not be empty")
	}

	switch {
	case strings.HasPrefix(repositoryURL, "git@github.com:"):
		return normalizeOwnerRepoPath(strings.TrimPrefix(repositoryURL, "git@github.com:"))
	case strings.HasPrefix(repositoryURL, "ssh://git@github.com/"):
		return normalizeOwnerRepoPath(strings.TrimPrefix(repositoryURL, "ssh://git@github.com/"))
	case strings.HasPrefix(repositoryURL, "https://github.com/"), strings.HasPrefix(repositoryURL, "http://github.com/"):
		parsedURL, err := url.Parse(repositoryURL)
		if err != nil {
			return "", fmt.Errorf("parse repository URL %q: %w", repositoryURL, err)
		}
		return normalizeOwnerRepoPath(parsedURL.Path)
	default:
		return normalizeOwnerRepoPath(repositoryURL)
	}
}

func normalizeOwnerRepoPath(raw string) (string, error) {
	trimmed := strings.Trim(strings.TrimSpace(raw), "/")
	trimmed = strings.TrimSuffix(trimmed, ".git")

	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid GitHub repository reference %q", raw)
	}
	if strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", fmt.Errorf("invalid GitHub repository reference %q", raw)
	}

	return strings.ToLower(parts[0] + "/" + parts[1]), nil
}
