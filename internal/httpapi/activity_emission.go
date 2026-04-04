package httpapi

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
)

func (s *Server) emitActivity(ctx context.Context, input activitysvc.RecordInput) error {
	if s == nil || s.activityEmitter == nil {
		return nil
	}
	if _, err := s.activityEmitter.Emit(ctx, input); err != nil {
		return err
	}
	return nil
}

func (s *Server) emitActivities(ctx context.Context, inputs ...activitysvc.RecordInput) error {
	for _, input := range inputs {
		if err := s.emitActivity(ctx, input); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) emitProviderActivityForAffectedProjects(
	ctx context.Context,
	organizationID uuid.UUID,
	providerID uuid.UUID,
	build func(projectID uuid.UUID) activitysvc.RecordInput,
) error {
	projectIDs, err := s.affectedProviderProjectIDs(ctx, organizationID, providerID)
	if err != nil {
		return err
	}
	for _, projectID := range projectIDs {
		if err := s.emitActivity(ctx, build(projectID)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) affectedProviderProjectIDs(
	ctx context.Context,
	organizationID uuid.UUID,
	providerID uuid.UUID,
) ([]uuid.UUID, error) {
	if s == nil || s.catalog.Empty() {
		return nil, nil
	}

	projects, err := s.catalog.ListProjects(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list projects for provider activity: %w", err)
	}

	projectIDs := make([]uuid.UUID, 0, len(projects))
	for _, project := range projects {
		if project.DefaultAgentProviderID != nil && *project.DefaultAgentProviderID == providerID {
			projectIDs = append(projectIDs, project.ID)
			continue
		}

		agents, err := s.catalog.ListAgents(ctx, project.ID)
		if err != nil {
			return nil, fmt.Errorf("list agents for provider activity: %w", err)
		}
		if slices.ContainsFunc(agents, func(item domain.Agent) bool {
			return item.ProviderID == providerID
		}) {
			projectIDs = append(projectIDs, project.ID)
		}
	}

	slices.SortFunc(projectIDs, func(left uuid.UUID, right uuid.UUID) int {
		return strings.Compare(left.String(), right.String())
	})
	return slices.Compact(projectIDs), nil
}

func (s *Server) emitMachineActivityForAffectedProjects(
	ctx context.Context,
	organizationID uuid.UUID,
	machineID uuid.UUID,
	build func(projectID uuid.UUID) activitysvc.RecordInput,
) error {
	projectIDs, err := s.affectedMachineProjectIDs(ctx, organizationID, machineID)
	if err != nil {
		return err
	}
	for _, projectID := range projectIDs {
		if err := s.emitActivity(ctx, build(projectID)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) affectedMachineProjectIDs(
	ctx context.Context,
	organizationID uuid.UUID,
	machineID uuid.UUID,
) ([]uuid.UUID, error) {
	if s == nil || s.catalog.Empty() || s.catalog.ProjectService == nil {
		return nil, nil
	}

	projects, err := s.catalog.ListProjects(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list projects for machine activity: %w", err)
	}

	providerIDs := map[uuid.UUID]struct{}{}
	if s.catalog.AgentProviderService != nil {
		providers, err := s.catalog.ListAgentProviders(ctx, organizationID)
		if err != nil {
			return nil, fmt.Errorf("list providers for machine activity: %w", err)
		}
		for _, provider := range providers {
			if provider.MachineID == machineID {
				providerIDs[provider.ID] = struct{}{}
			}
		}
	}

	projectIDs := make([]uuid.UUID, 0, len(projects))
	for _, project := range projects {
		if slices.Contains(project.AccessibleMachineIDs, machineID) {
			projectIDs = append(projectIDs, project.ID)
			continue
		}
		if project.DefaultAgentProviderID != nil {
			if _, ok := providerIDs[*project.DefaultAgentProviderID]; ok {
				projectIDs = append(projectIDs, project.ID)
				continue
			}
		}
		if len(providerIDs) == 0 || s.catalog.AgentService == nil {
			continue
		}
		agents, err := s.catalog.ListAgents(ctx, project.ID)
		if err != nil {
			return nil, fmt.Errorf("list agents for machine activity: %w", err)
		}
		if slices.ContainsFunc(agents, func(item domain.Agent) bool {
			_, ok := providerIDs[item.ProviderID]
			return ok
		}) {
			projectIDs = append(projectIDs, project.ID)
		}
	}

	slices.SortFunc(projectIDs, func(left uuid.UUID, right uuid.UUID) int {
		return strings.Compare(left.String(), right.String())
	})
	return slices.Compact(projectIDs), nil
}

func ticketStatusChangedFields(raw rawUpdateTicketStatusRequest) []string {
	fields := make([]string, 0, 7)
	if raw.Name != nil {
		fields = append(fields, "name")
	}
	if raw.Stage != nil {
		fields = append(fields, "stage")
	}
	if raw.Color != nil {
		fields = append(fields, "color")
	}
	if raw.Icon != nil {
		fields = append(fields, "icon")
	}
	if raw.IsDefault != nil {
		fields = append(fields, "is_default")
	}
	if raw.Description != nil {
		fields = append(fields, "description")
	}
	return fields
}

func mapStatusNames(items []ticketstatus.Status) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}

func projectStatusMetadata(before domain.Project, after domain.Project) map[string]any {
	return map[string]any{
		"project_name":   after.Name,
		"from_status":    before.Status.String(),
		"to_status":      after.Status.String(),
		"changed_fields": []string{"status"},
	}
}

func workflowTimeoutMetadata(raw rawUpdateWorkflowRequest, item workflowResponse) map[string]any {
	metadata := map[string]any{
		"workflow_name":  item.Name,
		"changed_fields": []string{"timeout_minutes", "stall_timeout_minutes"},
	}
	if raw.TimeoutMinutes != nil {
		metadata["timeout_minutes"] = *raw.TimeoutMinutes
	}
	if raw.StallTimeoutMinutes != nil {
		metadata["stall_timeout_minutes"] = *raw.StallTimeoutMinutes
	}
	return metadata
}

func workflowChangedFields(raw rawUpdateWorkflowRequest) []string {
	fields := make([]string, 0, 3)
	if raw.Name != nil {
		fields = append(fields, "name")
	}
	if raw.Type != nil {
		fields = append(fields, "type")
	}
	if raw.HarnessPath != nil {
		fields = append(fields, "harness_path")
	}
	return fields
}

func ticketCommentMetadata(comment ticketCommentResponse) map[string]any {
	return map[string]any{
		"comment_id":     comment.ID,
		"created_by":     comment.CreatedBy,
		"edit_count":     comment.EditCount,
		"is_deleted":     comment.IsDeleted,
		"changed_fields": []string{"comment"},
	}
}

func uuidPointersEqual(left *uuid.UUID, right *uuid.UUID) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
}

func mapsEqual(left map[string]any, right map[string]any) bool {
	return reflect.DeepEqual(left, right)
}
