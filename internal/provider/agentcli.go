package provider

import (
	"context"
	"fmt"
	"io"
	"strings"
)

type AgentCLICommand string

func ParseAgentCLICommand(raw string) (AgentCLICommand, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("agent cli command must not be empty")
	}

	return AgentCLICommand(trimmed), nil
}

func MustParseAgentCLICommand(raw string) AgentCLICommand {
	command, err := ParseAgentCLICommand(raw)
	if err != nil {
		panic(err)
	}

	return command
}

func (c AgentCLICommand) String() string {
	return string(c)
}

type AgentCLIProcessSpec struct {
	Command          AgentCLICommand
	Args             []string
	WorkingDirectory *AbsolutePath
	Environment      []string
}

func NewAgentCLIProcessSpec(
	command AgentCLICommand,
	args []string,
	workingDirectory *AbsolutePath,
	environment []string,
) (AgentCLIProcessSpec, error) {
	if command == "" {
		return AgentCLIProcessSpec{}, fmt.Errorf("agent cli command must not be empty")
	}
	if workingDirectory != nil && *workingDirectory == "" {
		return AgentCLIProcessSpec{}, fmt.Errorf("working directory must not be empty when provided")
	}

	normalizedEnv := make([]string, 0, len(environment))
	for _, entry := range environment {
		if err := validateProcessEnvironmentEntry(entry); err != nil {
			return AgentCLIProcessSpec{}, err
		}
		normalizedEnv = append(normalizedEnv, entry)
	}

	return AgentCLIProcessSpec{
		Command:          command,
		Args:             append([]string(nil), args...),
		WorkingDirectory: workingDirectory,
		Environment:      normalizedEnv,
	}, nil
}

func validateProcessEnvironmentEntry(entry string) error {
	if entry == "" {
		return fmt.Errorf("environment entry must not be empty")
	}
	name, _, found := strings.Cut(entry, "=")
	if !found {
		return fmt.Errorf("environment entry %q must use KEY=VALUE format", entry)
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("environment entry %q must have a non-empty key", entry)
	}

	return nil
}

type AgentCLIProcess interface {
	PID() int
	Stdin() io.WriteCloser
	Stdout() io.ReadCloser
	Stderr() io.ReadCloser
	Wait() error
	Stop(context.Context) error
}

type AgentCLIProcessManager interface {
	Start(context.Context, AgentCLIProcessSpec) (AgentCLIProcess, error)
}
