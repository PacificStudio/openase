package executable

import (
	"os/exec"

	"github.com/BetterAndBetterII/openase/internal/provider"
)

type pathResolver struct{}

func NewPathResolver() provider.ExecutableResolver {
	return pathResolver{}
}

func (pathResolver) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}
