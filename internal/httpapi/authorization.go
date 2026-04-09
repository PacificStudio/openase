package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type rawCreateRoleBindingRequest struct {
	SubjectKind string  `json:"subject_kind"`
	SubjectKey  string  `json:"subject_key"`
	RoleKey     string  `json:"role_key"`
	ExpiresAt   *string `json:"expires_at"`
}

type roleBindingResponse struct {
	ID          string  `json:"id"`
	ScopeKind   string  `json:"scope_kind"`
	ScopeID     string  `json:"scope_id"`
	SubjectKind string  `json:"subject_kind"`
	SubjectKey  string  `json:"subject_key"`
	RoleKey     string  `json:"role_key"`
	GrantedBy   string  `json:"granted_by"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

type humanRouteScopeResolver uint8

const (
	humanRouteScopeResolverInstance humanRouteScopeResolver = iota
	humanRouteScopeResolverOrganization
	humanRouteScopeResolverProject
)

type humanRouteAuthorizationRule struct {
	scopeResolver humanRouteScopeResolver
	resource      string
	paramName     string
	permission    humanauthdomain.PermissionKey
	checkRequired bool
}

func (s *Server) registerRoleBindingRoutes(api *echo.Group) {
	api.GET("/instance/role-bindings", s.handleListInstanceRoleBindings)
	api.POST("/instance/role-bindings", s.handleCreateInstanceRoleBinding)
	api.DELETE("/instance/role-bindings/:bindingId", s.handleDeleteInstanceRoleBinding)
	api.GET("/organizations/:orgId/role-bindings", s.handleListOrganizationRoleBindings)
	api.POST("/organizations/:orgId/role-bindings", s.handleCreateOrganizationRoleBinding)
	api.DELETE("/organizations/:orgId/role-bindings/:bindingId", s.handleDeleteOrganizationRoleBinding)
	api.GET("/projects/:projectId/role-bindings", s.handleListProjectRoleBindings)
	api.POST("/projects/:projectId/role-bindings", s.handleCreateProjectRoleBinding)
	api.DELETE("/projects/:projectId/role-bindings/:bindingId", s.handleDeleteProjectRoleBinding)
}

func (s *Server) authorizeHumanAPI(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		runtimeState, err := s.currentRuntimeAccessControlState(c)
		if err != nil {
			return writeAuthRuntimeUnavailable(c, "AUTH_RUNTIME_STATE_FAILED", err)
		}
		if runtimeState.LoginRequired && s.humanAuthorizer == nil {
			return writeAPIError(c, http.StatusServiceUnavailable, "AUTHORIZATION_UNAVAILABLE", "authorization service unavailable")
		}
		principal, ok := currentHumanPrincipal(c)
		if !ok || !runtimeState.LoginRequired {
			return next(c)
		}
		scope, permission, checkRequired, err := s.requiredScopeAndPermission(c, principal)
		if err != nil {
			return writeAPIError(c, http.StatusForbidden, "AUTHORIZATION_DENIED", err.Error())
		}
		if !checkRequired {
			return next(c)
		}
		allowed, roles, permissions, err := s.humanAuthorizer.HasPermission(c.Request().Context(), principal, scope, permission)
		if err != nil {
			return writeAPIError(c, http.StatusForbidden, "AUTHORIZATION_DENIED", err.Error())
		}
		if !allowed {
			return writeAPIError(c, http.StatusForbidden, "AUTHORIZATION_DENIED", "required permission is missing")
		}
		_ = authzEvaluation{scope: scope, permission: permission, roles: roles, permissions: permissions}
		return next(c)
	}
}

func (s *Server) requiredScopeAndPermission(
	c echo.Context,
	principal humanauthdomain.AuthenticatedPrincipal,
) (humanauthdomain.ScopeRef, humanauthdomain.PermissionKey, bool, error) {
	path := c.Path()
	method := c.Request().Method
	rule, ok := humanRouteAuthorizationRuleFor(path, method)
	if !ok {
		return humanauthdomain.ScopeRef{}, "", true, humanauthservice.ErrPermissionDenied
	}

	switch rule.scopeResolver {
	case humanRouteScopeResolverInstance:
		return humanauthdomain.ScopeRef{Kind: humanauthdomain.ScopeKindInstance, ID: ""}, rule.permission, rule.checkRequired, nil
	case humanRouteScopeResolverOrganization:
		scope, err := s.humanAuthorizer.ResolveOrganizationScope(c.Request().Context(), rule.resource, parseUUIDStringUnsafe(c.Param(rule.paramName)))
		if err != nil {
			return humanauthdomain.ScopeRef{}, "", false, err
		}
		return scope, rule.permission, rule.checkRequired, nil
	case humanRouteScopeResolverProject:
		scope, err := s.humanAuthorizer.ResolveProjectScope(c.Request().Context(), rule.resource, parseUUIDStringUnsafe(c.Param(rule.paramName)))
		if err != nil {
			return humanauthdomain.ScopeRef{}, "", false, err
		}
		return scope, rule.permission, rule.checkRequired, nil
	default:
		return humanauthdomain.ScopeRef{}, "", true, humanauthservice.ErrPermissionDenied
	}
}

func humanRouteAuthorizationRuleFor(path string, method string) (humanRouteAuthorizationRule, bool) {
	switch path {
	case "/api/v1/app-context", "/api/v1/system/dashboard", "/api/v1/system/metrics", "/api/v1/workspace/summary", "/api/v1/provider-model-options", "/api/v1/openapi.json", "/api/v1/auth/me/permissions", "/api/v1/auth/sessions", "/api/v1/auth/sessions/:id", "/api/v1/auth/sessions/revoke-all", "/api/v1/roles/builtin", "/api/v1/roles/builtin/:roleSlug", "/api/v1/harness/variables", "/api/v1/harness/validate", "/api/v1/notification-event-types", "/api/v1/chat", "/api/v1/chat/:sessionId", "/api/v1/chat/conversations", "/api/v1/org-invitations/accept":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverInstance,
			checkRequired: false,
		}, true
	case "/api/v1/auth/users/:userId/sessions/revoke", "/api/v1/instance/sessions/:id":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverInstance,
			permission:    humanauthdomain.PermissionSecurityUpdate,
			checkRequired: true,
		}, true
	case "/api/v1/admin/auth":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverInstance,
			permission:    humanauthdomain.PermissionSecurityRead,
			checkRequired: true,
		}, true
	case "/api/v1/admin/auth/oidc-draft", "/api/v1/admin/auth/oidc-draft/test", "/api/v1/admin/auth/oidc-enable", "/api/v1/admin/auth/disable":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverInstance,
			permission:    humanauthdomain.PermissionSecurityUpdate,
			checkRequired: true,
		}, true
	case "/api/v1/projects/:projectId/security-settings/oidc-draft", "/api/v1/projects/:projectId/security-settings/oidc-draft/test", "/api/v1/projects/:projectId/security-settings/oidc-enable":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverInstance,
			permission:    humanauthdomain.PermissionSecurityUpdate,
			checkRequired: true,
		}, true
	case "/api/v1/instance/users", "/api/v1/instance/users/:userId", "/api/v1/instance/users/:userId/status":
		permission := humanauthdomain.PermissionSecurityUpdate
		if method == http.MethodGet {
			permission = humanauthdomain.PermissionSecurityRead
		}
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverInstance,
			permission:    permission,
			checkRequired: true,
		}, true
	case "/api/v1/instance/role-bindings", "/api/v1/instance/role-bindings/:bindingId":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverInstance,
			permission:    humanauthdomain.PermissionRBACManage,
			checkRequired: true,
		}, true
	case "/api/v1/orgs":
		if method == http.MethodGet {
			return humanRouteAuthorizationRule{
				scopeResolver: humanRouteScopeResolverInstance,
				checkRequired: false,
			}, true
		}
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverInstance,
			permission:    humanauthdomain.PermissionOrgCreate,
			checkRequired: true,
		}, true
	case "/api/v1/orgs/:orgId/projects":
		if method == http.MethodGet {
			return humanRouteAuthorizationRule{
				scopeResolver: humanRouteScopeResolverOrganization,
				resource:      "organization",
				paramName:     "orgId",
				checkRequired: false,
			}, true
		}
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverOrganization,
			resource:      "organization",
			paramName:     "orgId",
			permission:    humanauthdomain.PermissionProjectCreate,
			checkRequired: true,
		}, true
	case "/api/v1/orgs/:orgId/security/github-credential", "/api/v1/orgs/:orgId/security/github-credential/import-gh-cli", "/api/v1/orgs/:orgId/security/github-credential/retest":
		permission := humanauthdomain.PermissionSecurityUpdate
		if method == http.MethodGet {
			permission = humanauthdomain.PermissionSecurityRead
		}
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverOrganization,
			resource:      "organization",
			paramName:     "orgId",
			permission:    permission,
			checkRequired: true,
		}, true
	case "/api/v1/orgs/:orgId", "/api/v1/orgs/:orgId/summary", "/api/v1/orgs/:orgId/machines", "/api/v1/orgs/:orgId/providers", "/api/v1/orgs/:orgId/channels", "/api/v1/orgs/:orgId/machines/stream", "/api/v1/orgs/:orgId/providers/stream", "/api/v1/orgs/:orgId/token-usage", "/api/v1/orgs/:orgId/security-settings/secrets", "/api/v1/orgs/:orgId/security-settings/secrets/:secretId", "/api/v1/orgs/:orgId/security-settings/secrets/:secretId/rotate", "/api/v1/orgs/:orgId/security-settings/secrets/:secretId/disable", "/api/v1/organizations/:orgId/role-bindings", "/api/v1/organizations/:orgId/role-bindings/:bindingId", "/api/v1/orgs/:orgId/members", "/api/v1/orgs/:orgId/members/:membershipId", "/api/v1/orgs/:orgId/members/:membershipId/transfer-ownership", "/api/v1/orgs/:orgId/invitations", "/api/v1/orgs/:orgId/invitations/:invitationId/resend", "/api/v1/orgs/:orgId/invitations/:invitationId/cancel":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverOrganization,
			resource:      "organization",
			paramName:     "orgId",
			permission:    organizationPermissionForPath(path, method),
			checkRequired: true,
		}, true
	case "/api/v1/machines/:machineId", "/api/v1/machines/:machineId/test", "/api/v1/machines/:machineId/refresh-health", "/api/v1/machines/:machineId/resources":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverOrganization,
			resource:      "machine",
			paramName:     "machineId",
			permission:    machinePermissionForPath(path, method),
			checkRequired: true,
		}, true
	case "/api/v1/providers/:providerId":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverOrganization,
			resource:      "provider",
			paramName:     "providerId",
			permission:    providerPermissionForPath(method),
			checkRequired: true,
		}, true
	case "/api/v1/channels/:channelId", "/api/v1/channels/:channelId/test":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverOrganization,
			resource:      "channel",
			paramName:     "channelId",
			permission:    channelPermissionForPath(path, method),
			checkRequired: true,
		}, true
	case "/api/v1/projects/:projectId", "/api/v1/projects/:projectId/activity", "/api/v1/projects/:projectId/events/stream", "/api/v1/projects/:projectId/updates", "/api/v1/projects/:projectId/updates/:threadId", "/api/v1/projects/:projectId/updates/:threadId/revisions", "/api/v1/projects/:projectId/updates/:threadId/comments", "/api/v1/projects/:projectId/updates/:threadId/comments/:commentId", "/api/v1/projects/:projectId/updates/:threadId/comments/:commentId/revisions", "/api/v1/projects/:projectId/notification-rules", "/api/v1/projects/:projectId/scheduled-jobs", "/api/v1/projects/:projectId/skills", "/api/v1/projects/:projectId/skills/import", "/api/v1/projects/:projectId/skills/refresh", "/api/v1/projects/:projectId/workflows", "/api/v1/projects/:projectId/statuses", "/api/v1/projects/:projectId/statuses/reset", "/api/v1/projects/:projectId/tickets", "/api/v1/projects/:projectId/tickets/archived", "/api/v1/projects/:projectId/tickets/:ticketId/detail", "/api/v1/projects/:projectId/tickets/:ticketId/repo-scopes", "/api/v1/projects/:projectId/tickets/:ticketId/runs", "/api/v1/projects/:projectId/tickets/:ticketId/runs/:runId", "/api/v1/projects/:projectId/repos", "/api/v1/projects/:projectId/token-usage", "/api/v1/projects/:projectId/github/namespaces", "/api/v1/projects/:projectId/github/repos", "/api/v1/projects/:projectId/agents", "/api/v1/projects/:projectId/agent-runs", "/api/v1/projects/:projectId/agents/:agentId/output", "/api/v1/projects/:projectId/agents/:agentId/output/stream", "/api/v1/projects/:projectId/agents/:agentId/steps", "/api/v1/projects/:projectId/agents/:agentId/steps/stream", "/api/v1/projects/:projectId/security-settings", "/api/v1/projects/:projectId/security-settings/secrets", "/api/v1/projects/:projectId/security-settings/secrets/resolve-for-runtime", "/api/v1/projects/:projectId/security-settings/secret-bindings", "/api/v1/projects/:projectId/security-settings/github-outbound-credential", "/api/v1/projects/:projectId/security-settings/github-outbound-credential/import-gh-cli", "/api/v1/projects/:projectId/security-settings/github-outbound-credential/retest", "/api/v1/projects/:projectId/hr-advisor", "/api/v1/projects/:projectId/hr-advisor/activate", "/api/v1/projects/:projectId/role-bindings", "/api/v1/projects/:projectId/role-bindings/:bindingId", "/api/v1/chat/projects/:projectId/conversations/stream":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverProject,
			resource:      "project",
			paramName:     "projectId",
			permission:    projectPermissionForPath(path, method),
			checkRequired: true,
		}, true
	case "/api/v1/projects/:projectId/security-settings/secrets/:secretId", "/api/v1/projects/:projectId/security-settings/secrets/:secretId/rotate", "/api/v1/projects/:projectId/security-settings/secrets/:secretId/disable", "/api/v1/projects/:projectId/security-settings/secret-bindings/:bindingId":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverProject,
			resource:      "project",
			paramName:     "projectId",
			permission:    projectPermissionForPath(path, method),
			checkRequired: true,
		}, true
	case "/api/v1/projects/:projectId/repos/:repoId", "/api/v1/projects/:projectId/tickets/:ticketId/repo-scopes/:scopeId":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverProject,
			resource:      "project",
			paramName:     "projectId",
			permission:    repoPermissionForPath(method),
			checkRequired: true,
		}, true
	case "/api/v1/skills/:skillId", "/api/v1/skills/:skillId/files", "/api/v1/skills/:skillId/history", "/api/v1/skills/:skillId/enable", "/api/v1/skills/:skillId/disable", "/api/v1/skills/:skillId/bind", "/api/v1/skills/:skillId/unbind":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverProject,
			resource:      "skill",
			paramName:     "skillId",
			permission:    skillPermissionForPath(path, method),
			checkRequired: true,
		}, true
	case "/api/v1/workflows/:workflowId", "/api/v1/workflows/:workflowId/impact", "/api/v1/workflows/:workflowId/harness", "/api/v1/workflows/:workflowId/harness/history", "/api/v1/workflows/:workflowId/retire", "/api/v1/workflows/:workflowId/replace-references", "/api/v1/workflows/:workflowId/skills/bind", "/api/v1/workflows/:workflowId/skills/unbind":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverProject,
			resource:      "workflow",
			paramName:     "workflowId",
			permission:    workflowPermissionForPath(path, method),
			checkRequired: true,
		}, true
	case "/api/v1/statuses/:statusId":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverProject,
			resource:      "status",
			paramName:     "statusId",
			permission:    statusPermissionForPath(method),
			checkRequired: true,
		}, true
	case "/api/v1/agents/:agentId", "/api/v1/agents/:agentId/interrupt", "/api/v1/agents/:agentId/pause", "/api/v1/agents/:agentId/resume", "/api/v1/agents/:agentId/retire":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverProject,
			resource:      "agent",
			paramName:     "agentId",
			permission:    agentPermissionForPath(path, method),
			checkRequired: true,
		}, true
	case "/api/v1/scheduled-jobs/:jobId", "/api/v1/scheduled-jobs/:jobId/trigger":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverProject,
			resource:      "scheduled_job",
			paramName:     "jobId",
			permission:    scheduledJobPermissionForPath(path, method),
			checkRequired: true,
		}, true
	case "/api/v1/notification-rules/:ruleId":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverProject,
			resource:      "notification_rule",
			paramName:     "ruleId",
			permission:    notificationPermissionForPath(method),
			checkRequired: true,
		}, true
	case "/api/v1/tickets/:ticketId", "/api/v1/tickets/:ticketId/comments", "/api/v1/tickets/:ticketId/comments/:commentId", "/api/v1/tickets/:ticketId/comments/:commentId/revisions", "/api/v1/tickets/:ticketId/dependencies", "/api/v1/tickets/:ticketId/dependencies/:dependencyId", "/api/v1/tickets/:ticketId/external-links", "/api/v1/tickets/:ticketId/external-links/:externalLinkId", "/api/v1/tickets/:ticketId/workspace/reset", "/api/v1/tickets/:ticketId/retry/resume":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverProject,
			resource:      "ticket",
			paramName:     "ticketId",
			permission:    ticketPermissionForPath(path, method),
			checkRequired: true,
		}, true
	case "/api/v1/chat/conversations/:conversationId", "/api/v1/chat/conversations/:conversationId/entries", "/api/v1/chat/conversations/:conversationId/stream", "/api/v1/chat/conversations/:conversationId/workspace-diff", "/api/v1/chat/conversations/:conversationId/turns", "/api/v1/chat/conversations/:conversationId/interrupts/:interruptId/respond", "/api/v1/chat/conversations/:conversationId/runtime":
		return humanRouteAuthorizationRule{
			scopeResolver: humanRouteScopeResolverProject,
			resource:      "conversation",
			paramName:     "conversationId",
			permission:    chatPermissionForPath(path, method),
			checkRequired: true,
		}, true
	default:
		return humanRouteAuthorizationRule{}, false
	}
}

func (s *Server) handleGetMyPermissions(c echo.Context) error {
	authContext, err := s.resolveAuthRequestContext(c, invalidHumanSessionAsError)
	if err != nil {
		return writeHumanSessionAuthError(c, err)
	}
	scope := humanauthdomain.ScopeRef{Kind: humanauthdomain.ScopeKindInstance, ID: ""}
	if projectID := strings.TrimSpace(c.QueryParam("project_id")); projectID != "" {
		scope = humanauthdomain.ScopeRef{Kind: humanauthdomain.ScopeKindProject, ID: projectID}
	}
	if orgID := strings.TrimSpace(c.QueryParam("org_id")); orgID != "" {
		scope = humanauthdomain.ScopeRef{Kind: humanauthdomain.ScopeKindOrganization, ID: orgID}
	}

	roles := authContext.Roles
	permissions := authContext.Permissions
	groups := authContext.Groups
	if authContext.PrincipalKind == authRequestPrincipalKindHumanSession {
		if s.humanAuthorizer == nil || authContext.HumanPrincipal == nil {
			return writeAPIError(c, http.StatusServiceUnavailable, "AUTHORIZATION_UNAVAILABLE", "authorization service unavailable")
		}
		roles, permissions, err = s.humanAuthorizer.Evaluate(
			c.Request().Context(),
			authContext.HumanPrincipal.User,
			authContext.HumanPrincipal.Identity,
			authContext.HumanPrincipal.Groups,
			scope,
		)
		if err != nil {
			return writeAPIError(c, http.StatusForbidden, "AUTHORIZATION_DENIED", err.Error())
		}
	}
	response := map[string]any{
		"auth_mode":                    authContext.RuntimeState.AuthMode.String(),
		"login_required":               authContext.LoginRequired,
		"authenticated":                authContext.Authenticated,
		"principal_kind":               string(authContext.PrincipalKind),
		"auth_configured":              authContext.AuthConfigured,
		"session_governance_available": authContext.SessionGovernanceAvailable,
		"can_manage_auth":              authContext.CanManageAuth,
		"scope": map[string]any{
			"kind": scope.Kind,
			"id":   scope.ID,
		},
		"roles":       roleKeysToStrings(roles),
		"permissions": permissionKeysToStrings(permissions),
		"groups":      groupMembershipsToResponse(groups),
	}
	if authContext.User != nil {
		response["user"] = authContext.User
	}
	return c.JSON(http.StatusOK, response)
}

func (s *Server) handleListOrganizationRoleBindings(c echo.Context) error {
	return s.handleListRoleBindings(c, humanauthdomain.ScopeKindOrganization, c.Param("orgId"))
}

func (s *Server) handleListInstanceRoleBindings(c echo.Context) error {
	return s.handleListRoleBindings(c, humanauthdomain.ScopeKindInstance, "")
}

func (s *Server) handleCreateInstanceRoleBinding(c echo.Context) error {
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", "human session required")
	}
	var raw rawCreateRoleBindingRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	expiresAt, err := parseRoleBindingExpiresAt(raw.ExpiresAt)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_EXPIRES_AT", "expires_at must be RFC3339")
	}
	item, err := s.humanAuthService.CreateInstanceRoleBinding(c.Request().Context(), humanauthservice.CreateRoleBindingInput{
		SubjectKind: raw.SubjectKind,
		SubjectKey:  raw.SubjectKey,
		RoleKey:     raw.RoleKey,
		GrantedBy:   principal.ActorID(),
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		if errors.Is(err, humanauthservice.ErrPermissionDenied) {
			return writeAPIError(c, http.StatusForbidden, "ROLE_BINDING_FORBIDDEN", err.Error())
		}
		return writeAPIError(c, http.StatusBadRequest, "ROLE_BINDING_CREATE_FAILED", err.Error())
	}
	return c.JSON(http.StatusCreated, map[string]any{"role_binding": mapRoleBindingResponse(item.Generic())})
}

func (s *Server) handleCreateOrganizationRoleBinding(c echo.Context) error {
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", "human session required")
	}
	organizationID, err := uuid.Parse(strings.TrimSpace(c.Param("orgId")))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_ORGANIZATION_ID", "organization id must be a UUID")
	}
	var raw rawCreateRoleBindingRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	expiresAt, err := parseRoleBindingExpiresAt(raw.ExpiresAt)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_EXPIRES_AT", "expires_at must be RFC3339")
	}
	item, err := s.humanAuthService.CreateOrganizationRoleBinding(c.Request().Context(), organizationID, principal, humanauthservice.CreateRoleBindingInput{
		SubjectKind: raw.SubjectKind,
		SubjectKey:  raw.SubjectKey,
		RoleKey:     raw.RoleKey,
		GrantedBy:   principal.ActorID(),
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		if errors.Is(err, humanauthservice.ErrPermissionDenied) {
			return writeAPIError(c, http.StatusForbidden, "ROLE_BINDING_FORBIDDEN", err.Error())
		}
		return writeAPIError(c, http.StatusBadRequest, "ROLE_BINDING_CREATE_FAILED", err.Error())
	}
	return c.JSON(http.StatusCreated, map[string]any{"role_binding": mapRoleBindingResponse(item.Generic())})
}

func (s *Server) handleDeleteOrganizationRoleBinding(c echo.Context) error {
	organizationID, err := uuid.Parse(strings.TrimSpace(c.Param("orgId")))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_ORGANIZATION_ID", "organization id must be a UUID")
	}
	return s.handleDeleteRoleBinding(c, organizationID, humanauthdomain.ScopeKindOrganization, c.Param("bindingId"))
}

func (s *Server) handleListProjectRoleBindings(c echo.Context) error {
	return s.handleListRoleBindings(c, humanauthdomain.ScopeKindProject, c.Param("projectId"))
}

func (s *Server) handleCreateProjectRoleBinding(c echo.Context) error {
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", "human session required")
	}
	projectID, err := uuid.Parse(strings.TrimSpace(c.Param("projectId")))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", "project id must be a UUID")
	}
	var raw rawCreateRoleBindingRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	expiresAt, err := parseRoleBindingExpiresAt(raw.ExpiresAt)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_EXPIRES_AT", "expires_at must be RFC3339")
	}
	item, err := s.humanAuthService.CreateProjectRoleBinding(c.Request().Context(), projectID, humanauthservice.CreateRoleBindingInput{
		SubjectKind: raw.SubjectKind,
		SubjectKey:  raw.SubjectKey,
		RoleKey:     raw.RoleKey,
		GrantedBy:   principal.ActorID(),
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "ROLE_BINDING_CREATE_FAILED", err.Error())
	}
	return c.JSON(http.StatusCreated, map[string]any{"role_binding": mapRoleBindingResponse(item.Generic())})
}

func (s *Server) handleDeleteProjectRoleBinding(c echo.Context) error {
	projectID, err := uuid.Parse(strings.TrimSpace(c.Param("projectId")))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", "project id must be a UUID")
	}
	return s.handleDeleteRoleBinding(c, projectID, humanauthdomain.ScopeKindProject, c.Param("bindingId"))
}

func (s *Server) handleListRoleBindings(c echo.Context, scopeKind humanauthdomain.ScopeKind, scopeID string) error {
	items, err := s.humanAuthService.ListRoleBindings(c.Request().Context(), scopeKind, scopeID)
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ROLE_BINDINGS_LIST_FAILED", err.Error())
	}
	response := make([]roleBindingResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapRoleBindingResponse(item))
	}
	return c.JSON(http.StatusOK, map[string]any{"role_bindings": response})
}

func (s *Server) handleDeleteInstanceRoleBinding(c echo.Context) error {
	return s.handleDeleteRoleBinding(c, uuid.Nil, humanauthdomain.ScopeKindInstance, c.Param("bindingId"))
}

func (s *Server) handleDeleteRoleBinding(
	c echo.Context,
	scopeResourceID uuid.UUID,
	scopeKind humanauthdomain.ScopeKind,
	bindingID string,
) error {
	parsed, err := uuid.Parse(strings.TrimSpace(bindingID))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_ROLE_BINDING_ID", "role binding id must be a UUID")
	}
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", "human session required")
	}
	switch scopeKind {
	case humanauthdomain.ScopeKindInstance:
		err = s.humanAuthService.DeleteInstanceRoleBinding(c.Request().Context(), parsed)
	case humanauthdomain.ScopeKindOrganization:
		err = s.humanAuthService.DeleteOrganizationRoleBinding(c.Request().Context(), scopeResourceID, principal, parsed)
	case humanauthdomain.ScopeKindProject:
		err = s.humanAuthService.DeleteProjectRoleBinding(c.Request().Context(), scopeResourceID, parsed)
	default:
		err = humanauthservice.ErrPermissionDenied
	}
	if err != nil {
		if err == humanauthservice.ErrRoleBindingNotFound {
			return writeAPIError(c, http.StatusNotFound, "ROLE_BINDING_NOT_FOUND", err.Error())
		}
		if errors.Is(err, humanauthservice.ErrPermissionDenied) {
			return writeAPIError(c, http.StatusForbidden, "ROLE_BINDING_FORBIDDEN", err.Error())
		}
		return writeAPIError(c, http.StatusBadRequest, "ROLE_BINDING_DELETE_FAILED", err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func parseRoleBindingExpiresAt(raw *string) (*time.Time, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*raw))
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func organizationPermissionForPath(path, method string) humanauthdomain.PermissionKey {
	if strings.Contains(path, "/role-bindings") {
		return humanauthdomain.PermissionRBACManage
	}
	switch {
	case strings.Contains(path, "/security-settings"):
		if method == http.MethodGet {
			return humanauthdomain.PermissionSecurityRead
		}
		return humanauthdomain.PermissionSecurityUpdate
	case strings.Contains(path, "/projects"):
		if method == http.MethodGet {
			return humanauthdomain.PermissionProjectRead
		}
		return humanauthdomain.PermissionProjectCreate
	case strings.Contains(path, "/machines"):
		if method == http.MethodGet {
			return humanauthdomain.PermissionMachineRead
		}
		return humanauthdomain.PermissionMachineCreate
	case strings.Contains(path, "/providers"):
		if method == http.MethodGet {
			return humanauthdomain.PermissionProviderRead
		}
		return humanauthdomain.PermissionProviderCreate
	case strings.Contains(path, "/channels"):
		if method == http.MethodGet {
			return humanauthdomain.PermissionNotificationRead
		}
		return humanauthdomain.PermissionNotificationCreate
	case method == http.MethodGet:
		return humanauthdomain.PermissionOrgRead
	case method == http.MethodDelete:
		return humanauthdomain.PermissionOrgDelete
	default:
		return humanauthdomain.PermissionOrgUpdate
	}
}

func projectPermissionForPath(path, method string) humanauthdomain.PermissionKey {
	switch {
	case strings.Contains(path, "/repos") || strings.Contains(path, "/github/"):
		if method == http.MethodGet {
			return humanauthdomain.PermissionRepoRead
		}
		return humanauthdomain.PermissionRepoCreate
	case strings.Contains(path, "/updates"):
		return projectUpdatePermissionForPath(method)
	case strings.Contains(path, "/notification-rules"):
		if method == http.MethodGet {
			return humanauthdomain.PermissionNotificationRead
		}
		return humanauthdomain.PermissionNotificationCreate
	case strings.Contains(path, "/scheduled-jobs"):
		if method == http.MethodGet {
			return humanauthdomain.PermissionJobRead
		}
		return humanauthdomain.PermissionJobCreate
	case strings.Contains(path, "/skills"):
		if method == http.MethodGet {
			return humanauthdomain.PermissionSkillRead
		}
		return humanauthdomain.PermissionSkillCreate
	case strings.Contains(path, "/workflows"):
		if method == http.MethodGet {
			return humanauthdomain.PermissionWorkflowRead
		}
		return humanauthdomain.PermissionWorkflowCreate
	case strings.Contains(path, "/statuses"):
		if method == http.MethodGet {
			return humanauthdomain.PermissionStatusRead
		}
		return humanauthdomain.PermissionStatusCreate
	case strings.Contains(path, "/tickets"):
		if method == http.MethodGet {
			return humanauthdomain.PermissionTicketRead
		}
		return humanauthdomain.PermissionTicketCreate
	case strings.Contains(path, "/security-settings"):
		if method == http.MethodGet {
			return humanauthdomain.PermissionSecurityRead
		}
		return humanauthdomain.PermissionSecurityUpdate
	case strings.Contains(path, "/agents") || strings.Contains(path, "/agent-runs"):
		if method == http.MethodGet {
			return humanauthdomain.PermissionAgentRead
		}
		return humanauthdomain.PermissionAgentCreate
	case strings.Contains(path, "/hr-advisor"):
		if method == http.MethodGet {
			return humanauthdomain.PermissionProjectRead
		}
		return humanauthdomain.PermissionProjectUpdate
	case strings.Contains(path, "/role-bindings"):
		return humanauthdomain.PermissionRBACManage
	default:
		if method == http.MethodGet {
			return humanauthdomain.PermissionProjectRead
		}
		if method == http.MethodDelete {
			return humanauthdomain.PermissionProjectDelete
		}
		return humanauthdomain.PermissionProjectUpdate
	}
}

func skillPermissionForPath(_ string, method string) humanauthdomain.PermissionKey {
	if method == http.MethodGet {
		return humanauthdomain.PermissionSkillRead
	}
	if method == http.MethodDelete {
		return humanauthdomain.PermissionSkillDelete
	}
	return humanauthdomain.PermissionSkillUpdate
}

func workflowPermissionForPath(path, method string) humanauthdomain.PermissionKey {
	switch {
	case strings.Contains(path, "/harness"):
		if method == http.MethodGet {
			return humanauthdomain.PermissionHarnessRead
		}
		return humanauthdomain.PermissionHarnessUpdate
	case strings.Contains(path, "/skills/"):
		return humanauthdomain.PermissionSkillUpdate
	case method == http.MethodGet:
		return humanauthdomain.PermissionWorkflowRead
	case method == http.MethodDelete:
		return humanauthdomain.PermissionWorkflowDelete
	default:
		return humanauthdomain.PermissionWorkflowUpdate
	}
}

func agentPermissionForPath(path, method string) humanauthdomain.PermissionKey {
	if method == http.MethodGet {
		return humanauthdomain.PermissionAgentRead
	}
	if method == http.MethodDelete {
		return humanauthdomain.PermissionAgentDelete
	}
	if strings.HasSuffix(path, "/interrupt") ||
		strings.HasSuffix(path, "/pause") ||
		strings.HasSuffix(path, "/resume") ||
		strings.HasSuffix(path, "/retire") {
		return humanauthdomain.PermissionAgentControl
	}
	return humanauthdomain.PermissionAgentUpdate
}

func ticketPermissionForPath(path, method string) humanauthdomain.PermissionKey {
	switch {
	case strings.Contains(path, "/comments"):
		switch method {
		case http.MethodGet:
			return humanauthdomain.PermissionTicketCommentRead
		case http.MethodDelete:
			return humanauthdomain.PermissionTicketCommentDelete
		case http.MethodPost:
			return humanauthdomain.PermissionTicketCommentCreate
		default:
			return humanauthdomain.PermissionTicketCommentUpdate
		}
	case strings.Contains(path, "/workspace/reset"), strings.Contains(path, "/retry/resume"), strings.Contains(path, "/dependencies"), strings.Contains(path, "/external-links"):
		return humanauthdomain.PermissionTicketUpdate
	default:
		if method == http.MethodGet {
			return humanauthdomain.PermissionTicketRead
		}
		if method == http.MethodDelete {
			return humanauthdomain.PermissionTicketDelete
		}
		return humanauthdomain.PermissionTicketUpdate
	}
}

func chatPermissionForPath(path, method string) humanauthdomain.PermissionKey {
	switch method {
	case http.MethodGet:
		return humanauthdomain.PermissionConversationRead
	case http.MethodDelete:
		return humanauthdomain.PermissionConversationDelete
	case http.MethodPost:
		if strings.HasSuffix(path, "/turns") || strings.Contains(path, "/interrupts/") {
			return humanauthdomain.PermissionConversationUpdate
		}
		return humanauthdomain.PermissionConversationCreate
	default:
		return humanauthdomain.PermissionConversationUpdate
	}
}

func machinePermissionForPath(path, method string) humanauthdomain.PermissionKey {
	if method == http.MethodGet {
		return humanauthdomain.PermissionMachineRead
	}
	if strings.HasSuffix(path, "/test") || strings.HasSuffix(path, "/refresh-health") {
		return humanauthdomain.PermissionMachineUpdate
	}
	return humanauthdomain.PermissionMachineUpdate
}

func providerPermissionForPath(method string) humanauthdomain.PermissionKey {
	if method == http.MethodGet {
		return humanauthdomain.PermissionProviderRead
	}
	if method == http.MethodDelete {
		return humanauthdomain.PermissionProviderDelete
	}
	return humanauthdomain.PermissionProviderUpdate
}

func channelPermissionForPath(path, method string) humanauthdomain.PermissionKey {
	if method == http.MethodGet {
		return humanauthdomain.PermissionNotificationRead
	}
	if method == http.MethodDelete {
		return humanauthdomain.PermissionNotificationDelete
	}
	if strings.HasSuffix(path, "/test") {
		return humanauthdomain.PermissionNotificationUpdate
	}
	return humanauthdomain.PermissionNotificationUpdate
}

func repoPermissionForPath(method string) humanauthdomain.PermissionKey {
	if method == http.MethodGet {
		return humanauthdomain.PermissionRepoRead
	}
	if method == http.MethodDelete {
		return humanauthdomain.PermissionRepoDelete
	}
	return humanauthdomain.PermissionRepoUpdate
}

func projectUpdatePermissionForPath(method string) humanauthdomain.PermissionKey {
	switch method {
	case http.MethodGet:
		return humanauthdomain.PermissionProjectUpdateRead
	case http.MethodDelete:
		return humanauthdomain.PermissionProjectUpdateDelete
	case http.MethodPost:
		return humanauthdomain.PermissionProjectUpdateCreate
	default:
		return humanauthdomain.PermissionProjectUpdateUpdate
	}
}

func statusPermissionForPath(method string) humanauthdomain.PermissionKey {
	switch method {
	case http.MethodGet:
		return humanauthdomain.PermissionStatusRead
	case http.MethodDelete:
		return humanauthdomain.PermissionStatusDelete
	default:
		return humanauthdomain.PermissionStatusUpdate
	}
}

func scheduledJobPermissionForPath(path, method string) humanauthdomain.PermissionKey {
	switch {
	case strings.HasSuffix(path, "/trigger"):
		return humanauthdomain.PermissionJobTrigger
	case method == http.MethodGet:
		return humanauthdomain.PermissionJobRead
	case method == http.MethodDelete:
		return humanauthdomain.PermissionJobDelete
	default:
		return humanauthdomain.PermissionJobUpdate
	}
}

func notificationPermissionForPath(method string) humanauthdomain.PermissionKey {
	switch method {
	case http.MethodGet:
		return humanauthdomain.PermissionNotificationRead
	case http.MethodDelete:
		return humanauthdomain.PermissionNotificationDelete
	default:
		return humanauthdomain.PermissionNotificationUpdate
	}
}

func mapRoleBindingResponse(item humanauthdomain.RoleBinding) roleBindingResponse {
	response := roleBindingResponse{
		ID:          item.ID.String(),
		ScopeKind:   string(item.ScopeKind),
		ScopeID:     item.ScopeID,
		SubjectKind: string(item.SubjectKind),
		SubjectKey:  item.SubjectKey,
		RoleKey:     string(item.RoleKey),
		GrantedBy:   item.GrantedBy,
		CreatedAt:   item.CreatedAt.UTC().Format(time.RFC3339),
	}
	if item.ExpiresAt != nil {
		value := item.ExpiresAt.UTC().Format(time.RFC3339)
		response.ExpiresAt = &value
	}
	return response
}

func roleKeysToStrings(values []humanauthdomain.RoleKey) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

func permissionKeysToStrings(values []humanauthdomain.PermissionKey) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

func groupMembershipsToResponse(values []humanauthdomain.UserGroupMembership) []map[string]any {
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		result = append(result, map[string]any{
			"group_key":  value.GroupKey,
			"group_name": value.GroupName,
			"issuer":     value.Issuer,
		})
	}
	return result
}

func parseUUIDStringUnsafe(raw string) uuid.UUID {
	parsed, _ := uuid.Parse(strings.TrimSpace(raw))
	return parsed
}

func (s *Server) requireHumanPermission(
	c echo.Context,
	scope humanauthdomain.ScopeRef,
	permission humanauthdomain.PermissionKey,
) error {
	runtimeState, err := s.currentRuntimeAccessControlState(c)
	if err != nil {
		return writeAuthRuntimeUnavailable(c, "AUTH_RUNTIME_STATE_FAILED", err)
	}
	if !runtimeState.LoginRequired {
		return nil
	}
	if s.humanAuthorizer == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "AUTHORIZATION_UNAVAILABLE", "authorization service unavailable")
	}
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", "human session required")
	}
	allowed, _, _, err := s.humanAuthorizer.HasPermission(c.Request().Context(), principal, scope, permission)
	if err != nil {
		return writeAPIError(c, http.StatusForbidden, "AUTHORIZATION_DENIED", err.Error())
	}
	if !allowed {
		return writeAPIError(c, http.StatusForbidden, "AUTHORIZATION_DENIED", "required permission is missing")
	}
	return nil
}
