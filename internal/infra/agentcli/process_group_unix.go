//go:build !windows

package agentcli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/BetterAndBetterII/openase/internal/logging"
)

var _ = logging.DeclareComponent("agentcli-process-group-unix")

func configureProcessGroup(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func interruptProcess(process *os.Process) error {
	if process == nil {
		return fmt.Errorf("process not started")
	}
	if err := signalProcessGroup(process, syscall.SIGINT); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("interrupt process: %w", err)
	}
	return nil
}

func killProcess(process *os.Process) error {
	if process == nil {
		return fmt.Errorf("process not started")
	}
	if err := signalProcessGroup(process, syscall.SIGKILL); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("kill process: %w", err)
	}
	return nil
}

func signalProcessGroup(process *os.Process, signal syscall.Signal) error {
	if process == nil {
		return fmt.Errorf("process not started")
	}

	pgid, err := syscall.Getpgid(process.Pid)
	if err == nil && pgid > 0 {
		if err := syscall.Kill(-pgid, signal); err != nil && !errors.Is(err, syscall.ESRCH) {
			return err
		}
		return nil
	}

	if err := process.Signal(signal); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	return nil
}
