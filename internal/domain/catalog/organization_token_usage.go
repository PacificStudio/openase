package catalog

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const DefaultOrganizationTokenUsageWindowDays = 30

type TokenUsageScopeKind string

const (
	TokenUsageScopeKindOrganization TokenUsageScopeKind = "organization"
	TokenUsageScopeKindProject      TokenUsageScopeKind = "project"
)

type TokenUsageScope struct {
	Kind TokenUsageScopeKind
	ID   uuid.UUID
}

type OrganizationTokenUsageListInput struct {
	From string
	To   string
}

type ProjectTokenUsageListInput struct {
	From string
	To   string
}

type ScopedTokenUsageListInput struct {
	From string
	To   string
}

type GetOrganizationTokenUsage struct {
	OrganizationID uuid.UUID
	FromDate       time.Time
	ToDate         time.Time
}

type GetScopedTokenUsage struct {
	Scope    TokenUsageScope
	FromDate time.Time
	ToDate   time.Time
}

type ScopedDailyTokenUsage struct {
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

type ScopedTokenUsagePeakDay struct {
	Date        time.Time
	TotalTokens int64
}

type ScopedTokenUsageSummary struct {
	TotalTokens    int64
	AvgDailyTokens int64
	PeakDay        *ScopedTokenUsagePeakDay
}

type ScopedTokenUsageReport struct {
	Scope    TokenUsageScope
	FromDate time.Time
	ToDate   time.Time
	Days     []ScopedDailyTokenUsage
	Summary  ScopedTokenUsageSummary
}

type OrganizationDailyTokenUsage = ScopedDailyTokenUsage
type OrganizationTokenUsagePeakDay = ScopedTokenUsagePeakDay
type OrganizationTokenUsageSummary = ScopedTokenUsageSummary

type OrganizationTokenUsageReport struct {
	OrganizationID uuid.UUID
	FromDate       time.Time
	ToDate         time.Time
	Days           []OrganizationDailyTokenUsage
	Summary        OrganizationTokenUsageSummary
}

type GetProjectTokenUsage struct {
	ProjectID uuid.UUID
	FromDate  time.Time
	ToDate    time.Time
}

type ProjectDailyTokenUsage = ScopedDailyTokenUsage
type ProjectTokenUsagePeakDay = ScopedTokenUsagePeakDay
type ProjectTokenUsageSummary = ScopedTokenUsageSummary

type ProjectTokenUsageReport struct {
	ProjectID uuid.UUID
	FromDate  time.Time
	ToDate    time.Time
	Days      []ProjectDailyTokenUsage
	Summary   ProjectTokenUsageSummary
}

func NewOrganizationTokenUsageScope(organizationID uuid.UUID) TokenUsageScope {
	return TokenUsageScope{
		Kind: TokenUsageScopeKindOrganization,
		ID:   organizationID,
	}
}

func NewProjectTokenUsageScope(projectID uuid.UUID) TokenUsageScope {
	return TokenUsageScope{
		Kind: TokenUsageScopeKindProject,
		ID:   projectID,
	}
}

func (input GetOrganizationTokenUsage) Scope() TokenUsageScope {
	return NewOrganizationTokenUsageScope(input.OrganizationID)
}

func (input GetProjectTokenUsage) Scope() TokenUsageScope {
	return NewProjectTokenUsageScope(input.ProjectID)
}

func ParseScopedTokenUsage(
	scope TokenUsageScope,
	raw ScopedTokenUsageListInput,
	now time.Time,
) (GetScopedTokenUsage, error) {
	fromDate, toDate, err := parseTokenUsageDateRange(raw.From, raw.To, now)
	if err != nil {
		return GetScopedTokenUsage{}, err
	}

	return GetScopedTokenUsage{
		Scope:    scope,
		FromDate: fromDate,
		ToDate:   toDate,
	}, nil
}

func ParseOrganizationTokenUsage(
	organizationID uuid.UUID,
	raw OrganizationTokenUsageListInput,
	now time.Time,
) (GetOrganizationTokenUsage, error) {
	parsed, err := ParseScopedTokenUsage(NewOrganizationTokenUsageScope(organizationID), ScopedTokenUsageListInput(raw), now)
	if err != nil {
		return GetOrganizationTokenUsage{}, err
	}

	return GetOrganizationTokenUsage{
		OrganizationID: organizationID,
		FromDate:       parsed.FromDate,
		ToDate:         parsed.ToDate,
	}, nil
}

func ParseProjectTokenUsage(
	projectID uuid.UUID,
	raw ProjectTokenUsageListInput,
	now time.Time,
) (GetProjectTokenUsage, error) {
	parsed, err := ParseScopedTokenUsage(NewProjectTokenUsageScope(projectID), ScopedTokenUsageListInput(raw), now)
	if err != nil {
		return GetProjectTokenUsage{}, err
	}

	return GetProjectTokenUsage{
		ProjectID: projectID,
		FromDate:  parsed.FromDate,
		ToDate:    parsed.ToDate,
	}, nil
}

func parseTokenUsageDateRange(from string, to string, now time.Time) (time.Time, time.Time, error) {
	fromRaw := strings.TrimSpace(from)
	toRaw := strings.TrimSpace(to)
	if fromRaw == "" && toRaw == "" {
		end := startOfUTCDate(now.UTC())
		start := end.AddDate(0, 0, -(DefaultOrganizationTokenUsageWindowDays - 1))
		return start, end, nil
	}
	if fromRaw == "" || toRaw == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("from and to must both be provided in YYYY-MM-DD format")
	}

	fromDate, err := parseUTCDate("from", fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	toDate, err := parseUTCDate("to", toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if toDate.Before(fromDate) {
		return time.Time{}, time.Time{}, fmt.Errorf("to must be greater than or equal to from")
	}
	return fromDate, toDate, nil
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
