package catalog

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const DefaultOrganizationTokenUsageWindowDays = 30

type OrganizationTokenUsageListInput struct {
	From string
	To   string
}

type GetOrganizationTokenUsage struct {
	OrganizationID uuid.UUID
	FromDate       time.Time
	ToDate         time.Time
}

type OrganizationDailyTokenUsage struct {
	UsageDate         time.Time
	InputTokens       int64
	OutputTokens      int64
	CachedInputTokens int64
	ReasoningTokens   int64
	TotalTokens       int64
	FinalizedRunCount int
	RecomputedAt      time.Time
	SourceMode        string
}

type OrganizationTokenUsagePeakDay struct {
	Date        time.Time
	TotalTokens int64
}

type OrganizationTokenUsageSummary struct {
	TotalTokens    int64
	AvgDailyTokens int64
	PeakDay        *OrganizationTokenUsagePeakDay
}

type OrganizationTokenUsageReport struct {
	OrganizationID uuid.UUID
	FromDate       time.Time
	ToDate         time.Time
	Days           []OrganizationDailyTokenUsage
	Summary        OrganizationTokenUsageSummary
}

func ParseOrganizationTokenUsage(
	organizationID uuid.UUID,
	raw OrganizationTokenUsageListInput,
	now time.Time,
) (GetOrganizationTokenUsage, error) {
	fromRaw := strings.TrimSpace(raw.From)
	toRaw := strings.TrimSpace(raw.To)
	if fromRaw == "" && toRaw == "" {
		end := startOfUTCDate(now.UTC())
		start := end.AddDate(0, 0, -(DefaultOrganizationTokenUsageWindowDays - 1))
		return GetOrganizationTokenUsage{
			OrganizationID: organizationID,
			FromDate:       start,
			ToDate:         end,
		}, nil
	}
	if fromRaw == "" || toRaw == "" {
		return GetOrganizationTokenUsage{}, fmt.Errorf("from and to must both be provided in YYYY-MM-DD format")
	}

	fromDate, err := parseUTCDate("from", fromRaw)
	if err != nil {
		return GetOrganizationTokenUsage{}, err
	}
	toDate, err := parseUTCDate("to", toRaw)
	if err != nil {
		return GetOrganizationTokenUsage{}, err
	}
	if toDate.Before(fromDate) {
		return GetOrganizationTokenUsage{}, fmt.Errorf("to must be greater than or equal to from")
	}

	return GetOrganizationTokenUsage{
		OrganizationID: organizationID,
		FromDate:       fromDate,
		ToDate:         toDate,
	}, nil
}

func parseUTCDate(fieldName string, raw string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be a valid date in YYYY-MM-DD format", fieldName)
	}
	return startOfUTCDate(parsed.UTC()), nil
}

func startOfUTCDate(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
