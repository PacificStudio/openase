package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
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

type securityScopedSecretBindingSecretResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Scope       string  `json:"scope"`
	Kind        string  `json:"kind"`
	Description string  `json:"description"`
	ProjectID   *string `json:"project_id,omitempty"`
	Disabled    bool    `json:"disabled"`
}

type securityScopedSecretBindingTargetResponse struct {
	ID         string `json:"id"`
	Scope      string `json:"scope"`
	Name       string `json:"name"`
	Identifier string `json:"identifier,omitempty"`
}

type securityScopedSecretBindingResponse struct {
	ID              string                                    `json:"id"`
	OrganizationID  string                                    `json:"organization_id"`
	ProjectID       string                                    `json:"project_id"`
	SecretID        string                                    `json:"secret_id"`
	Scope           string                                    `json:"scope"`
	ScopeResourceID string                                    `json:"scope_resource_id"`
	BindingKey      string                                    `json:"binding_key"`
	CreatedAt       string                                    `json:"created_at"`
	UpdatedAt       string                                    `json:"updated_at"`
	Secret          securityScopedSecretBindingSecretResponse `json:"secret"`
	Target          securityScopedSecretBindingTargetResponse `json:"target"`
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
	if err := s.emitSecretActivity(
		c.Request().Context(),
		projectID,
		activityevent.TypeSecretCreated,
		"Created scoped secret "+item.Name,
		map[string]any{
			"secret_id":       item.ID.String(),
			"secret_name":     item.Name,
			"secret_scope":    string(item.Scope),
			"secret_kind":     string(item.Kind),
			"value_preview":   item.StoredValue.Preview,
			"changed_fields":  []string{"secret"},
			"operation_scope": "project",
		},
	); err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECRET_ACTIVITY_FAILED", err.Error())
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
	if err := s.emitOrganizationSecretActivity(
		c.Request().Context(),
		organizationID,
		activityevent.TypeSecretCreated,
		"Created organization secret "+item.Name,
		map[string]any{
			"secret_id":       item.ID.String(),
			"secret_name":     item.Name,
			"secret_scope":    string(item.Scope),
			"secret_kind":     string(item.Kind),
			"value_preview":   item.StoredValue.Preview,
			"changed_fields":  []string{"secret"},
			"operation_scope": "organization",
		},
	); err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECRET_ACTIVITY_FAILED", err.Error())
	}
	return c.JSON(http.StatusCreated, map[string]any{"secret": s.organizationSecretResponse(c, organizationID, item)})
}

func (s *Server) handleListScopedSecretBindings(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.secretService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", secretsservice.ErrUnavailable.Error())
	}
	items, err := s.secretService.ListProjectBindings(c.Request().Context(), projectID)
	if err != nil {
		return writeScopedSecretError(c, err)
	}
	response := make([]securityScopedSecretBindingResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapScopedSecretBindingResponse(item))
	}
	return c.JSON(http.StatusOK, map[string]any{"bindings": response})
}

func (s *Server) handleCreateScopedSecretBinding(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.secretService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", secretsservice.ErrUnavailable.Error())
	}
	var raw rawCreateScopedSecretBindingRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	input, err := parseCreateScopedSecretBindingRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	item, err := s.secretService.CreateBinding(c.Request().Context(), input)
	if err != nil {
		return writeScopedSecretError(c, err)
	}
	if err := s.emitSecretActivity(
		c.Request().Context(),
		projectID,
		activityevent.TypeSecretBound,
		"Bound secret "+item.Secret.Name+" to "+item.Target.Name,
		map[string]any{
			"binding_id":         item.Binding.ID.String(),
			"binding_key":        item.Binding.BindingKey,
			"binding_scope":      string(item.Binding.Scope),
			"scope_resource_id":  item.Binding.ScopeResourceID.String(),
			"secret_id":          item.Secret.ID.String(),
			"secret_name":        item.Secret.Name,
			"secret_scope":       string(item.Secret.Scope),
			"target_id":          item.Target.ID.String(),
			"target_name":        item.Target.Name,
			"target_scope":       string(item.Target.Scope),
			"target_identifier":  item.Target.Identifier,
			"changed_fields":     []string{"binding"},
			"operation_scope":    "project",
			"operation_resource": "runtime_binding",
		},
	); err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECRET_ACTIVITY_FAILED", err.Error())
	}
	return c.JSON(http.StatusCreated, map[string]any{"binding": mapScopedSecretBindingResponse(item)})
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

func (s *Server) handleDeleteScopedSecretBinding(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.secretService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", secretsservice.ErrUnavailable.Error())
	}
	bindingID, err := parseUUIDPathParam(c, "bindingId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SECRET_BINDING_ID", err.Error())
	}
	if err := s.secretService.DeleteBinding(c.Request().Context(), secretsservice.DeleteBindingInput{
		ProjectID: projectID,
		BindingID: bindingID,
	}); err != nil {
		return writeScopedSecretError(c, err)
	}
	if err := s.emitSecretUnboundActivity(c.Request().Context(), projectID, bindingID); err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECRET_ACTIVITY_FAILED", err.Error())
	}
	return c.NoContent(http.StatusNoContent)
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
	if err := s.emitSecretActivity(
		c.Request().Context(),
		projectID,
		activityevent.TypeSecretRotated,
		"Rotated scoped secret "+item.Name,
		map[string]any{
			"secret_id":       item.ID.String(),
			"secret_name":     item.Name,
			"secret_scope":    string(item.Scope),
			"value_preview":   item.StoredValue.Preview,
			"rotated_at":      item.StoredValue.RotatedAt.UTC().Format(time.RFC3339),
			"changed_fields":  []string{"value"},
			"operation_scope": "project",
		},
	); err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECRET_ACTIVITY_FAILED", err.Error())
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
	if err := s.emitSecretActivity(
		c.Request().Context(),
		projectID,
		activityevent.TypeSecretDisabled,
		"Disabled scoped secret "+item.Name,
		map[string]any{
			"secret_id":       item.ID.String(),
			"secret_name":     item.Name,
			"secret_scope":    string(item.Scope),
			"disabled_at":     timePointerRFC3339(item.DisabledAt),
			"changed_fields":  []string{"disabled"},
			"operation_scope": "project",
		},
	); err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECRET_ACTIVITY_FAILED", err.Error())
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
	if err := s.emitSecretDeletedActivity(c.Request().Context(), projectID, secretID); err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECRET_ACTIVITY_FAILED", err.Error())
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
	if err := s.emitOrganizationSecretActivity(
		c.Request().Context(),
		organizationID,
		activityevent.TypeSecretRotated,
		"Rotated organization secret "+item.Name,
		map[string]any{
			"secret_id":       item.ID.String(),
			"secret_name":     item.Name,
			"secret_scope":    string(item.Scope),
			"value_preview":   item.StoredValue.Preview,
			"rotated_at":      item.StoredValue.RotatedAt.UTC().Format(time.RFC3339),
			"changed_fields":  []string{"value"},
			"operation_scope": "organization",
		},
	); err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECRET_ACTIVITY_FAILED", err.Error())
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
	if err := s.emitOrganizationSecretActivity(
		c.Request().Context(),
		organizationID,
		activityevent.TypeSecretDisabled,
		"Disabled organization secret "+item.Name,
		map[string]any{
			"secret_id":       item.ID.String(),
			"secret_name":     item.Name,
			"secret_scope":    string(item.Scope),
			"disabled_at":     timePointerRFC3339(item.DisabledAt),
			"changed_fields":  []string{"disabled"},
			"operation_scope": "organization",
		},
	); err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECRET_ACTIVITY_FAILED", err.Error())
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
	if err := s.emitOrganizationSecretDeletedActivity(c.Request().Context(), organizationID, secretID); err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECRET_ACTIVITY_FAILED", err.Error())
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
			Value:        secretsdomain.RedactValue(item.Value),
		})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"resolved":     response,
		"missing_keys": missing,
	})
}

func (s *Server) emitSecretActivity(
	ctx context.Context,
	projectID uuid.UUID,
	eventType activityevent.Type,
	message string,
	metadata map[string]any,
) error {
	return s.emitActivity(ctx, activitysvc.RecordInput{
		ProjectID: projectID,
		EventType: eventType,
		Message:   message,
		Metadata:  cloneSecretActivityMetadata(metadata),
	})
}

func (s *Server) emitOrganizationSecretActivity(
	ctx context.Context,
	organizationID uuid.UUID,
	eventType activityevent.Type,
	message string,
	metadata map[string]any,
) error {
	if s == nil || s.catalog.Empty() || s.catalog.ProjectService == nil || organizationID == uuid.Nil {
		return nil
	}
	projects, err := s.catalog.ListProjects(ctx, organizationID)
	if err != nil {
		return err
	}
	for _, item := range projects {
		if err := s.emitSecretActivity(ctx, item.ID, eventType, message, metadata); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) emitSecretUnboundActivity(ctx context.Context, projectID uuid.UUID, bindingID uuid.UUID) error {
	if s == nil || s.secretService == nil {
		return nil
	}
	metadata := map[string]any{
		"binding_id":         bindingID.String(),
		"changed_fields":     []string{"binding"},
		"operation_scope":    "project",
		"operation_resource": "runtime_binding",
	}
	message := "Removed scoped secret binding"
	items, err := s.secretService.ListProjectBindings(ctx, projectID)
	if err == nil {
		for _, item := range items {
			if item.Binding.ID != bindingID {
				continue
			}
			message = "Unbound secret " + item.Secret.Name + " from " + item.Target.Name
			metadata["binding_key"] = item.Binding.BindingKey
			metadata["binding_scope"] = string(item.Binding.Scope)
			metadata["scope_resource_id"] = item.Binding.ScopeResourceID.String()
			metadata["secret_id"] = item.Secret.ID.String()
			metadata["secret_name"] = item.Secret.Name
			metadata["secret_scope"] = string(item.Secret.Scope)
			metadata["target_id"] = item.Target.ID.String()
			metadata["target_name"] = item.Target.Name
			metadata["target_scope"] = string(item.Target.Scope)
			metadata["target_identifier"] = item.Target.Identifier
			break
		}
	}
	return s.emitSecretActivity(ctx, projectID, activityevent.TypeSecretUnbound, message, metadata)
}

func (s *Server) emitSecretDeletedActivity(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID) error {
	if s == nil || s.secretService == nil {
		return nil
	}
	metadata := map[string]any{
		"secret_id":       secretID.String(),
		"changed_fields":  []string{"secret"},
		"operation_scope": "project",
	}
	message := "Deleted scoped secret"
	items, err := s.secretService.ListProjectSecretInventory(ctx, projectID)
	if err == nil {
		for _, item := range items {
			if item.Secret.ID != secretID {
				continue
			}
			message = "Deleted scoped secret " + item.Secret.Name
			metadata["secret_name"] = item.Secret.Name
			metadata["secret_scope"] = string(item.Secret.Scope)
			metadata["secret_kind"] = string(item.Secret.Kind)
			break
		}
	}
	return s.emitSecretActivity(ctx, projectID, activityevent.TypeSecretDeleted, message, metadata)
}

func (s *Server) emitOrganizationSecretDeletedActivity(
	ctx context.Context,
	organizationID uuid.UUID,
	secretID uuid.UUID,
) error {
	metadata := map[string]any{
		"secret_id":       secretID.String(),
		"changed_fields":  []string{"secret"},
		"operation_scope": "organization",
	}
	message := "Deleted organization secret"
	if s != nil && s.secretService != nil {
		items, err := s.secretService.ListOrganizationSecretInventory(ctx, organizationID)
		if err == nil {
			for _, item := range items {
				if item.Secret.ID != secretID {
					continue
				}
				message = "Deleted organization secret " + item.Secret.Name
				metadata["secret_name"] = item.Secret.Name
				metadata["secret_scope"] = string(item.Secret.Scope)
				metadata["secret_kind"] = string(item.Secret.Kind)
				break
			}
		}
	}
	return s.emitOrganizationSecretActivity(ctx, organizationID, activityevent.TypeSecretDeleted, message, metadata)
}

func cloneSecretActivityMetadata(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = value
	}
	return cloned
}

func timePointerRFC3339(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
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

func mapScopedSecretBindingResponse(item secretsdomain.BindingRecord) securityScopedSecretBindingResponse {
	response := securityScopedSecretBindingResponse{
		ID:              item.Binding.ID.String(),
		OrganizationID:  item.Binding.OrganizationID.String(),
		ProjectID:       item.Binding.ProjectID.String(),
		SecretID:        item.Binding.SecretID.String(),
		Scope:           string(item.Binding.Scope),
		ScopeResourceID: item.Binding.ScopeResourceID.String(),
		BindingKey:      item.Binding.BindingKey,
		CreatedAt:       item.Binding.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       item.Binding.UpdatedAt.UTC().Format(time.RFC3339),
		Secret: securityScopedSecretBindingSecretResponse{
			ID:          item.Secret.ID.String(),
			Name:        item.Secret.Name,
			Scope:       string(item.Secret.Scope),
			Kind:        string(item.Secret.Kind),
			Description: item.Secret.Description,
			Disabled:    item.Secret.DisabledAt != nil,
		},
		Target: securityScopedSecretBindingTargetResponse{
			ID:    item.Target.ID.String(),
			Scope: string(item.Target.Scope),
			Name:  item.Target.Name,
		},
	}
	if item.Secret.ProjectID != uuid.Nil {
		projectID := item.Secret.ProjectID.String()
		response.Secret.ProjectID = &projectID
	}
	if item.Target.Identifier != "" {
		response.Target.Identifier = item.Target.Identifier
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
	case errors.Is(err, secretsservice.ErrBindingNotFound):
		return writeAPIError(c, http.StatusNotFound, "SECRET_BINDING_NOT_FOUND", err.Error())
	case errors.Is(err, secretsservice.ErrBindingConflict):
		return writeAPIError(c, http.StatusConflict, "SECRET_BINDING_CONFLICT", err.Error())
	case errors.Is(err, secretsservice.ErrBindingTarget):
		return writeAPIError(c, http.StatusBadRequest, "SECRET_BINDING_TARGET_NOT_FOUND", err.Error())
	case errors.Is(err, secretsdomain.ErrResolutionScopeConflict):
		return writeAPIError(c, http.StatusConflict, "SECRET_BINDING_CONFLICT", err.Error())
	default:
		return writeAPIError(c, http.StatusInternalServerError, "SECRET_OPERATION_FAILED", err.Error())
	}
}
