package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type projectSecuritySurfaceResponse struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Exposed    bool   `json:"exposed"`
	Configured bool   `json:"configured"`
	Summary    string `json:"summary"`
}

type projectSecurityAgentPlatformResponse struct {
	Exposed           bool     `json:"exposed"`
	ActiveTokenCount  int      `json:"active_token_count"`
	ExpiredTokenCount int      `json:"expired_token_count"`
	LastTokenIssuedAt *string  `json:"last_token_issued_at,omitempty"`
	LastTokenUsedAt   *string  `json:"last_token_used_at,omitempty"`
	DefaultScopes     []string `json:"default_scopes"`
	PrivilegedScopes  []string `json:"privileged_scopes"`
	Summary           string   `json:"summary"`
}

type projectSecuritySettingsResponse struct {
	ProjectID     string                               `json:"project_id"`
	RuntimeMode   string                               `json:"runtime_mode"`
	Surfaces      []projectSecuritySurfaceResponse     `json:"surfaces"`
	AgentPlatform projectSecurityAgentPlatformResponse `json:"agent_platform"`
	Notes         []string                             `json:"notes"`
}

func (s *Server) registerSecuritySettingsRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/security", s.handleGetProjectSecuritySettings)
}

func (s *Server) handleGetProjectSecuritySettings(c echo.Context) error {
	if s.catalog == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	project, err := s.catalog.GetProject(c.Request().Context(), projectID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"security": projectSecuritySettingsResponse{
			ProjectID:   project.ID.String(),
			RuntimeMode: string(s.cfg.Mode),
			Surfaces: []projectSecuritySurfaceResponse{
				buildGitHubWebhookSurfaceResponse(string(s.cfg.Mode), strings.TrimSpace(s.github.WebhookSecret) != ""),
				buildAgentPlatformSurfaceResponse(s.agentPlatform != nil),
			},
			AgentPlatform: buildProjectSecurityAgentPlatformResponse(c.Request().Context(), s.agentPlatform, project.ID),
			Notes: []string{
				"Security settings currently audit runtime ingress and project-scoped agent token exposure.",
				"Global browser auth and future OIDC setup remain outside the current app settings scope.",
			},
		},
	})
}

func buildGitHubWebhookSurfaceResponse(mode string, secretConfigured bool) projectSecuritySurfaceResponse {
	exposed := mode != "orchestrate"
	summary := "Inbound GitHub webhooks are not exposed in orchestrate-only mode."
	if exposed && secretConfigured {
		summary = "Inbound GitHub webhooks are exposed and protected by a configured shared secret."
	}
	if exposed && !secretConfigured {
		summary = "Inbound GitHub webhooks are exposed, but no shared secret is configured yet."
	}

	return projectSecuritySurfaceResponse{
		Key:        "github-webhooks",
		Label:      "GitHub inbound webhooks",
		Exposed:    exposed,
		Configured: secretConfigured,
		Summary:    summary,
	}
}

func buildAgentPlatformSurfaceResponse(exposed bool) projectSecuritySurfaceResponse {
	summary := "The project-scoped agent platform API is not exposed by this runtime."
	if exposed {
		summary = "The project-scoped agent platform API can accept bearer tokens issued for this project."
	}

	return projectSecuritySurfaceResponse{
		Key:        "agent-platform",
		Label:      "Agent platform API",
		Exposed:    exposed,
		Configured: exposed,
		Summary:    summary,
	}
}

func buildProjectSecurityAgentPlatformResponse(
	ctx context.Context,
	service *agentplatform.Service,
	projectID uuid.UUID,
) projectSecurityAgentPlatformResponse {
	response := projectSecurityAgentPlatformResponse{
		Exposed:          service != nil,
		DefaultScopes:    agentplatform.DefaultScopes(),
		PrivilegedScopes: agentplatform.PrivilegedScopes(),
		Summary:          "The project-scoped agent platform API is not exposed by this runtime.",
	}
	if service == nil {
		return response
	}

	inventory, err := service.ProjectTokenInventory(ctx, projectID)
	if err != nil {
		response.Summary = fmt.Sprintf("Failed to load project token inventory: %v", err)
		return response
	}

	response.ActiveTokenCount = inventory.ActiveTokenCount
	response.ExpiredTokenCount = inventory.ExpiredTokenCount
	response.LastTokenIssuedAt = timeStringPointer(inventory.LastIssuedAt)
	response.LastTokenUsedAt = timeStringPointer(inventory.LastUsedAt)
	response.Summary = fmt.Sprintf(
		"Project-scoped agent tokens are active on this runtime. %d active and %d expired tokens are currently recorded.",
		inventory.ActiveTokenCount,
		inventory.ExpiredTokenCount,
	)

	return response
}

func timeStringPointer(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}
