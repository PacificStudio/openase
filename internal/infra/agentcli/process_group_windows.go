//go:build windows

package agentcli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/BetterAndBetterII/openase/internal/logging"
)

var agentCLIProcessGroupWindowsComponent = logging.DeclareComponent("agentcli-process-group-windows")

func configureProcessGroup(cmd *exec.Cmd) {
	_ = cmd
}

func interruptProcess(process *os.Process) error {
	if process == nil {
		return fmt.Errorf("process not started")
	}
	if err := process.Signal(os.Interrupt); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("interrupt process: %w", err)
	}
	return nil
}

func killProcess(process *os.Process) error {
	if process == nil {
		return fmt.Errorf("process not started")
	}
	if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("kill process: %w", err)
	}
	return nil
}
