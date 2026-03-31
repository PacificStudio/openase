package catalog

import (
	"context"
	"fmt"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

func (s *service) ListAgentProviders(ctx context.Context, organizationID uuid.UUID) ([]domain.AgentProvider, error) {
	items, err := s.repo.ListAgentProviders(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return annotateAgentProvidersAvailability(items), nil
}

func (s *service) CreateAgentProvider(ctx context.Context, input domain.CreateAgentProvider) (domain.AgentProvider, error) {
	resolved, err := s.resolveAgentProviderCLICommand(input.AdapterType, input.CliCommand)
	if err != nil {
		return domain.AgentProvider{}, err
	}
	input.CliCommand = resolved
	input.CliArgs = defaultAgentProviderCLIArgs(input.AdapterType, input.CliArgs)

	item, err := s.repo.CreateAgentProvider(ctx, input)
	if err != nil {
		return domain.AgentProvider{}, err
	}

	return annotateAgentProviderAvailability(item), nil
}

func (s *service) GetAgentProvider(ctx context.Context, id uuid.UUID) (domain.AgentProvider, error) {
	item, err := s.repo.GetAgentProvider(ctx, id)
	if err != nil {
		return domain.AgentProvider{}, err
	}

	return annotateAgentProviderAvailability(item), nil
}

func (s *service) UpdateAgentProvider(ctx context.Context, input domain.UpdateAgentProvider) (domain.AgentProvider, error) {
	resolved, err := s.resolveAgentProviderCLICommand(input.AdapterType, input.CliCommand)
	if err != nil {
		return domain.AgentProvider{}, err
	}
	input.CliCommand = resolved
	input.CliArgs = defaultAgentProviderCLIArgs(input.AdapterType, input.CliArgs)

	item, err := s.repo.UpdateAgentProvider(ctx, input)
	if err != nil {
		return domain.AgentProvider{}, err
	}

	return annotateAgentProviderAvailability(item), nil
}

func (s *service) ListAgents(ctx context.Context, projectID uuid.UUID) ([]domain.Agent, error) {
	return s.repo.ListAgents(ctx, projectID)
}

func (s *service) ListAgentRuns(ctx context.Context, projectID uuid.UUID) ([]domain.AgentRun, error) {
	return s.repo.ListAgentRuns(ctx, projectID)
}

func (s *service) CreateAgent(ctx context.Context, input domain.CreateAgent) (domain.Agent, error) {
	return s.repo.CreateAgent(ctx, input)
}

func (s *service) GetAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error) {
	return s.repo.GetAgent(ctx, id)
}

func (s *service) GetAgentRun(ctx context.Context, id uuid.UUID) (domain.AgentRun, error) {
	return s.repo.GetAgentRun(ctx, id)
}

func (s *service) RequestAgentPause(ctx context.Context, id uuid.UUID) (domain.Agent, error) {
	current, err := s.repo.GetAgent(ctx, id)
	if err != nil {
		return domain.Agent{}, err
	}

	nextState, err := domain.ResolvePauseRuntimeControlState(current)
	if err != nil {
		return domain.Agent{}, fmt.Errorf("%w: %v", ErrConflict, err)
	}

	return s.repo.UpdateAgentRuntimeControlState(ctx, domain.UpdateAgentRuntimeControlState{
		ID:                  id,
		RuntimeControlState: nextState,
	})
}

func (s *service) RequestAgentResume(ctx context.Context, id uuid.UUID) (domain.Agent, error) {
	current, err := s.repo.GetAgent(ctx, id)
	if err != nil {
		return domain.Agent{}, err
	}

	nextState, err := domain.ResolveResumeRuntimeControlState(current)
	if err != nil {
		return domain.Agent{}, fmt.Errorf("%w: %v", ErrConflict, err)
	}

	return s.repo.UpdateAgentRuntimeControlState(ctx, domain.UpdateAgentRuntimeControlState{
		ID:                  id,
		RuntimeControlState: nextState,
	})
}

func (s *service) DeleteAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error) {
	return s.repo.DeleteAgent(ctx, id)
}

func (s *service) resolveAgentProviderCLICommand(adapterType domain.AgentProviderAdapterType, cliCommand string) (string, error) {
	if cliCommand != "" {
		return cliCommand, nil
	}

	commandName, ok := defaultAgentProviderCommand(adapterType)
	if !ok {
		return "", fmt.Errorf("%w: cli_command must not be empty for adapter_type %s", ErrInvalidInput, adapterType)
	}
	if s.resolver == nil {
		return "", fmt.Errorf("%w: cli_command auto-detection is unavailable", ErrInvalidInput)
	}

	resolved, err := s.resolver.LookPath(commandName)
	if err != nil {
		return "", fmt.Errorf("%w: cli_command not provided and executable %q was not found in PATH", ErrInvalidInput, commandName)
	}

	return resolved, nil
}

func defaultAgentProviderCommand(adapterType domain.AgentProviderAdapterType) (string, bool) {
	switch adapterType {
	case domain.AgentProviderAdapterTypeClaudeCodeCLI:
		return "claude", true
	case domain.AgentProviderAdapterTypeCodexAppServer:
		return "codex", true
	case domain.AgentProviderAdapterTypeGeminiCLI:
		return "gemini", true
	default:
		return "", false
	}
}

func defaultAgentProviderCLIArgs(adapterType domain.AgentProviderAdapterType, cliArgs []string) []string {
	if len(cliArgs) > 0 {
		return append([]string(nil), cliArgs...)
	}

	switch adapterType {
	case domain.AgentProviderAdapterTypeCodexAppServer:
		return []string{"app-server", "--listen", "stdio://"}
	default:
		return nil
	}
}

func annotateAgentProvidersAvailability(
	items []domain.AgentProvider,
) []domain.AgentProvider {
	annotated := make([]domain.AgentProvider, 0, len(items))
	for _, item := range items {
		annotated = append(annotated, annotateAgentProviderAvailability(item))
	}

	return annotated
}

func annotateAgentProviderAvailability(item domain.AgentProvider) domain.AgentProvider {
	item = domain.DeriveAgentProviderAvailability(item, time.Now().UTC())
	return domain.DeriveAgentProviderCapabilities(item)
}

func preferredAvailableProviderID(items []domain.AgentProvider) *uuid.UUID {
	preferred := []struct {
		name        string
		adapterType domain.AgentProviderAdapterType
	}{
		{name: "OpenAI Codex", adapterType: domain.AgentProviderAdapterTypeCodexAppServer},
		{name: "Claude Code", adapterType: domain.AgentProviderAdapterTypeClaudeCodeCLI},
		{name: "Gemini CLI", adapterType: domain.AgentProviderAdapterTypeGeminiCLI},
	}
	for _, candidate := range preferred {
		for _, item := range items {
			if item.Available && item.Name == candidate.name && item.AdapterType == candidate.adapterType {
				id := item.ID
				return &id
			}
		}
	}

	for _, item := range items {
		if item.Available {
			id := item.ID
			return &id
		}
	}

	return nil
}
