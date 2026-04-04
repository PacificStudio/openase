package httpapi

import (
	"net/http"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/labstack/echo/v4"
)

type workspaceDashboardMetricsResponse struct {
	OrganizationCount int     `json:"organization_count"`
	ProjectCount      int     `json:"project_count"`
	ProviderCount     int     `json:"provider_count"`
	RunningAgents     int     `json:"running_agents"`
	ActiveTickets     int     `json:"active_tickets"`
	TodayCost         float64 `json:"today_cost"`
	TotalTokens       int64   `json:"total_tokens"`
}

type workspaceOrganizationSummaryResponse struct {
	OrganizationID string  `json:"organization_id"`
	Name           string  `json:"name"`
	Slug           string  `json:"slug"`
	ProjectCount   int     `json:"project_count"`
	ProviderCount  int     `json:"provider_count"`
	RunningAgents  int     `json:"running_agents"`
	ActiveTickets  int     `json:"active_tickets"`
	TodayCost      float64 `json:"today_cost"`
	TotalTokens    int64   `json:"total_tokens"`
}

type organizationDashboardMetricsResponse struct {
	OrganizationID     string  `json:"organization_id"`
	ProjectCount       int     `json:"project_count"`
	ActiveProjectCount int     `json:"active_project_count"`
	ProviderCount      int     `json:"provider_count"`
	RunningAgents      int     `json:"running_agents"`
	ActiveTickets      int     `json:"active_tickets"`
	TodayCost          float64 `json:"today_cost"`
	TotalTokens        int64   `json:"total_tokens"`
}

type organizationProjectSummaryResponse struct {
	ProjectID      string  `json:"project_id"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Status         string  `json:"status"`
	RunningAgents  int     `json:"running_agents"`
	ActiveTickets  int     `json:"active_tickets"`
	TodayCost      float64 `json:"today_cost"`
	TotalTokens    int64   `json:"total_tokens"`
	LastActivityAt *string `json:"last_activity_at,omitempty"`
}

type organizationTokenUsageDayResponse struct {
	Date              string `json:"date"`
	InputTokens       int64  `json:"input_tokens"`
	OutputTokens      int64  `json:"output_tokens"`
	CachedInputTokens int64  `json:"cached_input_tokens"`
	ReasoningTokens   int64  `json:"reasoning_tokens"`
	TotalTokens       int64  `json:"total_tokens"`
	FinalizedRunCount int    `json:"finalized_run_count"`
}

type organizationTokenUsagePeakDayResponse struct {
	Date        string `json:"date"`
	TotalTokens int64  `json:"total_tokens"`
}

type organizationTokenUsageSummaryResponse struct {
	TotalTokens    int64                                  `json:"total_tokens"`
	AvgDailyTokens int64                                  `json:"avg_daily_tokens"`
	PeakDay        *organizationTokenUsagePeakDayResponse `json:"peak_day,omitempty"`
}

type organizationTokenUsageResponse struct {
	Days    []organizationTokenUsageDayResponse   `json:"days"`
	Summary organizationTokenUsageSummaryResponse `json:"summary"`
}

type projectTokenUsageDayResponse = organizationTokenUsageDayResponse
type projectTokenUsagePeakDayResponse = organizationTokenUsagePeakDayResponse
type projectTokenUsageSummaryResponse = organizationTokenUsageSummaryResponse
type projectTokenUsageResponse = organizationTokenUsageResponse

func (s *Server) registerWorkspaceSummaryRoutes(api *echo.Group) {
	api.GET("/workspace/summary", s.handleGetWorkspaceSummary)
	api.GET("/orgs/:orgId/summary", s.handleGetOrganizationSummary)
	api.GET("/orgs/:orgId/token-usage", s.handleGetOrganizationTokenUsage)
	api.GET("/projects/:projectId/token-usage", s.handleGetProjectTokenUsage)
}

func (s *Server) handleGetWorkspaceSummary(c echo.Context) error {
	summary, err := s.catalog.GetWorkspaceDashboardSummary(c.Request().Context())
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"workspace":     mapWorkspaceDashboardMetrics(summary),
		"organizations": mapWorkspaceOrganizationSummaries(summary.Organizations),
	})
}

func (s *Server) handleGetOrganizationSummary(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_ORG_ID", err.Error())
	}

	summary, err := s.catalog.GetOrganizationDashboardSummary(c.Request().Context(), orgID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"organization": mapOrganizationDashboardMetrics(summary),
		"projects":     mapOrganizationProjectSummaries(summary.Projects),
	})
}

func (s *Server) handleGetOrganizationTokenUsage(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	query, err := domain.ParseOrganizationTokenUsage(orgID, domain.OrganizationTokenUsageListInput{
		From: c.QueryParam("from"),
		To:   c.QueryParam("to"),
	}, time.Now().UTC())
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	report, err := s.catalog.GetOrganizationTokenUsage(c.Request().Context(), query)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, mapOrganizationTokenUsageResponse(report))
}

func (s *Server) handleGetProjectTokenUsage(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}

	query, err := domain.ParseProjectTokenUsage(projectID, domain.ProjectTokenUsageListInput{
		From: c.QueryParam("from"),
		To:   c.QueryParam("to"),
	}, time.Now().UTC())
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	report, err := s.catalog.GetProjectTokenUsage(c.Request().Context(), query)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, mapProjectTokenUsageResponse(report))
}

func mapWorkspaceDashboardMetrics(summary domain.WorkspaceDashboardSummary) workspaceDashboardMetricsResponse {
	return workspaceDashboardMetricsResponse{
		OrganizationCount: summary.OrganizationCount,
		ProjectCount:      summary.ProjectCount,
		ProviderCount:     summary.ProviderCount,
		RunningAgents:     summary.RunningAgents,
		ActiveTickets:     summary.ActiveTickets,
		TodayCost:         summary.TodayCost,
		TotalTokens:       summary.TotalTokens,
	}
}

func mapWorkspaceOrganizationSummaries(items []domain.WorkspaceOrganizationSummary) []workspaceOrganizationSummaryResponse {
	response := make([]workspaceOrganizationSummaryResponse, 0, len(items))
	for _, item := range items {
		response = append(response, workspaceOrganizationSummaryResponse{
			OrganizationID: item.OrganizationID.String(),
			Name:           item.Name,
			Slug:           item.Slug,
			ProjectCount:   item.ProjectCount,
			ProviderCount:  item.ProviderCount,
			RunningAgents:  item.RunningAgents,
			ActiveTickets:  item.ActiveTickets,
			TodayCost:      item.TodayCost,
			TotalTokens:    item.TotalTokens,
		})
	}

	return response
}

func mapOrganizationDashboardMetrics(summary domain.OrganizationDashboardSummary) organizationDashboardMetricsResponse {
	return organizationDashboardMetricsResponse{
		OrganizationID:     summary.OrganizationID.String(),
		ProjectCount:       summary.ProjectCount,
		ActiveProjectCount: summary.ActiveProjectCount,
		ProviderCount:      summary.ProviderCount,
		RunningAgents:      summary.RunningAgents,
		ActiveTickets:      summary.ActiveTickets,
		TodayCost:          summary.TodayCost,
		TotalTokens:        summary.TotalTokens,
	}
}

func mapOrganizationProjectSummaries(items []domain.OrganizationProjectSummary) []organizationProjectSummaryResponse {
	response := make([]organizationProjectSummaryResponse, 0, len(items))
	for _, item := range items {
		projectSummary := organizationProjectSummaryResponse{
			ProjectID:     item.ProjectID.String(),
			Name:          item.Name,
			Description:   item.Description,
			Status:        item.Status,
			RunningAgents: item.RunningAgents,
			ActiveTickets: item.ActiveTickets,
			TodayCost:     item.TodayCost,
			TotalTokens:   item.TotalTokens,
		}
		if item.LastActivityAt != nil {
			value := item.LastActivityAt.UTC().Format(time.RFC3339)
			projectSummary.LastActivityAt = &value
		}
		response = append(response, projectSummary)
	}

	return response
}

func mapOrganizationTokenUsageResponse(report domain.OrganizationTokenUsageReport) organizationTokenUsageResponse {
	response := organizationTokenUsageResponse{
		Days: make([]organizationTokenUsageDayResponse, 0, len(report.Days)),
		Summary: organizationTokenUsageSummaryResponse{
			TotalTokens:    report.Summary.TotalTokens,
			AvgDailyTokens: report.Summary.AvgDailyTokens,
		},
	}
	if report.Summary.PeakDay != nil {
		response.Summary.PeakDay = &organizationTokenUsagePeakDayResponse{
			Date:        report.Summary.PeakDay.Date.UTC().Format("2006-01-02"),
			TotalTokens: report.Summary.PeakDay.TotalTokens,
		}
	}

	for _, item := range report.Days {
		response.Days = append(response.Days, organizationTokenUsageDayResponse{
			Date:              item.UsageDate.UTC().Format("2006-01-02"),
			InputTokens:       item.InputTokens,
			OutputTokens:      item.OutputTokens,
			CachedInputTokens: item.CachedInputTokens,
			ReasoningTokens:   item.ReasoningTokens,
			TotalTokens:       item.TotalTokens,
			FinalizedRunCount: item.FinalizedRunCount,
		})
	}

	return response
}

func mapProjectTokenUsageResponse(report domain.ProjectTokenUsageReport) projectTokenUsageResponse {
	return projectTokenUsageResponse(mapOrganizationTokenUsageResponse(domain.OrganizationTokenUsageReport{
		FromDate: report.FromDate,
		ToDate:   report.ToDate,
		Days:     report.Days,
		Summary:  report.Summary,
	}))
}
