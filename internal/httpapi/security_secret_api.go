package httpapi

import (
	"errors"
	"net/http"
	"time"

	secretsdomain "github.com/BetterAndBetterII/openase/internal/domain/secrets"
	secretsservice "github.com/BetterAndBetterII/openase/internal/service/secrets"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type securityScopedSecretEncryptionResponse struct {
	Algorithm    string `json:"algorithm"`
	KeySource    string `json:"key_source"`
	KeyID        string `json:"key_id"`
	ValuePreview string `json:"value_preview"`
	RotatedAt    string `json:"rotated_at"`
}

type securityScopedSecretResponse struct {
	ID             string                                 `json:"id"`
	OrganizationID string                                 `json:"organization_id"`
	ProjectID      *string                                `json:"project_id,omitempty"`
	Scope          string                                 `json:"scope"`
	Name           string                                 `json:"name"`
	Kind           string                                 `json:"kind"`
	Description    string                                 `json:"description"`
	Disabled       bool                                   `json:"disabled"`
	DisabledAt     *string                                `json:"disabled_at,omitempty"`
	CreatedAt      string                                 `json:"created_at"`
	UpdatedAt      string                                 `json:"updated_at"`
	UsageCount     int                                    `json:"usage_count"`
	UsageScopes    []string                               `json:"usage_scopes,omitempty"`
	Encryption     securityScopedSecretEncryptionResponse `json:"encryption"`
}

type securityResolvedRuntimeSecretResponse struct {
	BindingKey   string `json:"binding_key"`
	BindingScope string `json:"binding_scope"`
	SecretID     string `json:"secret_id"`
	SecretName   string `json:"secret_name"`
	SecretScope  string `json:"secret_scope"`
	SecretKind   string `json:"secret_kind"`
	Value        string `json:"value"`
}

func (s *Server) handleListScopedSecrets(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.secretService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", secretsservice.ErrUnavailable.Error())
	}
	items, err := s.secretService.ListProjectSecretInventory(c.Request().Context(), projectID)
	if err != nil {
		return writeScopedSecretError(c, err)
	}
	response := make([]securityScopedSecretResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapScopedSecretInventoryResponse(item))
	}
	return c.JSON(http.StatusOK, map[string]any{"secrets": response})
}

func (s *Server) handleListOrganizationScopedSecrets(c echo.Context) error {
	organizationID, err := s.requireOrganizationSecurityContext(c)
	if err != nil {
		return err
	}
	if s.secretService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", secretsservice.ErrUnavailable.Error())
	}
	items, err := s.secretService.ListOrganizationSecretInventory(c.Request().Context(), organizationID)
	if err != nil {
		return writeScopedSecretError(c, err)
	}
	response := make([]securityScopedSecretResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapScopedSecretInventoryResponse(item))
	}
	return c.JSON(http.StatusOK, map[string]any{"secrets": response})
}

func (s *Server) handleCreateScopedSecret(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.secretService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", secretsservice.ErrUnavailable.Error())
	}
	var raw rawCreateScopedSecretRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	input := parseCreateScopedSecretRequest(projectID, raw)
	item, err := s.secretService.CreateSecret(c.Request().Context(), input)
	if err != nil {
		return writeScopedSecretError(c, err)
	}
	return c.JSON(http.StatusCreated, map[string]any{"secret": s.projectSecretResponse(c, projectID, item)})
}

func (s *Server) handleCreateOrganizationScopedSecret(c echo.Context) error {
	organizationID, err := s.requireOrganizationSecurityContext(c)
	if err != nil {
		return err
	}
	if s.secretService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", secretsservice.ErrUnavailable.Error())
	}
	var raw rawCreateScopedSecretRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	item, err := s.secretService.CreateOrganizationSecret(c.Request().Context(), parseCreateOrganizationScopedSecretRequest(organizationID, raw))
	if err != nil {
		return writeScopedSecretError(c, err)
	}
	return c.JSON(http.StatusCreated, map[string]any{"secret": s.organizationSecretResponse(c, organizationID, item)})
}

func (s *Server) handlePatchScopedSecret(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.secretService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", secretsservice.ErrUnavailable.Error())
	}
	secretID, err := parseUUIDPathParam(c, "secretId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SECRET_ID", err.Error())
	}
	var raw rawPatchScopedSecretRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	item, err := s.secretService.UpdateSecretMetadata(c.Request().Context(), parsePatchScopedSecretRequest(projectID, secretID, raw))
	if err != nil {
		return writeScopedSecretError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"secret": s.projectSecretResponse(c, projectID, item)})
}

func (s *Server) handleRotateScopedSecret(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.secretService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", secretsservice.ErrUnavailable.Error())
	}
	secretID, err := parseUUIDPathParam(c, "secretId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SECRET_ID", err.Error())
	}
	var raw rawRotateScopedSecretRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	item, err := s.secretService.RotateSecret(c.Request().Context(), parseRotateScopedSecretRequest(projectID, secretID, raw))
	if err != nil {
		return writeScopedSecretError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"secret": s.projectSecretResponse(c, projectID, item)})
}

func (s *Server) handleDisableScopedSecret(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.secretService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", secretsservice.ErrUnavailable.Error())
	}
	secretID, err := parseUUIDPathParam(c, "secretId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SECRET_ID", err.Error())
	}
	item, err := s.secretService.DisableSecret(c.Request().Context(), secretsservice.DisableSecretInput{ProjectID: projectID, SecretID: secretID})
	if err != nil {
		return writeScopedSecretError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"secret": s.projectSecretResponse(c, projectID, item)})
}

func (s *Server) handleDeleteScopedSecret(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.secretService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", secretsservice.ErrUnavailable.Error())
	}
	secretID, err := parseUUIDPathParam(c, "secretId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SECRET_ID", err.Error())
	}
	if err := s.secretService.DeleteSecret(c.Request().Context(), secretsservice.DeleteSecretInput{ProjectID: projectID, SecretID: secretID}); err != nil {
		return writeScopedSecretError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleRotateOrganizationScopedSecret(c echo.Context) error {
	organizationID, err := s.requireOrganizationSecurityContext(c)
	if err != nil {
		return err
	}
	if s.secretService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", secretsservice.ErrUnavailable.Error())
	}
	secretID, err := parseUUIDPathParam(c, "secretId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SECRET_ID", err.Error())
	}
	var raw rawRotateScopedSecretRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	item, err := s.secretService.RotateOrganizationSecret(c.Request().Context(), secretsservice.RotateOrganizationSecretInput{
		OrganizationID: organizationID,
		SecretID:       secretID,
		Value:          raw.Value,
	})
	if err != nil {
		return writeScopedSecretError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"secret": s.organizationSecretResponse(c, organizationID, item)})
}

func (s *Server) handleDisableOrganizationScopedSecret(c echo.Context) error {
	organizationID, err := s.requireOrganizationSecurityContext(c)
	if err != nil {
		return err
	}
	if s.secretService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", secretsservice.ErrUnavailable.Error())
	}
	secretID, err := parseUUIDPathParam(c, "secretId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SECRET_ID", err.Error())
	}
	item, err := s.secretService.DisableOrganizationSecret(c.Request().Context(), secretsservice.DisableOrganizationSecretInput{
		OrganizationID: organizationID,
		SecretID:       secretID,
	})
	if err != nil {
		return writeScopedSecretError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"secret": s.organizationSecretResponse(c, organizationID, item)})
}

func (s *Server) handleDeleteOrganizationScopedSecret(c echo.Context) error {
	organizationID, err := s.requireOrganizationSecurityContext(c)
	if err != nil {
		return err
	}
	if s.secretService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", secretsservice.ErrUnavailable.Error())
	}
	secretID, err := parseUUIDPathParam(c, "secretId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SECRET_ID", err.Error())
	}
	if err := s.secretService.DeleteOrganizationSecret(c.Request().Context(), secretsservice.DeleteOrganizationSecretInput{
		OrganizationID: organizationID,
		SecretID:       secretID,
	}); err != nil {
		return writeScopedSecretError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleResolveScopedSecretsForRuntime(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.secretService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", secretsservice.ErrUnavailable.Error())
	}
	var raw rawResolveScopedSecretsRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	input, err := parseResolveScopedSecretsRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	resolved, missing, err := s.secretService.ResolveForRuntime(c.Request().Context(), input)
	if err != nil {
		return writeScopedSecretError(c, err)
	}
	response := make([]securityResolvedRuntimeSecretResponse, 0, len(resolved))
	for _, item := range resolved {
		response = append(response, securityResolvedRuntimeSecretResponse{
			BindingKey:   item.BindingKey,
			BindingScope: string(item.BindingScope),
			SecretID:     item.SecretID.String(),
			SecretName:   item.SecretName,
			SecretScope:  string(item.SecretScope),
			SecretKind:   string(item.SecretKind),
			Value:        item.Value,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"resolved":     response,
		"missing_keys": missing,
	})
}

func mapScopedSecretResponse(item secretsdomain.Secret) securityScopedSecretResponse {
	response := securityScopedSecretResponse{
		ID:             item.ID.String(),
		OrganizationID: item.OrganizationID.String(),
		Scope:          string(item.Scope),
		Name:           item.Name,
		Kind:           string(item.Kind),
		Description:    item.Description,
		Disabled:       item.DisabledAt != nil,
		CreatedAt:      item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.UTC().Format(time.RFC3339),
		Encryption: securityScopedSecretEncryptionResponse{
			Algorithm:    item.StoredValue.Algorithm,
			KeySource:    string(item.StoredValue.KeySource),
			KeyID:        item.StoredValue.KeyID,
			ValuePreview: item.StoredValue.Preview,
			RotatedAt:    item.StoredValue.RotatedAt.UTC().Format(time.RFC3339),
		},
	}
	if item.ProjectID != uuid.Nil {
		projectID := item.ProjectID.String()
		response.ProjectID = &projectID
	}
	if item.DisabledAt != nil {
		disabledAt := item.DisabledAt.UTC().Format(time.RFC3339)
		response.DisabledAt = &disabledAt
	}
	return response
}

func mapScopedSecretInventoryResponse(item secretsdomain.InventorySecret) securityScopedSecretResponse {
	response := mapScopedSecretResponse(item.Secret)
	response.UsageCount = item.UsageCount
	if len(item.UsageScopes) > 0 {
		response.UsageScopes = make([]string, 0, len(item.UsageScopes))
		for _, scope := range item.UsageScopes {
			response.UsageScopes = append(response.UsageScopes, string(scope))
		}
	}
	return response
}

func (s *Server) projectSecretResponse(c echo.Context, projectID uuid.UUID, item secretsdomain.Secret) securityScopedSecretResponse {
	inventory, err := s.secretService.ListProjectSecretInventory(c.Request().Context(), projectID)
	if err != nil {
		return mapScopedSecretResponse(item)
	}
	return matchScopedSecretInventoryResponse(inventory, item)
}

func (s *Server) organizationSecretResponse(c echo.Context, organizationID uuid.UUID, item secretsdomain.Secret) securityScopedSecretResponse {
	inventory, err := s.secretService.ListOrganizationSecretInventory(c.Request().Context(), organizationID)
	if err != nil {
		return mapScopedSecretResponse(item)
	}
	return matchScopedSecretInventoryResponse(inventory, item)
}

func matchScopedSecretInventoryResponse(inventory []secretsdomain.InventorySecret, item secretsdomain.Secret) securityScopedSecretResponse {
	for _, candidate := range inventory {
		if candidate.Secret.ID == item.ID {
			return mapScopedSecretInventoryResponse(candidate)
		}
	}
	return mapScopedSecretResponse(item)
}

func writeScopedSecretError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, secretsservice.ErrUnavailable):
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	case errors.Is(err, secretsservice.ErrInvalidInput):
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, secretsservice.ErrSecretNotFound):
		return writeAPIError(c, http.StatusNotFound, "SECRET_NOT_FOUND", err.Error())
	case errors.Is(err, secretsservice.ErrSecretNameConflict):
		return writeAPIError(c, http.StatusConflict, "SECRET_NAME_CONFLICT", err.Error())
	case errors.Is(err, secretsdomain.ErrResolutionScopeConflict):
		return writeAPIError(c, http.StatusConflict, "SECRET_BINDING_CONFLICT", err.Error())
	default:
		return writeAPIError(c, http.StatusInternalServerError, "SECRET_OPERATION_FAILED", err.Error())
	}
}
