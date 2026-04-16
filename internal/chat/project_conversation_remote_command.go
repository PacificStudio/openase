package chat

import (
	"context"
	"fmt"
	"strings"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
)

func (s *ProjectConversationService) openProjectConversationCommandSession(
	ctx context.Context,
	machine catalogdomain.Machine,
	purpose string,
) (machinetransport.CommandSession, error) {
	if s == nil || s.runtimeManager == nil {
		return nil, fmt.Errorf("project conversation runtime manager unavailable for %s", purpose)
	}
	resolved, err := s.runtimeManager.resolveRuntimeTransport(machine)
	if err != nil {
		return nil, err
	}
	executor := resolved.CommandSessionExecutor()
	if executor == nil {
		return nil, fmt.Errorf(
			"%w: command session unavailable for %s on machine %s",
			machinetransport.ErrTransportUnavailable,
			purpose,
			machine.Name,
		)
	}
	session, err := executor.OpenCommandSession(ctx, machine)
	if err != nil {
		return nil, fmt.Errorf("open remote command session for %s: %w", purpose, err)
	}
	return session, nil
}

func (s *ProjectConversationService) runProjectConversationRemoteCommand(
	ctx context.Context,
	machine catalogdomain.Machine,
	command string,
	allowExitCodeOne bool,
	purpose string,
) ([]byte, error) {
	session, err := s.openProjectConversationCommandSession(ctx, machine, purpose)
	if err != nil {
		return nil, err
	}
	defer func() { _ = session.Close() }()

	output, err := session.CombinedOutput(command)
	if err != nil && (!allowExitCodeOne || !projectConversationCommandExitedWithCode(err, 1)) {
		return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}
