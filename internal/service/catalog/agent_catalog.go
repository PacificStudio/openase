package catalog

import (
	"context"
	"fmt"
	"time"

	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

func (s *service) ListAgentProviders(ctx context.Context, organizationID uuid.UUID) ([]domain.AgentProvider, error) {
	return s.repo.ListAgentProviders(ctx, organizationID)
}

func (s *service) CreateAgentProvider(ctx context.Context, input domain.CreateAgentProvider) (domain.AgentProvider, error) {
	resolved, err := s.resolveAgentProviderCLICommand(input.AdapterType, input.CliCommand)
	if err != nil {
		return domain.AgentProvider{}, err
	}
	input.CliCommand = resolved
	input.CliArgs = defaultAgentProviderCLIArgs(input.AdapterType, input.CliArgs)

	return s.repo.CreateAgentProvider(ctx, input)
}

func (s *service) GetAgentProvider(ctx context.Context, id uuid.UUID) (domain.AgentProvider, error) {
	return s.repo.GetAgentProvider(ctx, id)
}

func (s *service) UpdateAgentProvider(ctx context.Context, input domain.UpdateAgentProvider) (domain.AgentProvider, error) {
	resolved, err := s.resolveAgentProviderCLICommand(input.AdapterType, input.CliCommand)
	if err != nil {
		return domain.AgentProvider{}, err
	}
	input.CliCommand = resolved
	input.CliArgs = defaultAgentProviderCLIArgs(input.AdapterType, input.CliArgs)

	return s.repo.UpdateAgentProvider(ctx, input)
}

func (s *service) ListAgents(ctx context.Context, projectID uuid.UUID) ([]domain.Agent, error) {
	return s.repo.ListAgents(ctx, projectID)
}

func (s *service) CreateAgent(ctx context.Context, input domain.CreateAgent) (domain.Agent, error) {
	return s.repo.CreateAgent(ctx, input)
}

func (s *service) GetAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error) {
	return s.repo.GetAgent(ctx, id)
}

func (s *service) PauseAgentRuntime(ctx context.Context, id uuid.UUID) (domain.AgentRuntimeControlResult, error) {
	current, err := s.repo.GetAgent(ctx, id)
	if err != nil {
		return domain.AgentRuntimeControlResult{}, err
	}

	nextState, err := domain.BuildPauseAgentRuntime(current)
	if err != nil {
		return domain.AgentRuntimeControlResult{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	updated, err := s.repo.UpdateAgentRuntimeState(ctx, nextState)
	if err != nil {
		return domain.AgentRuntimeControlResult{}, err
	}

	return domain.AgentRuntimeControlResult{
		Agent:       updated,
		Transition:  domain.AgentRuntimeControlPause,
		RequestedAt: time.Now().UTC(),
	}, nil
}

func (s *service) ResumeAgentRuntime(ctx context.Context, id uuid.UUID) (domain.AgentRuntimeControlResult, error) {
	current, err := s.repo.GetAgent(ctx, id)
	if err != nil {
		return domain.AgentRuntimeControlResult{}, err
	}

	nextState, err := domain.BuildResumeAgentRuntime(current)
	if err != nil {
		return domain.AgentRuntimeControlResult{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	updated, err := s.repo.UpdateAgentRuntimeState(ctx, nextState)
	if err != nil {
		return domain.AgentRuntimeControlResult{}, err
	}

	return domain.AgentRuntimeControlResult{
		Agent:       updated,
		Transition:  domain.AgentRuntimeControlResume,
		RequestedAt: time.Now().UTC(),
	}, nil
}

func (s *service) DeleteAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error) {
	return s.repo.DeleteAgent(ctx, id)
}

func (s *service) resolveAgentProviderCLICommand(adapterType entagentprovider.AdapterType, cliCommand string) (string, error) {
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

func defaultAgentProviderCommand(adapterType entagentprovider.AdapterType) (string, bool) {
	switch adapterType {
	case entagentprovider.AdapterTypeClaudeCodeCli:
		return "claude", true
	case entagentprovider.AdapterTypeCodexAppServer:
		return "codex", true
	case entagentprovider.AdapterTypeGeminiCli:
		return "gemini", true
	default:
		return "", false
	}
}

func defaultAgentProviderCLIArgs(adapterType entagentprovider.AdapterType, cliArgs []string) []string {
	if len(cliArgs) > 0 {
		return append([]string(nil), cliArgs...)
	}

	switch adapterType {
	case entagentprovider.AdapterTypeCodexAppServer:
		return []string{"app-server", "--listen", "stdio://"}
	default:
		return nil
	}
}
