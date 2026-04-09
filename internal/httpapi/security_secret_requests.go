package httpapi

import (
	secretsservice "github.com/BetterAndBetterII/openase/internal/service/secrets"
	"github.com/google/uuid"
)

type rawCreateScopedSecretRequest struct {
	Scope       string `json:"scope"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Value       string `json:"value"`
}

type rawCreateScopedSecretBindingRequest struct {
	SecretID        string `json:"secret_id"`
	Scope           string `json:"scope"`
	ScopeResourceID string `json:"scope_resource_id"`
	BindingKey      string `json:"binding_key"`
}

type rawPatchScopedSecretRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type rawRotateScopedSecretRequest struct {
	Value string `json:"value"`
}

type rawResolveScopedSecretsRequest struct {
	BindingKeys []string `json:"binding_keys"`
	TicketID    *string  `json:"ticket_id"`
	WorkflowID  *string  `json:"workflow_id"`
	AgentID     *string  `json:"agent_id"`
}

func parseCreateScopedSecretRequest(projectID uuid.UUID, raw rawCreateScopedSecretRequest) secretsservice.CreateSecretInput {
	return secretsservice.CreateSecretInput{
		ProjectID:   projectID,
		Scope:       raw.Scope,
		Name:        raw.Name,
		Kind:        raw.Kind,
		Description: raw.Description,
		Value:       raw.Value,
	}
}

func parseCreateScopedSecretBindingRequest(projectID uuid.UUID, raw rawCreateScopedSecretBindingRequest) (secretsservice.CreateBindingInput, error) {
	secretID, err := parseUUIDString("secret_id", raw.SecretID)
	if err != nil {
		return secretsservice.CreateBindingInput{}, err
	}
	scopeResourceID, err := parseUUIDString("scope_resource_id", raw.ScopeResourceID)
	if err != nil {
		return secretsservice.CreateBindingInput{}, err
	}
	return secretsservice.CreateBindingInput{
		ProjectID:       projectID,
		SecretID:        secretID,
		Scope:           raw.Scope,
		ScopeResourceID: scopeResourceID,
		BindingKey:      raw.BindingKey,
	}, nil
}

func parseCreateOrganizationScopedSecretRequest(organizationID uuid.UUID, raw rawCreateScopedSecretRequest) secretsservice.CreateOrganizationSecretInput {
	return secretsservice.CreateOrganizationSecretInput{
		OrganizationID: organizationID,
		Name:           raw.Name,
		Kind:           raw.Kind,
		Description:    raw.Description,
		Value:          raw.Value,
	}
}

func parsePatchScopedSecretRequest(projectID uuid.UUID, secretID uuid.UUID, raw rawPatchScopedSecretRequest) secretsservice.UpdateSecretMetadataInput {
	return secretsservice.UpdateSecretMetadataInput{
		ProjectID:   projectID,
		SecretID:    secretID,
		Name:        raw.Name,
		Description: raw.Description,
	}
}

func parseRotateScopedSecretRequest(projectID uuid.UUID, secretID uuid.UUID, raw rawRotateScopedSecretRequest) secretsservice.RotateSecretInput {
	return secretsservice.RotateSecretInput{
		ProjectID: projectID,
		SecretID:  secretID,
		Value:     raw.Value,
	}
}

func parseResolveScopedSecretsRequest(projectID uuid.UUID, raw rawResolveScopedSecretsRequest) (secretsservice.ResolveRuntimeInput, error) {
	ticketID, err := parseOptionalUUIDString("ticket_id", raw.TicketID)
	if err != nil {
		return secretsservice.ResolveRuntimeInput{}, err
	}
	workflowID, err := parseOptionalUUIDString("workflow_id", raw.WorkflowID)
	if err != nil {
		return secretsservice.ResolveRuntimeInput{}, err
	}
	agentID, err := parseOptionalUUIDString("agent_id", raw.AgentID)
	if err != nil {
		return secretsservice.ResolveRuntimeInput{}, err
	}
	return secretsservice.ResolveRuntimeInput{
		ProjectID:   projectID,
		BindingKeys: cloneStringSlice(raw.BindingKeys),
		TicketID:    ticketID,
		WorkflowID:  workflowID,
		AgentID:     agentID,
	}, nil
}
