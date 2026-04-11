package catalog

import (
	"context"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

func (r *EntRepository) GetOrganizationTokenUsage(
	ctx context.Context,
	input domain.GetOrganizationTokenUsage,
) (domain.OrganizationTokenUsageReport, error) {
	report, err := r.getScopedTokenUsage(ctx, domain.GetScopedTokenUsage{
		Scope:    input.Scope(),
		FromDate: input.FromDate,
		ToDate:   input.ToDate,
	})
	if err != nil {
		return domain.OrganizationTokenUsageReport{}, err
	}

	return domain.OrganizationTokenUsageReport{
		OrganizationID: input.OrganizationID,
		FromDate:       report.FromDate,
		ToDate:         report.ToDate,
		Days:           report.Days,
		Summary:        report.Summary,
	}, nil
}
