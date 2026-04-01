package executable

import (
	"os/exec"

	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

var _ = logging.DeclareComponent("executable-path")

type pathResolver struct{}

// NewPathResolver returns an executable resolver backed by the system PATH.
func NewPathResolver() provider.ExecutableResolver {
	return pathResolver{}
}

// LookPath resolves an executable name from PATH.
func (pathResolver) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}
