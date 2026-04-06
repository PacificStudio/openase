package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	"github.com/labstack/echo/v4"
)

type userIdentitySummaryResponse struct {
	ID            string `json:"id"`
	Issuer        string `json:"issuer"`
	Subject       string `json:"subject"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	LastSyncedAt  string `json:"last_synced_at"`
}

type userDirectoryEntryResponse struct {
	ID              string                       `json:"id"`
	Status          string                       `json:"status"`
	PrimaryEmail    string                       `json:"primary_email"`
	DisplayName     string                       `json:"display_name"`
	AvatarURL       string                       `json:"avatar_url"`
	LastLoginAt     *string                      `json:"last_login_at,omitempty"`
	CreatedAt       string                       `json:"created_at"`
	UpdatedAt       string                       `json:"updated_at"`
	PrimaryIdentity *userIdentitySummaryResponse `json:"primary_identity,omitempty"`
}

type userStatusAuditResponse struct {
	Status              string `json:"status"`
	Reason              string `json:"reason"`
	Source              string `json:"source"`
	ActorID             string `json:"actor_id"`
	ChangedAt           string `json:"changed_at"`
	RevokedSessionCount int    `json:"revoked_session_count"`
}

type userGroupMembershipResponse struct {
	ID           string `json:"id"`
	Issuer       string `json:"issuer"`
	GroupKey     string `json:"group_key"`
	GroupName    string `json:"group_name"`
	LastSyncedAt string `json:"last_synced_at"`
}

type userIdentityDetailResponse struct {
	ID            string `json:"id"`
	Issuer        string `json:"issuer"`
	Subject       string `json:"subject"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	ClaimsVersion int    `json:"claims_version"`
	RawClaimsJSON string `json:"raw_claims_json"`
	LastSyncedAt  string `json:"last_synced_at"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type userDirectoryListResponse struct {
	Users []userDirectoryEntryResponse `json:"users"`
}

type userDirectoryDetailResponse struct {
	User               userDirectoryEntryResponse    `json:"user"`
	Identities         []userIdentityDetailResponse  `json:"identities"`
	Groups             []userGroupMembershipResponse `json:"groups"`
	ActiveSessionCount int                           `json:"active_session_count"`
	LatestStatusAudit  *userStatusAuditResponse      `json:"latest_status_audit,omitempty"`
	RecentAuditEvents  []authAuditEventResponse      `json:"recent_audit_events"`
}

type userStatusTransitionResponse struct {
	User                userDirectoryEntryResponse `json:"user"`
	Changed             bool                       `json:"changed"`
	RevokedSessionCount int                        `json:"revoked_session_count"`
	LatestStatusAudit   *userStatusAuditResponse   `json:"latest_status_audit,omitempty"`
}

type rawUserStatusTransitionRequest struct {
	Status         string `json:"status"`
	Reason         string `json:"reason"`
	RevokeSessions *bool  `json:"revoke_sessions,omitempty"`
}

func (s *Server) registerUserDirectoryRoutes(api *echo.Group) {
	api.GET("/instance/users", s.handleListUsers)
	api.GET("/instance/users/:userId", s.handleGetUser)
	api.POST("/instance/users/:userId/status", s.handleTransitionUserStatus)
}

func (s *Server) handleListUsers(c echo.Context) error {
	rawLimit := 0
	if limitValue := strings.TrimSpace(c.QueryParam("limit")); limitValue != "" {
		parsed, err := strconv.Atoi(limitValue)
		if err != nil {
			return writeAPIError(c, http.StatusBadRequest, "INVALID_USER_LIMIT", "limit must be a positive integer")
		}
		rawLimit = parsed
	}
	filter, err := humanauthdomain.NewUserDirectoryFilter(
		c.QueryParam("q"),
		c.QueryParam("status"),
		rawLimit,
	)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_USER_FILTER", err.Error())
	}

	entries, err := s.humanAuthService.ListUserDirectory(c.Request().Context(), filter)
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "USER_DIRECTORY_LIST_FAILED", err.Error())
	}
	response := userDirectoryListResponse{Users: make([]userDirectoryEntryResponse, 0, len(entries))}
	for _, entry := range entries {
		response.Users = append(response.Users, mapUserDirectoryEntryResponse(entry))
	}
	return c.JSON(http.StatusOK, response)
}

func (s *Server) handleGetUser(c echo.Context) error {
	userID, err := parseUUIDPathParamValue(c, "userId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_USER_ID", err.Error())
	}
	detail, err := s.humanAuthService.GetUserDirectoryDetail(c.Request().Context(), userID)
	if err != nil {
		if errors.Is(err, humanauthservice.ErrUserNotFound) {
			return writeAPIError(c, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		}
		return writeAPIError(c, http.StatusInternalServerError, "USER_DIRECTORY_DETAIL_FAILED", err.Error())
	}
	return c.JSON(http.StatusOK, mapUserDirectoryDetailResponse(detail))
}

func (s *Server) handleTransitionUserStatus(c echo.Context) error {
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", humanauthservice.ErrUnauthorized.Error())
	}
	userID, err := parseUUIDPathParamValue(c, "userId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_USER_ID", err.Error())
	}
	var raw rawUserStatusTransitionRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	status, err := humanauthdomain.ParseUserStatus(raw.Status)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_USER_STATUS", err.Error())
	}
	revokeSessions := status == humanauthdomain.UserStatusDisabled
	if raw.RevokeSessions != nil {
		revokeSessions = *raw.RevokeSessions
	}
	input, err := humanauthdomain.NewUserStatusTransitionInput(
		userID,
		status,
		raw.Reason,
		principal.ActorID(),
		humanauthdomain.UserStatusTransitionSourceAdminManual,
		revokeSessions,
	)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_USER_STATUS_TRANSITION", err.Error())
	}

	result, err := s.humanAuthService.TransitionUserStatus(c.Request().Context(), input)
	if err != nil {
		if errors.Is(err, humanauthservice.ErrUserNotFound) {
			return writeAPIError(c, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		}
		return writeAPIError(c, http.StatusBadRequest, "USER_STATUS_TRANSITION_FAILED", err.Error())
	}
	return c.JSON(http.StatusOK, userStatusTransitionResponse{
		User:                mapUserDirectoryEntryResponse(humanauthdomain.UserDirectoryEntry{User: result.User}),
		Changed:             result.Changed,
		RevokedSessionCount: result.RevokedSessionCount,
		LatestStatusAudit:   mapUserStatusAuditResponse(result.LatestStatusAudit),
	})
}

func mapUserDirectoryEntryResponse(entry humanauthdomain.UserDirectoryEntry) userDirectoryEntryResponse {
	response := userDirectoryEntryResponse{
		ID:           entry.User.ID.String(),
		Status:       string(entry.User.Status),
		PrimaryEmail: entry.User.PrimaryEmail,
		DisplayName:  entry.User.DisplayName,
		AvatarURL:    entry.User.AvatarURL,
		CreatedAt:    entry.User.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    entry.User.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if entry.User.LastLoginAt != nil {
		value := entry.User.LastLoginAt.UTC().Format(time.RFC3339)
		response.LastLoginAt = &value
	}
	if entry.PrimaryIdentity != nil {
		response.PrimaryIdentity = &userIdentitySummaryResponse{
			ID:            entry.PrimaryIdentity.ID.String(),
			Issuer:        entry.PrimaryIdentity.Issuer,
			Subject:       entry.PrimaryIdentity.Subject,
			Email:         entry.PrimaryIdentity.Email,
			EmailVerified: entry.PrimaryIdentity.EmailVerified,
			LastSyncedAt:  entry.PrimaryIdentity.LastSyncedAt.UTC().Format(time.RFC3339),
		}
	}
	return response
}

func mapUserDirectoryDetailResponse(detail humanauthdomain.UserDirectoryDetail) userDirectoryDetailResponse {
	response := userDirectoryDetailResponse{
		User:               mapUserDirectoryEntryResponse(humanauthdomain.UserDirectoryEntry{User: detail.User}),
		Identities:         make([]userIdentityDetailResponse, 0, len(detail.Identities)),
		Groups:             make([]userGroupMembershipResponse, 0, len(detail.Groups)),
		ActiveSessionCount: detail.ActiveSessionCount,
		LatestStatusAudit:  mapUserStatusAuditResponse(detail.LatestStatusAudit),
		RecentAuditEvents:  make([]authAuditEventResponse, 0, len(detail.RecentAuditEvents)),
	}
	for _, identity := range detail.Identities {
		response.Identities = append(response.Identities, userIdentityDetailResponse{
			ID:            identity.ID.String(),
			Issuer:        identity.Issuer,
			Subject:       identity.Subject,
			Email:         identity.Email,
			EmailVerified: identity.EmailVerified,
			ClaimsVersion: identity.ClaimsVersion,
			RawClaimsJSON: identity.RawClaimsJSON,
			LastSyncedAt:  identity.LastSyncedAt.UTC().Format(time.RFC3339),
			CreatedAt:     identity.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:     identity.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	for _, group := range detail.Groups {
		response.Groups = append(response.Groups, userGroupMembershipResponse{
			ID:           group.ID.String(),
			Issuer:       group.Issuer,
			GroupKey:     group.GroupKey,
			GroupName:    group.GroupName,
			LastSyncedAt: group.LastSyncedAt.UTC().Format(time.RFC3339),
		})
	}
	for _, event := range detail.RecentAuditEvents {
		response.RecentAuditEvents = append(response.RecentAuditEvents, mapAuthAuditEventResponse(event))
	}
	return response
}

func mapUserStatusAuditResponse(audit *humanauthdomain.UserStatusAudit) *userStatusAuditResponse {
	if audit == nil {
		return nil
	}
	return &userStatusAuditResponse{
		Status:              string(audit.Status),
		Reason:              audit.Reason,
		Source:              string(audit.Source),
		ActorID:             audit.ActorID,
		ChangedAt:           audit.ChangedAt.UTC().Format(time.RFC3339),
		RevokedSessionCount: audit.RevokedSessionCount,
	}
}
