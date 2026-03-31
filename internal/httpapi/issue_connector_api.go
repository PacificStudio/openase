package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/issueconnector"
	issueconnectorservice "github.com/BetterAndBetterII/openase/internal/service/issueconnector"
	"github.com/labstack/echo/v4"
)

type issueConnectorConfigResponse struct {
	Type                    string            `json:"type"`
	BaseURL                 string            `json:"base_url"`
	ProjectRef              string            `json:"project_ref"`
	PollInterval            string            `json:"poll_interval"`
	SyncDirection           string            `json:"sync_direction"`
	Filters                 domain.Filters    `json:"filters"`
	StatusMapping           map[string]string `json:"status_mapping"`
	AutoWorkflow            string            `json:"auto_workflow"`
	AuthTokenConfigured     bool              `json:"auth_token_configured"`
	WebhookSecretConfigured bool              `json:"webhook_secret_configured"`
}

type issueConnectorResponse struct {
	ID         string                       `json:"id"`
	ProjectID  string                       `json:"project_id"`
	Type       string                       `json:"type"`
	Name       string                       `json:"name"`
	Status     string                       `json:"status"`
	Config     issueConnectorConfigResponse `json:"config"`
	LastSyncAt *string                      `json:"last_sync_at,omitempty"`
	LastError  string                       `json:"last_error"`
	Stats      domain.SyncStats             `json:"stats"`
}

type issueConnectorTestResponse struct {
	Result issueconnectorservice.TestResult `json:"result"`
}

type issueConnectorSyncResponse struct {
	Connector issueConnectorResponse          `json:"connector"`
	Report    orchestratorConnectorSyncReport `json:"report"`
}

type orchestratorConnectorSyncReport struct {
	ConnectorsScanned int `json:"connectors_scanned"`
	ConnectorsSynced  int `json:"connectors_synced"`
	ConnectorsFailed  int `json:"connectors_failed"`
	IssuesSynced      int `json:"issues_synced"`
}

type issueConnectorStatsEnvelope struct {
	Stats issueconnectorservice.StatsResult `json:"stats"`
}

func (s *Server) registerIssueConnectorRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/connectors", s.handleListIssueConnectors)
	api.POST("/projects/:projectId/connectors", s.handleCreateIssueConnector)
	api.PATCH("/connectors/:connectorId", s.handleUpdateIssueConnector)
	api.DELETE("/connectors/:connectorId", s.handleDeleteIssueConnector)
	api.POST("/connectors/:connectorId/test", s.handleTestIssueConnector)
	api.POST("/connectors/:connectorId/sync", s.handleSyncIssueConnector)
	api.GET("/connectors/:connectorId/stats", s.handleGetIssueConnectorStats)
}

func (s *Server) handleListIssueConnectors(c echo.Context) error {
	if s.issueConnectorSvc == nil {
		return writeIssueConnectorError(c, issueconnectorservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	connectors, err := s.issueConnectorSvc.List(c.Request().Context(), projectID)
	if err != nil {
		return writeIssueConnectorError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"connectors": mapIssueConnectorResponses(connectors),
	})
}

func (s *Server) handleCreateIssueConnector(c echo.Context) error {
	if s.issueConnectorSvc == nil {
		return writeIssueConnectorError(c, issueconnectorservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	var raw rawCreateIssueConnectorRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	input, err := parseCreateIssueConnectorRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	connector, err := s.issueConnectorSvc.Create(c.Request().Context(), input)
	if err != nil {
		return writeIssueConnectorError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"connector": mapIssueConnectorResponse(connector),
	})
}

func (s *Server) handleUpdateIssueConnector(c echo.Context) error {
	if s.issueConnectorSvc == nil {
		return writeIssueConnectorError(c, issueconnectorservice.ErrUnavailable)
	}

	connectorID, err := parseUUIDPathParamValue(c, "connectorId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONNECTOR_ID", err.Error())
	}

	var raw rawUpdateIssueConnectorRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	input, err := parseUpdateIssueConnectorRequest(connectorID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	connector, err := s.issueConnectorSvc.Update(c.Request().Context(), input)
	if err != nil {
		return writeIssueConnectorError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"connector": mapIssueConnectorResponse(connector),
	})
}

func (s *Server) handleDeleteIssueConnector(c echo.Context) error {
	if s.issueConnectorSvc == nil {
		return writeIssueConnectorError(c, issueconnectorservice.ErrUnavailable)
	}

	connectorID, err := parseUUIDPathParamValue(c, "connectorId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONNECTOR_ID", err.Error())
	}

	if err := s.issueConnectorSvc.Delete(c.Request().Context(), connectorID); err != nil {
		return writeIssueConnectorError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"deleted_connector_id": connectorID,
	})
}

func (s *Server) handleTestIssueConnector(c echo.Context) error {
	if s.issueConnectorSvc == nil {
		return writeIssueConnectorError(c, issueconnectorservice.ErrUnavailable)
	}

	connectorID, err := parseUUIDPathParamValue(c, "connectorId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONNECTOR_ID", err.Error())
	}

	result, err := s.issueConnectorSvc.Test(c.Request().Context(), connectorID)
	if err != nil {
		return writeIssueConnectorError(c, err)
	}

	return c.JSON(http.StatusOK, issueConnectorTestResponse{Result: result})
}

func (s *Server) handleSyncIssueConnector(c echo.Context) error {
	if s.issueConnectorSvc == nil {
		return writeIssueConnectorError(c, issueconnectorservice.ErrUnavailable)
	}

	connectorID, err := parseUUIDPathParamValue(c, "connectorId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONNECTOR_ID", err.Error())
	}

	result, err := s.issueConnectorSvc.Sync(c.Request().Context(), connectorID)
	if err != nil {
		return writeIssueConnectorError(c, err)
	}

	return c.JSON(http.StatusOK, issueConnectorSyncResponse{
		Connector: mapIssueConnectorResponse(result.Connector),
		Report: orchestratorConnectorSyncReport{
			ConnectorsScanned: result.Report.ConnectorsScanned,
			ConnectorsSynced:  result.Report.ConnectorsSynced,
			ConnectorsFailed:  result.Report.ConnectorsFailed,
			IssuesSynced:      result.Report.IssuesSynced,
		},
	})
}

func (s *Server) handleGetIssueConnectorStats(c echo.Context) error {
	if s.issueConnectorSvc == nil {
		return writeIssueConnectorError(c, issueconnectorservice.ErrUnavailable)
	}

	connectorID, err := parseUUIDPathParamValue(c, "connectorId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONNECTOR_ID", err.Error())
	}

	stats, err := s.issueConnectorSvc.Stats(c.Request().Context(), connectorID)
	if err != nil {
		return writeIssueConnectorError(c, err)
	}

	return c.JSON(http.StatusOK, issueConnectorStatsEnvelope{Stats: stats})
}

func writeIssueConnectorError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, issueconnectorservice.ErrUnavailable), errors.Is(err, issueconnectorservice.ErrConnectorRuntimeAbsent):
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	case errors.Is(err, issueconnectorservice.ErrProjectNotFound):
		return writeAPIError(c, http.StatusNotFound, "PROJECT_NOT_FOUND", err.Error())
	case errors.Is(err, issueconnectorservice.ErrConnectorNotFound):
		return writeAPIError(c, http.StatusNotFound, "CONNECTOR_NOT_FOUND", err.Error())
	case errors.Is(err, issueconnectorservice.ErrConnectorConflict):
		return writeAPIError(c, http.StatusConflict, "CONNECTOR_CONFLICT", err.Error())
	case errors.Is(err, issueconnectorservice.ErrConnectorTypeNotFound):
		return writeAPIError(c, http.StatusBadRequest, "CONNECTOR_TYPE_NOT_FOUND", err.Error())
	default:
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func mapIssueConnectorResponses(items []domain.IssueConnector) []issueConnectorResponse {
	response := make([]issueConnectorResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapIssueConnectorResponse(item))
	}

	return response
}

func mapIssueConnectorResponse(item domain.IssueConnector) issueConnectorResponse {
	response := issueConnectorResponse{
		ID:        item.ID.String(),
		ProjectID: item.ProjectID.String(),
		Type:      string(item.Type),
		Name:      item.Name,
		Status:    string(item.Status),
		Config: issueConnectorConfigResponse{
			Type:                    string(item.Config.Type),
			BaseURL:                 item.Config.BaseURL,
			ProjectRef:              item.Config.ProjectRef,
			PollInterval:            item.Config.PollInterval.String(),
			SyncDirection:           string(item.Config.SyncDirection),
			Filters:                 item.Config.Filters,
			StatusMapping:           cloneHTTPStringMap(item.Config.StatusMapping),
			AutoWorkflow:            item.Config.AutoWorkflow,
			AuthTokenConfigured:     strings.TrimSpace(item.Config.AuthToken) != "",
			WebhookSecretConfigured: strings.TrimSpace(item.Config.WebhookSecret) != "",
		},
		LastError: item.LastError,
		Stats:     item.Stats,
	}
	if item.LastSyncAt != nil {
		syncedAt := item.LastSyncAt.UTC().Format(time.RFC3339)
		response.LastSyncAt = &syncedAt
	}

	return response
}
