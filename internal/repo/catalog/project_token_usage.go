package catalog

import (
	"context"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

func (r *EntRepository) GetProjectTokenUsage(
	ctx context.Context,
	input domain.GetProjectTokenUsage,
) (domain.ProjectTokenUsageReport, error) {
	report, err := r.getScopedTokenUsage(ctx, domain.GetScopedTokenUsage{
		Scope:    input.Scope(),
		FromDate: input.FromDate,
		ToDate:   input.ToDate,
	})
	if err != nil {
		return domain.ProjectTokenUsageReport{}, err
	}

	return domain.ProjectTokenUsageReport{
		ProjectID: input.ProjectID,
		FromDate:  report.FromDate,
		ToDate:    report.ToDate,
		Days:      report.Days,
		Summary:   report.Summary,
	}, nil
}
