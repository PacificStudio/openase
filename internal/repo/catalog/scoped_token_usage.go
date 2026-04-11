package catalog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entorganizationdailytokenusage "github.com/BetterAndBetterII/openase/ent/organizationdailytokenusage"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entprojectconversationrun "github.com/BetterAndBetterII/openase/ent/projectconversationrun"
	entprojectdailytokenusage "github.com/BetterAndBetterII/openase/ent/projectdailytokenusage"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

var errScopedDailyUsageRetry = errors.New("retry scoped daily usage materialization")

type runDailyUsageTotals struct {
	inputTokens       int64
	outputTokens      int64
	cachedInputTokens int64
	reasoningTokens   int64
	totalTokens       int64
	runCount          int
}

type scopedMaterializationTargets struct {
	agentRunIDs        []uuid.UUID
	conversationRunIDs []uuid.UUID
}

type resolvedTokenUsageScope struct {
	scope          domain.TokenUsageScope
	organizationID uuid.UUID
	projectIDs     []uuid.UUID
}

func organizationProjectIDs(ctx context.Context, client *ent.Client, organizationID uuid.UUID) ([]uuid.UUID, error) {
	if client == nil || organizationID == uuid.Nil {
		return nil, nil
	}

	items, err := client.Project.Query().
		Where(entproject.OrganizationIDEQ(organizationID)).
		IDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organization projects for token usage: %w", err)
	}
	return items, nil
}

func (r *EntRepository) getScopedTokenUsage(
	ctx context.Context,
	input domain.GetScopedTokenUsage,
) (domain.ScopedTokenUsageReport, error) {
	resolved, err := r.resolveScopedTokenUsage(ctx, input.Scope)
	if err != nil {
		return domain.ScopedTokenUsageReport{}, err
	}

	recomputedAt := time.Now().UTC()
	totalsByDay, targets, err := r.buildRunUsageTotalsByProjectScope(ctx, resolved.projectIDs, input.FromDate, input.ToDate)
	if err != nil {
		return domain.ScopedTokenUsageReport{}, err
	}
	if err := r.replaceScopedTokenUsageRange(ctx, resolved.scope, input.FromDate, input.ToDate, totalsByDay, targets, recomputedAt); err != nil {
		return domain.ScopedTokenUsageReport{}, err
	}

	return buildScopedTokenUsageReport(resolved.scope, input.FromDate, input.ToDate, recomputedAt, totalsByDay), nil
}

func (r *EntRepository) resolveScopedTokenUsage(
	ctx context.Context,
	scope domain.TokenUsageScope,
) (resolvedTokenUsageScope, error) {
	switch scope.Kind {
	case domain.TokenUsageScopeKindOrganization:
		organization, err := r.getActiveOrganization(ctx, scope.ID)
		if err != nil {
			return resolvedTokenUsageScope{}, err
		}
		projectIDs, err := organizationProjectIDs(ctx, r.client, organization.ID)
		if err != nil {
			return resolvedTokenUsageScope{}, err
		}
		return resolvedTokenUsageScope{
			scope:          scope,
			organizationID: organization.ID,
			projectIDs:     projectIDs,
		}, nil
	case domain.TokenUsageScopeKindProject:
		project, err := r.GetProject(ctx, scope.ID)
		if err != nil {
			return resolvedTokenUsageScope{}, err
		}
		return resolvedTokenUsageScope{
			scope:          scope,
			organizationID: project.OrganizationID,
			projectIDs:     []uuid.UUID{project.ID},
		}, nil
	default:
		return resolvedTokenUsageScope{}, fmt.Errorf("unsupported token usage scope kind %q", scope.Kind)
	}
}

func (r *EntRepository) buildRunUsageTotalsByProjectScope(
	ctx context.Context,
	projectIDs []uuid.UUID,
	fromDate time.Time,
	toDate time.Time,
) (map[time.Time]runDailyUsageTotals, scopedMaterializationTargets, error) {
	totalsByDay := make(map[time.Time]runDailyUsageTotals)
	if len(projectIDs) == 0 {
		return totalsByDay, scopedMaterializationTargets{}, nil
	}

	start := startOfUTCDay(fromDate)
	endExclusive := startOfUTCDay(toDate).AddDate(0, 0, 1)

	agentRuns, err := r.client.AgentRun.Query().
		Where(
			entagentrun.StatusIn(
				entagentrun.StatusCompleted,
				entagentrun.StatusErrored,
				entagentrun.StatusTerminated,
			),
			entagentrun.TerminalAtGTE(start),
			entagentrun.TerminalAtLT(endExclusive),
			entagentrun.HasTicketWith(entticket.ProjectIDIn(projectIDs...)),
		).
		All(ctx)
	if err != nil {
		return nil, scopedMaterializationTargets{}, fmt.Errorf("list agent runs for token usage: %w", err)
	}

	targets := scopedMaterializationTargets{
		agentRunIDs: make([]uuid.UUID, 0, len(agentRuns)),
	}
	for _, runItem := range agentRuns {
		if runItem == nil || runItem.TerminalAt == nil {
			continue
		}
		targets.agentRunIDs = append(targets.agentRunIDs, runItem.ID)
		day := startOfUTCDay(runItem.TerminalAt.UTC())
		totalsByDay[day] = addRunDailyUsageTotals(totalsByDay[day], runDailyUsageTotalsFromRun(runItem))
	}

	conversationRuns, err := r.client.ProjectConversationRun.Query().
		Where(
			entprojectconversationrun.StatusIn(
				entprojectconversationrun.StatusCompleted,
				entprojectconversationrun.StatusFailed,
				entprojectconversationrun.StatusTerminated,
			),
			entprojectconversationrun.TerminalAtGTE(start),
			entprojectconversationrun.TerminalAtLT(endExclusive),
			entprojectconversationrun.ProjectIDIn(projectIDs...),
		).
		All(ctx)
	if err != nil {
		return nil, scopedMaterializationTargets{}, fmt.Errorf("list project conversation runs for token usage: %w", err)
	}

	targets.conversationRunIDs = make([]uuid.UUID, 0, len(conversationRuns))
	for _, runItem := range conversationRuns {
		if runItem == nil || runItem.TerminalAt == nil {
			continue
		}
		targets.conversationRunIDs = append(targets.conversationRunIDs, runItem.ID)
		day := startOfUTCDay(runItem.TerminalAt.UTC())
		totalsByDay[day] = addRunDailyUsageTotals(totalsByDay[day], runDailyUsageTotalsFromConversationRun(runItem))
	}

	return totalsByDay, targets, nil
}

func (r *EntRepository) replaceScopedTokenUsageRange(
	ctx context.Context,
	scope domain.TokenUsageScope,
	fromDate time.Time,
	toDate time.Time,
	totalsByDay map[time.Time]runDailyUsageTotals,
	targets scopedMaterializationTargets,
	recomputedAt time.Time,
) (err error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start %s token usage replace tx: %w", scope.Kind, err)
	}
	defer rollbackOnError(ctx, tx, &err)

	for cursor := startOfUTCDay(fromDate); !cursor.After(toDate); cursor = cursor.AddDate(0, 0, 1) {
		totals := totalsByDay[cursor]
		sourceMode := "lazy_backfill"
		if totals.runCount > 0 {
			sourceMode = "materialized"
		}
		if err := replaceScopedDailyUsageRow(ctx, tx, scope, cursor, totals, recomputedAt.UTC(), sourceMode); err != nil {
			return err
		}
	}

	if len(targets.agentRunIDs) > 0 {
		if _, err := tx.AgentRun.Update().
			Where(entagentrun.IDIn(targets.agentRunIDs...)).
			SetSnapshotMaterializedAt(recomputedAt.UTC()).
			Save(ctx); err != nil {
			return fmt.Errorf("mark agent runs materialized for %s token usage: %w", scope.Kind, err)
		}
	}
	if len(targets.conversationRunIDs) > 0 {
		if _, err := tx.ProjectConversationRun.Update().
			Where(entprojectconversationrun.IDIn(targets.conversationRunIDs...)).
			SetSnapshotMaterializedAt(recomputedAt.UTC()).
			Save(ctx); err != nil {
			return fmt.Errorf("mark project conversation runs materialized for %s token usage: %w", scope.Kind, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s token usage replace tx: %w", scope.Kind, err)
	}
	return nil
}

func replaceScopedDailyUsageRow(
	ctx context.Context,
	tx *ent.Tx,
	scope domain.TokenUsageScope,
	usageDate time.Time,
	totals runDailyUsageTotals,
	recomputedAt time.Time,
	sourceMode string,
) error {
	switch scope.Kind {
	case domain.TokenUsageScopeKindOrganization:
		mode := entorganizationdailytokenusage.SourceModeLazyBackfill
		if sourceMode == "materialized" {
			mode = entorganizationdailytokenusage.SourceModeMaterialized
		}
		updatedCount, err := tx.OrganizationDailyTokenUsage.Update().
			Where(
				entorganizationdailytokenusage.OrganizationIDEQ(scope.ID),
				entorganizationdailytokenusage.UsageDateEQ(startOfUTCDay(usageDate)),
			).
			SetInputTokens(totals.inputTokens).
			SetOutputTokens(totals.outputTokens).
			SetCachedInputTokens(totals.cachedInputTokens).
			SetReasoningTokens(totals.reasoningTokens).
			SetTotalTokens(totals.totalTokens).
			SetFinalizedRunCount(totals.runCount).
			SetRecomputedAt(recomputedAt.UTC()).
			SetSourceMode(mode).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("replace organization daily token usage: %w", err)
		}
		if updatedCount > 0 {
			return nil
		}

		builder := tx.OrganizationDailyTokenUsage.Create().
			SetOrganizationID(scope.ID).
			SetUsageDate(startOfUTCDay(usageDate)).
			SetInputTokens(totals.inputTokens).
			SetOutputTokens(totals.outputTokens).
			SetCachedInputTokens(totals.cachedInputTokens).
			SetReasoningTokens(totals.reasoningTokens).
			SetTotalTokens(totals.totalTokens).
			SetFinalizedRunCount(totals.runCount).
			SetRecomputedAt(recomputedAt.UTC()).
			SetSourceMode(mode)
		if _, err := builder.Save(ctx); err != nil {
			if ent.IsConstraintError(err) {
				return errScopedDailyUsageRetry
			}
			return fmt.Errorf("create replaced organization daily token usage: %w", err)
		}
		return nil
	case domain.TokenUsageScopeKindProject:
		mode := entprojectdailytokenusage.SourceModeLazyBackfill
		if sourceMode == "materialized" {
			mode = entprojectdailytokenusage.SourceModeMaterialized
		}
		updatedCount, err := tx.ProjectDailyTokenUsage.Update().
			Where(
				entprojectdailytokenusage.ProjectIDEQ(scope.ID),
				entprojectdailytokenusage.UsageDateEQ(startOfUTCDay(usageDate)),
			).
			SetInputTokens(totals.inputTokens).
			SetOutputTokens(totals.outputTokens).
			SetCachedInputTokens(totals.cachedInputTokens).
			SetReasoningTokens(totals.reasoningTokens).
			SetTotalTokens(totals.totalTokens).
			SetFinalizedRunCount(totals.runCount).
			SetRecomputedAt(recomputedAt.UTC()).
			SetSourceMode(mode).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("replace project daily token usage: %w", err)
		}
		if updatedCount > 0 {
			return nil
		}

		builder := tx.ProjectDailyTokenUsage.Create().
			SetProjectID(scope.ID).
			SetUsageDate(startOfUTCDay(usageDate)).
			SetInputTokens(totals.inputTokens).
			SetOutputTokens(totals.outputTokens).
			SetCachedInputTokens(totals.cachedInputTokens).
			SetReasoningTokens(totals.reasoningTokens).
			SetTotalTokens(totals.totalTokens).
			SetFinalizedRunCount(totals.runCount).
			SetRecomputedAt(recomputedAt.UTC()).
			SetSourceMode(mode)
		if _, err := builder.Save(ctx); err != nil {
			if ent.IsConstraintError(err) {
				return errScopedDailyUsageRetry
			}
			return fmt.Errorf("create replaced project daily token usage: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported token usage scope kind %q", scope.Kind)
	}
}

func applyScopedDailyUsageDelta(
	ctx context.Context,
	tx *ent.Tx,
	scope domain.TokenUsageScope,
	usageDate time.Time,
	totals runDailyUsageTotals,
	recomputedAt time.Time,
	sourceMode string,
) error {
	switch scope.Kind {
	case domain.TokenUsageScopeKindOrganization:
		update := tx.OrganizationDailyTokenUsage.Update().
			Where(
				entorganizationdailytokenusage.OrganizationIDEQ(scope.ID),
				entorganizationdailytokenusage.UsageDateEQ(startOfUTCDay(usageDate)),
			).
			AddInputTokens(totals.inputTokens).
			AddOutputTokens(totals.outputTokens).
			AddCachedInputTokens(totals.cachedInputTokens).
			AddReasoningTokens(totals.reasoningTokens).
			AddTotalTokens(totals.totalTokens).
			AddFinalizedRunCount(totals.runCount).
			SetRecomputedAt(recomputedAt)
		if sourceMode == "lazy_backfill" {
			update.SetSourceMode(entorganizationdailytokenusage.SourceModeLazyBackfill)
		} else {
			update.SetSourceMode(entorganizationdailytokenusage.SourceModeMaterialized)
		}
		updatedCount, err := update.Save(ctx)
		if err != nil {
			return fmt.Errorf("update organization daily token usage: %w", err)
		}
		if updatedCount > 0 {
			return nil
		}

		builder := tx.OrganizationDailyTokenUsage.Create().
			SetOrganizationID(scope.ID).
			SetUsageDate(startOfUTCDay(usageDate)).
			SetInputTokens(totals.inputTokens).
			SetOutputTokens(totals.outputTokens).
			SetCachedInputTokens(totals.cachedInputTokens).
			SetReasoningTokens(totals.reasoningTokens).
			SetTotalTokens(totals.totalTokens).
			SetFinalizedRunCount(totals.runCount).
			SetRecomputedAt(recomputedAt)
		if sourceMode == "lazy_backfill" {
			builder.SetSourceMode(entorganizationdailytokenusage.SourceModeLazyBackfill)
		} else {
			builder.SetSourceMode(entorganizationdailytokenusage.SourceModeMaterialized)
		}
		if _, err := builder.Save(ctx); err != nil {
			if ent.IsConstraintError(err) {
				return errScopedDailyUsageRetry
			}
			return fmt.Errorf("create organization daily token usage: %w", err)
		}
		return nil
	case domain.TokenUsageScopeKindProject:
		update := tx.ProjectDailyTokenUsage.Update().
			Where(
				entprojectdailytokenusage.ProjectIDEQ(scope.ID),
				entprojectdailytokenusage.UsageDateEQ(startOfUTCDay(usageDate)),
			).
			AddInputTokens(totals.inputTokens).
			AddOutputTokens(totals.outputTokens).
			AddCachedInputTokens(totals.cachedInputTokens).
			AddReasoningTokens(totals.reasoningTokens).
			AddTotalTokens(totals.totalTokens).
			AddFinalizedRunCount(totals.runCount).
			SetRecomputedAt(recomputedAt)
		if sourceMode == "lazy_backfill" {
			update.SetSourceMode(entprojectdailytokenusage.SourceModeLazyBackfill)
		} else {
			update.SetSourceMode(entprojectdailytokenusage.SourceModeMaterialized)
		}
		updatedCount, err := update.Save(ctx)
		if err != nil {
			return fmt.Errorf("update project daily token usage: %w", err)
		}
		if updatedCount > 0 {
			return nil
		}

		builder := tx.ProjectDailyTokenUsage.Create().
			SetProjectID(scope.ID).
			SetUsageDate(startOfUTCDay(usageDate)).
			SetInputTokens(totals.inputTokens).
			SetOutputTokens(totals.outputTokens).
			SetCachedInputTokens(totals.cachedInputTokens).
			SetReasoningTokens(totals.reasoningTokens).
			SetTotalTokens(totals.totalTokens).
			SetFinalizedRunCount(totals.runCount).
			SetRecomputedAt(recomputedAt)
		if sourceMode == "lazy_backfill" {
			builder.SetSourceMode(entprojectdailytokenusage.SourceModeLazyBackfill)
		} else {
			builder.SetSourceMode(entprojectdailytokenusage.SourceModeMaterialized)
		}
		if _, err := builder.Save(ctx); err != nil {
			if ent.IsConstraintError(err) {
				return errScopedDailyUsageRetry
			}
			return fmt.Errorf("create project daily token usage: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported token usage scope kind %q", scope.Kind)
	}
}

func buildScopedTokenUsageReport(
	scope domain.TokenUsageScope,
	fromDate time.Time,
	toDate time.Time,
	recomputedAt time.Time,
	totalsByDay map[time.Time]runDailyUsageTotals,
) domain.ScopedTokenUsageReport {
	report := domain.ScopedTokenUsageReport{
		Scope:    scope,
		FromDate: fromDate.UTC(),
		ToDate:   toDate.UTC(),
		Days:     make([]domain.ScopedDailyTokenUsage, 0, inclusiveUTCDateCount(fromDate, toDate)),
	}

	appendScopedTokenUsageDays(&report.Summary.TotalTokens, &report.Summary.AvgDailyTokens, &report.Summary.PeakDay, &report.Days, fromDate, toDate, recomputedAt, totalsByDay)
	return report
}

func appendScopedTokenUsageDays(
	totalTokens *int64,
	avgDailyTokens *int64,
	peakDay **domain.ScopedTokenUsagePeakDay,
	days *[]domain.ScopedDailyTokenUsage,
	fromDate time.Time,
	toDate time.Time,
	recomputedAt time.Time,
	totalsByDay map[time.Time]runDailyUsageTotals,
) {
	dayCount := 0
	for cursor := startOfUTCDay(fromDate); !cursor.After(toDate); cursor = cursor.AddDate(0, 0, 1) {
		dayCount++
		totals := totalsByDay[cursor]
		sourceMode := "lazy_backfill"
		if totals.runCount > 0 {
			sourceMode = "materialized"
		}
		row := domain.ScopedDailyTokenUsage{
			UsageDate:         cursor,
			InputTokens:       totals.inputTokens,
			OutputTokens:      totals.outputTokens,
			CachedInputTokens: totals.cachedInputTokens,
			ReasoningTokens:   totals.reasoningTokens,
			TotalTokens:       totals.totalTokens,
			FinalizedRunCount: totals.runCount,
			RecomputedAt:      recomputedAt.UTC(),
			SourceMode:        sourceMode,
		}
		*days = append(*days, row)
		*totalTokens += row.TotalTokens
		if *peakDay == nil || row.TotalTokens > (*peakDay).TotalTokens {
			*peakDay = &domain.ScopedTokenUsagePeakDay{
				Date:        row.UsageDate,
				TotalTokens: row.TotalTokens,
			}
		}
	}
	if dayCount > 0 {
		*avgDailyTokens = *totalTokens / int64(dayCount)
	}
}

func runDailyUsageTotalsFromRun(item *ent.AgentRun) runDailyUsageTotals {
	if item == nil {
		return runDailyUsageTotals{}
	}

	return runDailyUsageTotals{
		inputTokens:       item.InputTokens,
		outputTokens:      item.OutputTokens,
		cachedInputTokens: item.CachedInputTokens,
		reasoningTokens:   item.ReasoningTokens,
		totalTokens:       item.TotalTokens,
		runCount:          1,
	}
}

func runDailyUsageTotalsFromConversationRun(item *ent.ProjectConversationRun) runDailyUsageTotals {
	if item == nil {
		return runDailyUsageTotals{}
	}

	return runDailyUsageTotals{
		inputTokens:       item.InputTokens,
		outputTokens:      item.OutputTokens,
		cachedInputTokens: item.CachedInputTokens,
		reasoningTokens:   item.ReasoningTokens,
		totalTokens:       item.TotalTokens,
		runCount:          1,
	}
}

func addRunDailyUsageTotals(base runDailyUsageTotals, extra runDailyUsageTotals) runDailyUsageTotals {
	return runDailyUsageTotals{
		inputTokens:       base.inputTokens + extra.inputTokens,
		outputTokens:      base.outputTokens + extra.outputTokens,
		cachedInputTokens: base.cachedInputTokens + extra.cachedInputTokens,
		reasoningTokens:   base.reasoningTokens + extra.reasoningTokens,
		totalTokens:       base.totalTokens + extra.totalTokens,
		runCount:          base.runCount + extra.runCount,
	}
}

func inclusiveUTCDateCount(fromDate time.Time, toDate time.Time) int {
	count := 0
	for cursor := startOfUTCDay(fromDate); !cursor.After(toDate); cursor = cursor.AddDate(0, 0, 1) {
		count++
	}
	return count
}

func isTerminalRunStatus(status entagentrun.Status) bool {
	switch status {
	case entagentrun.StatusCompleted, entagentrun.StatusErrored, entagentrun.StatusTerminated:
		return true
	default:
		return false
	}
}

func isTerminalProjectConversationRunStatus(status entprojectconversationrun.Status) bool {
	switch status {
	case entprojectconversationrun.StatusCompleted, entprojectconversationrun.StatusFailed, entprojectconversationrun.StatusTerminated:
		return true
	default:
		return false
	}
}

func MaterializeAgentRunDailyUsage(
	ctx context.Context,
	client *ent.Client,
	runID uuid.UUID,
	recomputedAt time.Time,
) error {
	if client == nil || runID == uuid.Nil {
		return nil
	}

	for attempt := 0; attempt < 2; attempt++ {
		err := materializeAgentRunDailyUsageOnce(ctx, client, runID, recomputedAt.UTC())
		if errors.Is(err, errScopedDailyUsageRetry) {
			continue
		}
		return err
	}

	return fmt.Errorf("materialize agent run %s daily usage: %w", runID, errScopedDailyUsageRetry)
}

func materializeAgentRunDailyUsageOnce(
	ctx context.Context,
	client *ent.Client,
	runID uuid.UUID,
	recomputedAt time.Time,
) (err error) {
	tx, err := client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start agent run daily usage tx: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	runItem, err := tx.AgentRun.Query().
		Where(entagentrun.IDEQ(runID)).
		WithTicket(func(query *ent.TicketQuery) {
			query.WithProject()
		}).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("load agent run %s for daily usage: %w", runID, err)
	}
	if !isTerminalRunStatus(runItem.Status) || runItem.TerminalAt == nil || runItem.SnapshotMaterializedAt != nil {
		_ = tx.Rollback()
		return nil
	}
	if runItem.Edges.Ticket == nil || runItem.Edges.Ticket.Edges.Project == nil {
		return fmt.Errorf("load ticket project for run %s daily usage", runID)
	}

	markedCount, err := tx.AgentRun.Update().
		Where(
			entagentrun.IDEQ(runID),
			entagentrun.SnapshotMaterializedAtIsNil(),
		).
		SetSnapshotMaterializedAt(recomputedAt).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("mark run %s daily usage materialized: %w", runID, err)
	}
	if markedCount == 0 {
		_ = tx.Rollback()
		return nil
	}

	usageDate := startOfUTCDay(runItem.TerminalAt.UTC())
	totals := runDailyUsageTotalsFromRun(runItem)
	projectID := runItem.Edges.Ticket.ProjectID
	organizationID := runItem.Edges.Ticket.Edges.Project.OrganizationID

	if err := applyScopedDailyUsageDelta(ctx, tx, domain.NewOrganizationTokenUsageScope(organizationID), usageDate, totals, recomputedAt, "materialized"); err != nil {
		return err
	}
	if err := applyScopedDailyUsageDelta(ctx, tx, domain.NewProjectTokenUsageScope(projectID), usageDate, totals, recomputedAt, "materialized"); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit agent run daily usage %s: %w", runID, err)
	}

	return nil
}

func MaterializeProjectConversationRunDailyUsage(
	ctx context.Context,
	client *ent.Client,
	runID uuid.UUID,
	recomputedAt time.Time,
) error {
	if client == nil || runID == uuid.Nil {
		return nil
	}

	for attempt := 0; attempt < 2; attempt++ {
		err := materializeProjectConversationRunDailyUsageOnce(ctx, client, runID, recomputedAt.UTC())
		if errors.Is(err, errScopedDailyUsageRetry) {
			continue
		}
		return err
	}

	return fmt.Errorf("materialize project conversation run %s daily usage: %w", runID, errScopedDailyUsageRetry)
}

func materializeProjectConversationRunDailyUsageOnce(
	ctx context.Context,
	client *ent.Client,
	runID uuid.UUID,
	recomputedAt time.Time,
) (err error) {
	tx, err := client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start project conversation run daily usage tx: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	runItem, err := tx.ProjectConversationRun.Get(ctx, runID)
	if err != nil {
		return fmt.Errorf("load project conversation run %s for daily usage: %w", runID, err)
	}
	if !isTerminalProjectConversationRunStatus(runItem.Status) || runItem.TerminalAt == nil || runItem.SnapshotMaterializedAt != nil {
		_ = tx.Rollback()
		return nil
	}

	projectItem, err := tx.Project.Get(ctx, runItem.ProjectID)
	if err != nil {
		return fmt.Errorf("load project %s for project conversation run %s daily usage: %w", runItem.ProjectID, runID, err)
	}

	markedCount, err := tx.ProjectConversationRun.Update().
		Where(
			entprojectconversationrun.IDEQ(runID),
			entprojectconversationrun.SnapshotMaterializedAtIsNil(),
		).
		SetSnapshotMaterializedAt(recomputedAt).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("mark project conversation run %s daily usage materialized: %w", runID, err)
	}
	if markedCount == 0 {
		_ = tx.Rollback()
		return nil
	}

	usageDate := startOfUTCDay(runItem.TerminalAt.UTC())
	totals := runDailyUsageTotalsFromConversationRun(runItem)

	if err := applyScopedDailyUsageDelta(ctx, tx, domain.NewOrganizationTokenUsageScope(projectItem.OrganizationID), usageDate, totals, recomputedAt, "materialized"); err != nil {
		return err
	}
	if err := applyScopedDailyUsageDelta(ctx, tx, domain.NewProjectTokenUsageScope(runItem.ProjectID), usageDate, totals, recomputedAt, "materialized"); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit project conversation run daily usage %s: %w", runID, err)
	}

	return nil
}
