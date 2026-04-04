package catalog

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entorganizationdailytokenusage "github.com/BetterAndBetterII/openase/ent/organizationdailytokenusage"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

var errOrganizationDailyUsageRetry = errors.New("retry organization daily usage materialization")

type runDailyUsageTotals struct {
	inputTokens       int64
	outputTokens      int64
	cachedInputTokens int64
	reasoningTokens   int64
	totalTokens       int64
	runCount          int
}

func (r *EntRepository) GetOrganizationTokenUsage(
	ctx context.Context,
	input domain.GetOrganizationTokenUsage,
) (domain.OrganizationTokenUsageReport, error) {
	organization, err := r.getActiveOrganization(ctx, input.OrganizationID)
	if err != nil {
		return domain.OrganizationTokenUsageReport{}, err
	}

	recomputedAt := time.Now().UTC()
	projectIDs, err := organizationProjectIDs(ctx, r.client, organization.ID)
	if err != nil {
		return domain.OrganizationTokenUsageReport{}, err
	}
	totalsByDay, agentRunIDs, err := r.buildRunUsageTotalsByProjectScope(ctx, projectIDs, input.FromDate, input.ToDate)
	if err != nil {
		return domain.OrganizationTokenUsageReport{}, err
	}
	if err := r.replaceOrganizationTokenUsageRange(
		ctx,
		organization.ID,
		input.FromDate,
		input.ToDate,
		totalsByDay,
		agentRunIDs,
		recomputedAt,
	); err != nil {
		return domain.OrganizationTokenUsageReport{}, err
	}

	return buildOrganizationTokenUsageReport(organization.ID, input.FromDate, input.ToDate, recomputedAt, totalsByDay), nil
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
		if errors.Is(err, errOrganizationDailyUsageRetry) {
			continue
		}
		return err
	}

	return fmt.Errorf("materialize agent run %s daily usage: %w", runID, errOrganizationDailyUsageRetry)
}

func (r *EntRepository) materializeUnrecordedRunsInRange(
	ctx context.Context,
	organizationID uuid.UUID,
	fromDate time.Time,
	toDate time.Time,
	recomputedAt time.Time,
) error {
	runIDs, err := r.client.AgentRun.Query().
		Where(
			entagentrun.StatusIn(
				entagentrun.StatusCompleted,
				entagentrun.StatusErrored,
				entagentrun.StatusTerminated,
			),
			entagentrun.TerminalAtGTE(startOfUTCDay(fromDate)),
			entagentrun.TerminalAtLT(startOfUTCDay(toDate).AddDate(0, 0, 1)),
			entagentrun.SnapshotMaterializedAtIsNil(),
			entagentrun.HasTicketWith(
				entticket.HasProjectWith(
					entproject.OrganizationIDEQ(organizationID),
				),
			),
		).
		IDs(ctx)
	if err != nil {
		return fmt.Errorf("list unmaterialized runs for org token usage: %w", err)
	}

	for _, runID := range runIDs {
		if err := MaterializeAgentRunDailyUsage(ctx, r.client, runID, recomputedAt); err != nil {
			return err
		}
	}

	return nil
}

func (r *EntRepository) backfillMissingOrganizationTokenUsageDays(
	ctx context.Context,
	organizationID uuid.UUID,
	fromDate time.Time,
	toDate time.Time,
	recomputedAt time.Time,
) error {
	rows, err := r.listOrganizationTokenUsageRows(ctx, organizationID, fromDate, toDate)
	if err != nil {
		return err
	}
	existingDates := make([]time.Time, 0, len(rows))
	for _, row := range rows {
		existingDates = append(existingDates, row.UsageDate)
	}

	missingDates := missingUTCDateRange(fromDate, toDate, existingDates)
	if len(missingDates) == 0 {
		return nil
	}

	runItems, err := r.client.AgentRun.Query().
		Where(
			entagentrun.StatusIn(
				entagentrun.StatusCompleted,
				entagentrun.StatusErrored,
				entagentrun.StatusTerminated,
			),
			entagentrun.TerminalAtGTE(startOfUTCDay(fromDate)),
			entagentrun.TerminalAtLT(startOfUTCDay(toDate).AddDate(0, 0, 1)),
			entagentrun.HasTicketWith(
				entticket.HasProjectWith(
					entproject.OrganizationIDEQ(organizationID),
				),
			),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list terminal runs for org token usage backfill: %w", err)
	}

	totalsByDay := make(map[time.Time]runDailyUsageTotals, len(missingDates))
	for _, runItem := range runItems {
		if runItem == nil || runItem.TerminalAt == nil {
			continue
		}
		day := startOfUTCDay(runItem.TerminalAt.UTC())
		current := totalsByDay[day]
		totalsByDay[day] = addRunDailyUsageTotals(current, runDailyUsageTotalsFromRun(runItem))
	}

	for _, missingDate := range missingDates {
		totals := totalsByDay[missingDate]
		if err := createBackfillOrganizationUsageRow(
			ctx,
			r.client,
			organizationID,
			missingDate,
			totals,
			recomputedAt.UTC(),
		); err != nil {
			return err
		}
	}

	return nil
}

func (r *EntRepository) listOrganizationTokenUsageRows(
	ctx context.Context,
	organizationID uuid.UUID,
	fromDate time.Time,
	toDate time.Time,
) ([]domain.OrganizationDailyTokenUsage, error) {
	items, err := r.client.OrganizationDailyTokenUsage.Query().
		Where(
			entorganizationdailytokenusage.OrganizationIDEQ(organizationID),
			entorganizationdailytokenusage.UsageDateGTE(startOfUTCDay(fromDate)),
			entorganizationdailytokenusage.UsageDateLTE(startOfUTCDay(toDate)),
		).
		Order(entorganizationdailytokenusage.ByUsageDate()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organization daily token usage: %w", err)
	}

	rows := make([]domain.OrganizationDailyTokenUsage, 0, len(items))
	for _, item := range items {
		rows = append(rows, domain.OrganizationDailyTokenUsage{
			UsageDate:         item.UsageDate.UTC(),
			InputTokens:       item.InputTokens,
			OutputTokens:      item.OutputTokens,
			CachedInputTokens: item.CachedInputTokens,
			ReasoningTokens:   item.ReasoningTokens,
			TotalTokens:       item.TotalTokens,
			FinalizedRunCount: item.FinalizedRunCount,
			RecomputedAt:      item.RecomputedAt.UTC(),
			SourceMode:        item.SourceMode.String(),
		})
	}

	return rows, nil
}

func applyOrganizationDailyUsageDelta(
	ctx context.Context,
	tx *ent.Tx,
	organizationID uuid.UUID,
	usageDate time.Time,
	totals runDailyUsageTotals,
	recomputedAt time.Time,
	sourceMode string,
) error {
	update := tx.OrganizationDailyTokenUsage.Update().
		Where(
			entorganizationdailytokenusage.OrganizationIDEQ(organizationID),
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
		SetOrganizationID(organizationID).
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
			return errOrganizationDailyUsageRetry
		}
		return fmt.Errorf("create organization daily token usage: %w", err)
	}

	return nil
}

func createBackfillOrganizationUsageRow(
	ctx context.Context,
	client *ent.Client,
	organizationID uuid.UUID,
	usageDate time.Time,
	totals runDailyUsageTotals,
	recomputedAt time.Time,
) error {
	builder := client.OrganizationDailyTokenUsage.Create().
		SetOrganizationID(organizationID).
		SetUsageDate(startOfUTCDay(usageDate)).
		SetInputTokens(totals.inputTokens).
		SetOutputTokens(totals.outputTokens).
		SetCachedInputTokens(totals.cachedInputTokens).
		SetReasoningTokens(totals.reasoningTokens).
		SetTotalTokens(totals.totalTokens).
		SetFinalizedRunCount(totals.runCount).
		SetRecomputedAt(recomputedAt).
		SetSourceMode(entorganizationdailytokenusage.SourceModeLazyBackfill)
	if _, err := builder.Save(ctx); err != nil && !ent.IsConstraintError(err) {
		return fmt.Errorf("create lazy backfill organization token usage: %w", err)
	}

	return nil
}

func missingUTCDateRange(fromDate time.Time, toDate time.Time, existing []time.Time) []time.Time {
	normalizedExisting := make([]time.Time, 0, len(existing))
	for _, item := range existing {
		normalizedExisting = append(normalizedExisting, startOfUTCDay(item))
	}

	missing := make([]time.Time, 0)
	for cursor := startOfUTCDay(fromDate); !cursor.After(toDate); cursor = cursor.AddDate(0, 0, 1) {
		if slices.Contains(normalizedExisting, cursor) {
			continue
		}
		missing = append(missing, cursor)
	}

	return missing
}

func inclusiveUTCDateCount(fromDate time.Time, toDate time.Time) int {
	count := 0
	for cursor := startOfUTCDay(fromDate); !cursor.After(toDate); cursor = cursor.AddDate(0, 0, 1) {
		count++
	}
	return count
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

func isTerminalRunStatus(status entagentrun.Status) bool {
	switch status {
	case entagentrun.StatusCompleted, entagentrun.StatusErrored, entagentrun.StatusTerminated:
		return true
	default:
		return false
	}
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

	if err := applyOrganizationDailyUsageDelta(
		ctx,
		tx,
		runItem.Edges.Ticket.Edges.Project.OrganizationID,
		startOfUTCDay(runItem.TerminalAt.UTC()),
		runDailyUsageTotalsFromRun(runItem),
		recomputedAt,
		"materialized",
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit agent run daily usage %s: %w", runID, err)
	}

	return nil
}
