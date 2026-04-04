package httpapi

import (
	"errors"
	"net/http"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/notification"
	notificationservice "github.com/BetterAndBetterII/openase/internal/notification"
	"github.com/labstack/echo/v4"
)

type notificationChannelResponse struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	Name           string         `json:"name"`
	Type           string         `json:"type"`
	Config         map[string]any `json:"config"`
	IsEnabled      bool           `json:"is_enabled"`
	CreatedAt      string         `json:"created_at"`
}

type notificationRuleResponse struct {
	ID        string                      `json:"id"`
	ProjectID string                      `json:"project_id"`
	ChannelID string                      `json:"channel_id"`
	Name      string                      `json:"name"`
	EventType string                      `json:"event_type"`
	Filter    map[string]any              `json:"filter"`
	Template  string                      `json:"template"`
	IsEnabled bool                        `json:"is_enabled"`
	CreatedAt string                      `json:"created_at"`
	Channel   notificationChannelResponse `json:"channel"`
}

type notificationRuleEventTypeResponse struct {
	EventType       string `json:"event_type"`
	Label           string `json:"label"`
	DefaultTemplate string `json:"default_template"`
}

func (s *Server) registerNotificationRoutes(api *echo.Group) {
	api.GET("/notification-event-types", s.handleListNotificationRuleEventTypes)
	api.GET("/orgs/:orgId/channels", s.handleListNotificationChannels)
	api.POST("/orgs/:orgId/channels", s.handleCreateNotificationChannel)
	api.PATCH("/channels/:channelId", s.handleUpdateNotificationChannel)
	api.DELETE("/channels/:channelId", s.handleDeleteNotificationChannel)
	api.POST("/channels/:channelId/test", s.handleTestNotificationChannel)
	api.GET("/projects/:projectId/notification-rules", s.handleListNotificationRules)
	api.POST("/projects/:projectId/notification-rules", s.handleCreateNotificationRule)
	api.PATCH("/notification-rules/:ruleId", s.handleUpdateNotificationRule)
	api.DELETE("/notification-rules/:ruleId", s.handleDeleteNotificationRule)
}

func (s *Server) handleListNotificationChannels(c echo.Context) error {
	if s.notificationService == nil {
		return writeNotificationError(c, notificationservice.ErrUnavailable)
	}

	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	items, err := s.notificationService.List(c.Request().Context(), orgID)
	if err != nil {
		return writeNotificationError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"channels": mapNotificationChannelResponses(items),
	})
}

func (s *Server) handleCreateNotificationChannel(c echo.Context) error {
	if s.notificationService == nil {
		return writeNotificationError(c, notificationservice.ErrUnavailable)
	}

	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	var raw domain.ChannelInput
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := domain.ParseCreateChannel(orgID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.notificationService.Create(c.Request().Context(), input)
	if err != nil {
		return writeNotificationError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"channel": mapNotificationChannelResponse(item),
	})
}

func (s *Server) handleUpdateNotificationChannel(c echo.Context) error {
	if s.notificationService == nil {
		return writeNotificationError(c, notificationservice.ErrUnavailable)
	}

	channelID, err := parseUUIDPathParamValue(c, "channelId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHANNEL_ID", err.Error())
	}

	var raw domain.ChannelPatchInput
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := domain.ParseUpdateChannel(channelID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.notificationService.Update(c.Request().Context(), input)
	if err != nil {
		return writeNotificationError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"channel": mapNotificationChannelResponse(item),
	})
}

func (s *Server) handleDeleteNotificationChannel(c echo.Context) error {
	if s.notificationService == nil {
		return writeNotificationError(c, notificationservice.ErrUnavailable)
	}

	channelID, err := parseUUIDPathParamValue(c, "channelId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHANNEL_ID", err.Error())
	}

	if err := s.notificationService.Delete(c.Request().Context(), channelID); err != nil {
		return writeNotificationError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"deleted_channel_id": channelID.String(),
	})
}

func (s *Server) handleTestNotificationChannel(c echo.Context) error {
	if s.notificationService == nil {
		return writeNotificationError(c, notificationservice.ErrUnavailable)
	}

	channelID, err := parseUUIDPathParamValue(c, "channelId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHANNEL_ID", err.Error())
	}

	if err := s.notificationService.Test(c.Request().Context(), channelID); err != nil {
		return writeNotificationError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status": "sent",
	})
}

func (s *Server) handleListNotificationRuleEventTypes(c echo.Context) error {
	items := domain.SupportedRuleEvents()
	response := make([]notificationRuleEventTypeResponse, 0, len(items))
	for _, item := range items {
		response = append(response, notificationRuleEventTypeResponse{
			EventType:       item.EventType.String(),
			Label:           item.Label,
			DefaultTemplate: item.DefaultTemplate,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"event_types": response,
	})
}

func (s *Server) handleListNotificationRules(c echo.Context) error {
	if s.notificationService == nil {
		return writeNotificationError(c, notificationservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	items, err := s.notificationService.ListRules(c.Request().Context(), projectID)
	if err != nil {
		return writeNotificationError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"rules": mapNotificationRuleResponses(items),
	})
}

func (s *Server) handleCreateNotificationRule(c echo.Context) error {
	if s.notificationService == nil {
		return writeNotificationError(c, notificationservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	var raw domain.RuleInput
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := domain.ParseCreateRule(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.notificationService.CreateRule(c.Request().Context(), input)
	if err != nil {
		return writeNotificationError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"rule": mapNotificationRuleResponse(item),
	})
}

func (s *Server) handleUpdateNotificationRule(c echo.Context) error {
	if s.notificationService == nil {
		return writeNotificationError(c, notificationservice.ErrUnavailable)
	}

	ruleID, err := parseUUIDPathParamValue(c, "ruleId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_RULE_ID", err.Error())
	}

	var raw domain.RulePatchInput
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := domain.ParseUpdateRule(ruleID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.notificationService.UpdateRule(c.Request().Context(), input)
	if err != nil {
		return writeNotificationError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"rule": mapNotificationRuleResponse(item),
	})
}

func (s *Server) handleDeleteNotificationRule(c echo.Context) error {
	if s.notificationService == nil {
		return writeNotificationError(c, notificationservice.ErrUnavailable)
	}

	ruleID, err := parseUUIDPathParamValue(c, "ruleId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_RULE_ID", err.Error())
	}

	if err := s.notificationService.DeleteRule(c.Request().Context(), ruleID); err != nil {
		return writeNotificationError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"deleted_rule_id": ruleID.String(),
	})
}

func writeNotificationError(c echo.Context, err error) error {
	var channelConflict *domain.ChannelUsageConflict
	switch {
	case errors.Is(err, notificationservice.ErrUnavailable):
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	case errors.Is(err, notificationservice.ErrOrganizationNotFound):
		return writeAPIError(c, http.StatusNotFound, "ORGANIZATION_NOT_FOUND", err.Error())
	case errors.Is(err, notificationservice.ErrProjectNotFound):
		return writeAPIError(c, http.StatusNotFound, "PROJECT_NOT_FOUND", err.Error())
	case errors.Is(err, notificationservice.ErrChannelNotFound):
		return writeAPIError(c, http.StatusNotFound, "CHANNEL_NOT_FOUND", err.Error())
	case errors.As(err, &channelConflict):
		return writeAPIErrorWithDetails(c, http.StatusConflict, "CHANNEL_IN_USE", notificationservice.ErrChannelInUse.Error(), channelConflict)
	case errors.Is(err, notificationservice.ErrDuplicateChannelName):
		return writeAPIError(c, http.StatusConflict, "CHANNEL_NAME_CONFLICT", err.Error())
	case errors.Is(err, notificationservice.ErrRuleNotFound):
		return writeAPIError(c, http.StatusNotFound, "RULE_NOT_FOUND", err.Error())
	case errors.Is(err, notificationservice.ErrDuplicateRuleName):
		return writeAPIError(c, http.StatusConflict, "RULE_NAME_CONFLICT", err.Error())
	case errors.Is(err, notificationservice.ErrChannelProjectMismatch):
		return writeAPIError(c, http.StatusBadRequest, "CHANNEL_PROJECT_MISMATCH", err.Error())
	case errors.Is(err, notificationservice.ErrInvalidChannelConfig):
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHANNEL_CONFIG", err.Error())
	case errors.Is(err, domain.ErrChannelTypeUnsupported):
		return writeAPIError(c, http.StatusBadRequest, "CHANNEL_TYPE_UNSUPPORTED", err.Error())
	case errors.Is(err, notificationservice.ErrAdapterUnavailable):
		return writeAPIError(c, http.StatusBadRequest, "ADAPTER_UNAVAILABLE", err.Error())
	default:
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func mapNotificationChannelResponses(items []domain.Channel) []notificationChannelResponse {
	response := make([]notificationChannelResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapNotificationChannelResponse(item))
	}

	return response
}

func mapNotificationChannelResponse(item domain.Channel) notificationChannelResponse {
	return notificationChannelResponse{
		ID:             item.ID.String(),
		OrganizationID: item.OrganizationID.String(),
		Name:           item.Name,
		Type:           item.Type.String(),
		Config:         domain.RedactedConfig(item.Type, item.Config),
		IsEnabled:      item.IsEnabled,
		CreatedAt:      item.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func mapNotificationRuleResponses(items []domain.Rule) []notificationRuleResponse {
	response := make([]notificationRuleResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapNotificationRuleResponse(item))
	}

	return response
}

func mapNotificationRuleResponse(item domain.Rule) notificationRuleResponse {
	return notificationRuleResponse{
		ID:        item.ID.String(),
		ProjectID: item.ProjectID.String(),
		ChannelID: item.ChannelID.String(),
		Name:      item.Name,
		EventType: item.EventType.String(),
		Filter:    item.Filter,
		Template:  item.Template,
		IsEnabled: item.IsEnabled,
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
		Channel:   mapNotificationChannelResponse(item.Channel),
	}
}
