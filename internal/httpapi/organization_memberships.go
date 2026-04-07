package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	"github.com/labstack/echo/v4"
)

type organizationMembershipUserResponse struct {
	ID           string `json:"id"`
	PrimaryEmail string `json:"primary_email"`
	DisplayName  string `json:"display_name"`
	AvatarURL    string `json:"avatar_url,omitempty"`
}

type organizationInvitationResponse struct {
	ID         string  `json:"id"`
	Status     string  `json:"status"`
	Email      string  `json:"email"`
	Role       string  `json:"role"`
	InvitedBy  string  `json:"invited_by"`
	SentAt     string  `json:"sent_at"`
	ExpiresAt  string  `json:"expires_at"`
	AcceptedAt *string `json:"accepted_at,omitempty"`
	CanceledAt *string `json:"canceled_at,omitempty"`
}

type organizationMembershipResponse struct {
	ID               string                              `json:"id"`
	OrganizationID   string                              `json:"organization_id"`
	UserID           *string                             `json:"user_id,omitempty"`
	Email            string                              `json:"email"`
	Role             string                              `json:"role"`
	Status           string                              `json:"status"`
	InvitedBy        string                              `json:"invited_by"`
	InvitedAt        string                              `json:"invited_at"`
	AcceptedAt       *string                             `json:"accepted_at,omitempty"`
	SuspendedAt      *string                             `json:"suspended_at,omitempty"`
	RemovedAt        *string                             `json:"removed_at,omitempty"`
	CreatedAt        string                              `json:"created_at"`
	UpdatedAt        string                              `json:"updated_at"`
	User             *organizationMembershipUserResponse `json:"user,omitempty"`
	ActiveInvitation *organizationInvitationResponse     `json:"active_invitation,omitempty"`
}

type inviteOrganizationMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type updateOrganizationMembershipRequest struct {
	Role   *string `json:"role"`
	Status *string `json:"status"`
}

type transferOrganizationOwnershipRequest struct {
	PreviousOwnerRole *string `json:"previous_owner_role"`
}

type acceptOrganizationInvitationRequest struct {
	Token string `json:"token"`
}

func (s *Server) registerOrganizationMembershipRoutes(api *echo.Group) {
	api.GET("/orgs/:orgId/members", s.handleListOrganizationMembers)
	api.POST("/orgs/:orgId/invitations", s.handleInviteOrganizationMember)
	api.POST("/orgs/:orgId/invitations/:invitationId/resend", s.handleResendOrganizationInvitation)
	api.POST("/orgs/:orgId/invitations/:invitationId/cancel", s.handleCancelOrganizationInvitation)
	api.POST("/org-invitations/accept", s.handleAcceptOrganizationInvitation)
	api.PATCH("/orgs/:orgId/members/:membershipId", s.handlePatchOrganizationMembership)
	api.POST("/orgs/:orgId/members/:membershipId/transfer-ownership", s.handleTransferOrganizationOwnership)
}

func (s *Server) handleListOrganizationMembers(c echo.Context) error {
	if s.humanAuthService == nil {
		return writeAPIError(c, http.StatusNotFound, "AUTH_DISABLED", "organization membership is only available when oidc auth is enabled")
	}
	organizationID, err := parseUUIDPathParamValue(c, "orgId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_ORGANIZATION_ID", err.Error())
	}
	entries, err := s.humanAuthService.ListOrganizationMembershipEntries(c.Request().Context(), organizationID)
	if err != nil {
		return writeOrganizationMembershipError(c, err)
	}
	response := make([]organizationMembershipResponse, 0, len(entries))
	for _, entry := range entries {
		response = append(response, mapOrganizationMembershipResponse(entry))
	}
	return c.JSON(http.StatusOK, map[string]any{"memberships": response})
}

func (s *Server) handleInviteOrganizationMember(c echo.Context) error {
	if s.humanAuthService == nil {
		return writeAPIError(c, http.StatusNotFound, "AUTH_DISABLED", "organization membership is only available when oidc auth is enabled")
	}
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", "human session required")
	}
	organizationID, err := parseUUIDPathParamValue(c, "orgId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_ORGANIZATION_ID", err.Error())
	}
	var request inviteOrganizationMemberRequest
	if err := decodeJSON(c, &request); err != nil {
		return err
	}
	result, err := s.humanAuthService.InviteOrganizationMember(c.Request().Context(), organizationID, principal, humanauthservice.InviteOrganizationMemberInput{
		Email: request.Email,
		Role:  request.Role,
	})
	if err != nil {
		return writeOrganizationMembershipError(c, err)
	}
	return c.JSON(http.StatusCreated, map[string]any{
		"membership":   mapOrganizationMembershipResponse(result.Entry),
		"invitation":   mapOrganizationInvitationResponse(result.Invitation),
		"accept_token": result.AcceptToken,
	})
}

func (s *Server) handleResendOrganizationInvitation(c echo.Context) error {
	if s.humanAuthService == nil {
		return writeAPIError(c, http.StatusNotFound, "AUTH_DISABLED", "organization membership is only available when oidc auth is enabled")
	}
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", "human session required")
	}
	organizationID, err := parseUUIDPathParamValue(c, "orgId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_ORGANIZATION_ID", err.Error())
	}
	invitationID, err := parseUUIDPathParamValue(c, "invitationId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_INVITATION_ID", err.Error())
	}
	result, err := s.humanAuthService.ResendOrganizationInvitation(c.Request().Context(), organizationID, invitationID, principal)
	if err != nil {
		return writeOrganizationMembershipError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"membership":   mapOrganizationMembershipResponse(result.Entry),
		"invitation":   mapOrganizationInvitationResponse(result.Invitation),
		"accept_token": result.AcceptToken,
	})
}

func (s *Server) handleCancelOrganizationInvitation(c echo.Context) error {
	if s.humanAuthService == nil {
		return writeAPIError(c, http.StatusNotFound, "AUTH_DISABLED", "organization membership is only available when oidc auth is enabled")
	}
	organizationID, err := parseUUIDPathParamValue(c, "orgId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_ORGANIZATION_ID", err.Error())
	}
	invitationID, err := parseUUIDPathParamValue(c, "invitationId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_INVITATION_ID", err.Error())
	}
	entry, err := s.humanAuthService.CancelOrganizationInvitation(c.Request().Context(), organizationID, invitationID)
	if err != nil {
		return writeOrganizationMembershipError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"membership": mapOrganizationMembershipResponse(entry)})
}

func (s *Server) handleAcceptOrganizationInvitation(c echo.Context) error {
	if s.humanAuthService == nil {
		return writeAPIError(c, http.StatusNotFound, "AUTH_DISABLED", "organization membership is only available when oidc auth is enabled")
	}
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", "human session required")
	}
	var request acceptOrganizationInvitationRequest
	if err := decodeJSON(c, &request); err != nil {
		return err
	}
	entry, err := s.humanAuthService.AcceptOrganizationInvitation(c.Request().Context(), principal, request.Token)
	if err != nil {
		return writeOrganizationMembershipError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"membership": mapOrganizationMembershipResponse(entry)})
}

func (s *Server) handlePatchOrganizationMembership(c echo.Context) error {
	if s.humanAuthService == nil {
		return writeAPIError(c, http.StatusNotFound, "AUTH_DISABLED", "organization membership is only available when oidc auth is enabled")
	}
	organizationID, err := parseUUIDPathParamValue(c, "orgId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_ORGANIZATION_ID", err.Error())
	}
	membershipID, err := parseUUIDPathParamValue(c, "membershipId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_MEMBERSHIP_ID", err.Error())
	}
	var request updateOrganizationMembershipRequest
	if err := decodeJSON(c, &request); err != nil {
		return err
	}
	entry, err := s.humanAuthService.UpdateOrganizationMembership(c.Request().Context(), organizationID, membershipID, humanauthservice.UpdateOrganizationMembershipInput{
		Role:   request.Role,
		Status: request.Status,
	})
	if err != nil {
		return writeOrganizationMembershipError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"membership": mapOrganizationMembershipResponse(entry)})
}

func (s *Server) handleTransferOrganizationOwnership(c echo.Context) error {
	if s.humanAuthService == nil {
		return writeAPIError(c, http.StatusNotFound, "AUTH_DISABLED", "organization membership is only available when oidc auth is enabled")
	}
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", "human session required")
	}
	organizationID, err := parseUUIDPathParamValue(c, "orgId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_ORGANIZATION_ID", err.Error())
	}
	membershipID, err := parseUUIDPathParamValue(c, "membershipId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_MEMBERSHIP_ID", err.Error())
	}
	var request transferOrganizationOwnershipRequest
	if err := decodeJSON(c, &request); err != nil && !errors.Is(err, errAPIResponseCommitted) {
		return err
	}
	previousOwnerRole := ""
	if request.PreviousOwnerRole != nil {
		previousOwnerRole = *request.PreviousOwnerRole
	}
	entries, err := s.humanAuthService.TransferOrganizationOwnership(c.Request().Context(), organizationID, membershipID, principal, humanauthservice.TransferOrganizationOwnershipInput{PreviousOwnerRole: previousOwnerRole})
	if err != nil {
		return writeOrganizationMembershipError(c, err)
	}
	response := make([]organizationMembershipResponse, 0, len(entries))
	for _, entry := range entries {
		response = append(response, mapOrganizationMembershipResponse(entry))
	}
	return c.JSON(http.StatusOK, map[string]any{"memberships": response})
}

func writeOrganizationMembershipError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, humanauthservice.ErrOrganizationMembershipNotFound):
		return writeAPIError(c, http.StatusNotFound, "ORGANIZATION_MEMBERSHIP_NOT_FOUND", err.Error())
	case errors.Is(err, humanauthservice.ErrOrganizationInvitationNotFound):
		return writeAPIError(c, http.StatusNotFound, "ORGANIZATION_INVITATION_NOT_FOUND", err.Error())
	case errors.Is(err, humanauthservice.ErrOrganizationInvitationExpired):
		return writeAPIError(c, http.StatusConflict, "ORGANIZATION_INVITATION_EXPIRED", err.Error())
	case errors.Is(err, humanauthservice.ErrOrganizationInvitationPending), errors.Is(err, humanauthservice.ErrOrganizationMemberExists), errors.Is(err, humanauthservice.ErrLastOrganizationOwner), errors.Is(err, humanauthservice.ErrOrganizationAcceptanceRequired):
		return writeAPIError(c, http.StatusConflict, "ORGANIZATION_MEMBERSHIP_CONFLICT", err.Error())
	case errors.Is(err, humanauthservice.ErrOrganizationInvitationMismatch), errors.Is(err, humanauthservice.ErrPermissionDenied):
		return writeAPIError(c, http.StatusForbidden, "ORGANIZATION_MEMBERSHIP_FORBIDDEN", err.Error())
	default:
		status := http.StatusBadRequest
		if !strings.Contains(strings.ToLower(err.Error()), "unsupported") && !strings.Contains(strings.ToLower(err.Error()), "required") {
			status = http.StatusInternalServerError
		}
		return writeAPIError(c, status, "ORGANIZATION_MEMBERSHIP_FAILED", err.Error())
	}
}

func mapOrganizationMembershipResponse(entry humanauthdomain.OrganizationMembershipEntry) organizationMembershipResponse {
	response := organizationMembershipResponse{
		ID:             entry.Membership.ID.String(),
		OrganizationID: entry.Membership.OrganizationID.String(),
		Email:          entry.Membership.Email,
		Role:           entry.Membership.Role.String(),
		Status:         entry.Membership.Status.String(),
		InvitedBy:      entry.Membership.InvitedBy,
		InvitedAt:      entry.Membership.InvitedAt.Format(time.RFC3339),
		CreatedAt:      entry.Membership.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      entry.Membership.UpdatedAt.Format(time.RFC3339),
	}
	if entry.Membership.UserID != nil {
		userID := entry.Membership.UserID.String()
		response.UserID = &userID
	}
	response.AcceptedAt = formatOptionalTime(entry.Membership.AcceptedAt)
	response.SuspendedAt = formatOptionalTime(entry.Membership.SuspendedAt)
	response.RemovedAt = formatOptionalTime(entry.Membership.RemovedAt)
	if entry.User != nil {
		response.User = &organizationMembershipUserResponse{
			ID:           entry.User.ID.String(),
			PrimaryEmail: entry.User.PrimaryEmail,
			DisplayName:  entry.User.DisplayName,
			AvatarURL:    entry.User.AvatarURL,
		}
	}
	if entry.ActiveInvitation != nil {
		mapped := mapOrganizationInvitationResponse(*entry.ActiveInvitation)
		response.ActiveInvitation = &mapped
	}
	return response
}

func mapOrganizationInvitationResponse(invitation humanauthdomain.OrganizationInvitation) organizationInvitationResponse {
	return organizationInvitationResponse{
		ID:         invitation.ID.String(),
		Status:     invitation.Status.String(),
		Email:      invitation.Email,
		Role:       invitation.Role.String(),
		InvitedBy:  invitation.InvitedBy,
		SentAt:     invitation.SentAt.Format(time.RFC3339),
		ExpiresAt:  invitation.ExpiresAt.Format(time.RFC3339),
		AcceptedAt: formatOptionalTime(invitation.AcceptedAt),
		CanceledAt: formatOptionalTime(invitation.CanceledAt),
	}
}
