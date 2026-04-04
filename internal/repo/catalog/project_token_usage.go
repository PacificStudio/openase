package catalog

import (
	"context"
	"fmt"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entorganizationdailytokenusage "github.com/BetterAndBetterII/openase/ent/organizationdailytokenusage"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entprojectconversationrun "github.com/BetterAndBetterII/openase/ent/projectconversationrun"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

func (r *EntRepository) GetProjectTokenUsage(
	ctx context.Context,
	input domain.GetProjectTokenUsage,
) (domain.ProjectTokenUsageReport, error) {
	project, err := r.GetProject(ctx, input.ProjectID)
	if err != nil {
		return domain.ProjectTokenUsageReport{}, err
	}

	recomputedAt := time.Now().UTC()
	totalsByDay, _, err := r.buildRunUsageTotalsByProjectScope(ctx, []uuid.UUID{project.ID}, input.FromDate, input.ToDate)
	if err != nil {
		return domain.ProjectTokenUsageReport{}, err
	}

	return buildProjectTokenUsageReport(project.ID, input.FromDate, input.ToDate, recomputedAt, totalsByDay), nil
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

func (r *EntRepository) buildRunUsageTotalsByProjectScope(
	ctx context.Context,
	projectIDs []uuid.UUID,
	fromDate time.Time,
	toDate time.Time,
) (map[time.Time]runDailyUsageTotals, []uuid.UUID, error) {
	totalsByDay := make(map[time.Time]runDailyUsageTotals)
	if len(projectIDs) == 0 {
		return totalsByDay, nil, nil
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
		return nil, nil, fmt.Errorf("list agent runs for token usage: %w", err)
	}

	agentRunIDs := make([]uuid.UUID, 0, len(agentRuns))
	for _, runItem := range agentRuns {
		if runItem == nil || runItem.TerminalAt == nil {
			continue
		}
		agentRunIDs = append(agentRunIDs, runItem.ID)
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
		return nil, nil, fmt.Errorf("list project conversation runs for token usage: %w", err)
	}

	for _, runItem := range conversationRuns {
		if runItem == nil || runItem.TerminalAt == nil {
			continue
		}
		day := startOfUTCDay(runItem.TerminalAt.UTC())
		totalsByDay[day] = addRunDailyUsageTotals(totalsByDay[day], runDailyUsageTotalsFromConversationRun(runItem))
	}

	return totalsByDay, agentRunIDs, nil
}

func (r *EntRepository) replaceOrganizationTokenUsageRange(
	ctx context.Context,
	organizationID uuid.UUID,
	fromDate time.Time,
	toDate time.Time,
	totalsByDay map[time.Time]runDailyUsageTotals,
	agentRunIDs []uuid.UUID,
	recomputedAt time.Time,
) (err error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start organization token usage replace tx: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	for cursor := startOfUTCDay(fromDate); !cursor.After(toDate); cursor = cursor.AddDate(0, 0, 1) {
		totals := totalsByDay[cursor]
		sourceMode := entorganizationdailytokenusage.SourceModeLazyBackfill
		if totals.runCount > 0 {
			sourceMode = entorganizationdailytokenusage.SourceModeMaterialized
		}
		if err := replaceOrganizationDailyUsageRow(ctx, tx, organizationID, cursor, totals, recomputedAt.UTC(), sourceMode); err != nil {
			return err
		}
	}

	if len(agentRunIDs) > 0 {
		if _, err := tx.AgentRun.Update().
			Where(entagentrun.IDIn(agentRunIDs...)).
			SetSnapshotMaterializedAt(recomputedAt.UTC()).
			Save(ctx); err != nil {
			return fmt.Errorf("mark agent runs materialized for organization token usage: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit organization token usage replace tx: %w", err)
	}
	return nil
}

func replaceOrganizationDailyUsageRow(
	ctx context.Context,
	tx *ent.Tx,
	organizationID uuid.UUID,
	usageDate time.Time,
	totals runDailyUsageTotals,
	recomputedAt time.Time,
	sourceMode entorganizationdailytokenusage.SourceMode,
) error {
	updatedCount, err := tx.OrganizationDailyTokenUsage.Update().
		Where(
			entorganizationdailytokenusage.OrganizationIDEQ(organizationID),
			entorganizationdailytokenusage.UsageDateEQ(startOfUTCDay(usageDate)),
		).
		SetInputTokens(totals.inputTokens).
		SetOutputTokens(totals.outputTokens).
		SetCachedInputTokens(totals.cachedInputTokens).
		SetReasoningTokens(totals.reasoningTokens).
		SetTotalTokens(totals.totalTokens).
		SetFinalizedRunCount(totals.runCount).
		SetRecomputedAt(recomputedAt.UTC()).
		SetSourceMode(sourceMode).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("replace organization daily token usage: %w", err)
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
		SetRecomputedAt(recomputedAt.UTC()).
		SetSourceMode(sourceMode)
	if _, err := builder.Save(ctx); err != nil {
		if ent.IsConstraintError(err) {
			return errOrganizationDailyUsageRetry
		}
		return fmt.Errorf("create replaced organization daily token usage: %w", err)
	}
	return nil
}

func buildOrganizationTokenUsageReport(
	organizationID uuid.UUID,
	fromDate time.Time,
	toDate time.Time,
	recomputedAt time.Time,
	totalsByDay map[time.Time]runDailyUsageTotals,
) domain.OrganizationTokenUsageReport {
	report := domain.OrganizationTokenUsageReport{
		OrganizationID: organizationID,
		FromDate:       fromDate.UTC(),
		ToDate:         toDate.UTC(),
		Days:           make([]domain.OrganizationDailyTokenUsage, 0, inclusiveUTCDateCount(fromDate, toDate)),
	}

	appendTokenUsageDays(&report.Summary.TotalTokens, &report.Summary.AvgDailyTokens, &report.Summary.PeakDay, &report.Days, fromDate, toDate, recomputedAt, totalsByDay)
	return report
}

func buildProjectTokenUsageReport(
	projectID uuid.UUID,
	fromDate time.Time,
	toDate time.Time,
	recomputedAt time.Time,
	totalsByDay map[time.Time]runDailyUsageTotals,
) domain.ProjectTokenUsageReport {
	report := domain.ProjectTokenUsageReport{
		ProjectID: projectID,
		FromDate:  fromDate.UTC(),
		ToDate:    toDate.UTC(),
		Days:      make([]domain.ProjectDailyTokenUsage, 0, inclusiveUTCDateCount(fromDate, toDate)),
	}

	appendProjectTokenUsageDays(&report.Summary.TotalTokens, &report.Summary.AvgDailyTokens, &report.Summary.PeakDay, &report.Days, fromDate, toDate, recomputedAt, totalsByDay)
	return report
}

func appendTokenUsageDays(
	totalTokens *int64,
	avgDailyTokens *int64,
	peakDay **domain.OrganizationTokenUsagePeakDay,
	days *[]domain.OrganizationDailyTokenUsage,
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
		row := domain.OrganizationDailyTokenUsage{
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
			*peakDay = &domain.OrganizationTokenUsagePeakDay{
				Date:        row.UsageDate,
				TotalTokens: row.TotalTokens,
			}
		}
	}
	if dayCount > 0 {
		*avgDailyTokens = *totalTokens / int64(dayCount)
	}
}

func appendProjectTokenUsageDays(
	totalTokens *int64,
	avgDailyTokens *int64,
	peakDay **domain.ProjectTokenUsagePeakDay,
	days *[]domain.ProjectDailyTokenUsage,
	fromDate time.Time,
	toDate time.Time,
	recomputedAt time.Time,
	totalsByDay map[time.Time]runDailyUsageTotals,
) {
	projectDays := make([]domain.OrganizationDailyTokenUsage, 0, inclusiveUTCDateCount(fromDate, toDate))
	orgPeak := (*domain.OrganizationTokenUsagePeakDay)(nil)
	appendTokenUsageDays(totalTokens, avgDailyTokens, &orgPeak, &projectDays, fromDate, toDate, recomputedAt, totalsByDay)
	*days = append(*days, projectDays...)
	if orgPeak != nil {
		*peakDay = (*domain.ProjectTokenUsagePeakDay)(orgPeak)
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
