package executable

import (
	"os/exec"

	"github.com/BetterAndBetterII/openase/internal/provider"
)

type pathResolver struct{}

// NewPathResolver returns an executable resolver backed by the system PATH.
func NewPathResolver() provider.ExecutableResolver {
	return pathResolver{}
}

// LookPath resolves an executable name from PATH.
func (pathResolver) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}
