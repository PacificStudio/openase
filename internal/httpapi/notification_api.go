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

func (s *Server) registerNotificationRoutes(api *echo.Group) {
	api.GET("/orgs/:orgId/channels", s.handleListNotificationChannels)
	api.POST("/orgs/:orgId/channels", s.handleCreateNotificationChannel)
	api.PATCH("/channels/:channelId", s.handleUpdateNotificationChannel)
	api.DELETE("/channels/:channelId", s.handleDeleteNotificationChannel)
	api.POST("/channels/:channelId/test", s.handleTestNotificationChannel)
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

func writeNotificationError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, notificationservice.ErrUnavailable):
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	case errors.Is(err, notificationservice.ErrOrganizationNotFound):
		return writeAPIError(c, http.StatusNotFound, "ORGANIZATION_NOT_FOUND", err.Error())
	case errors.Is(err, notificationservice.ErrProjectNotFound):
		return writeAPIError(c, http.StatusNotFound, "PROJECT_NOT_FOUND", err.Error())
	case errors.Is(err, notificationservice.ErrChannelNotFound):
		return writeAPIError(c, http.StatusNotFound, "CHANNEL_NOT_FOUND", err.Error())
	case errors.Is(err, notificationservice.ErrDuplicateChannelName):
		return writeAPIError(c, http.StatusConflict, "CHANNEL_NAME_CONFLICT", err.Error())
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
