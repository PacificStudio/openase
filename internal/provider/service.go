package provider

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

var serviceNamePattern = regexp.MustCompile(`^[a-z0-9]+([._-][a-z0-9]+)*$`)

type ServiceName string

func ParseServiceName(raw string) (ServiceName, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("service name must not be empty")
	}
	if !serviceNamePattern.MatchString(trimmed) {
		return "", fmt.Errorf("service name %q must match %s", trimmed, serviceNamePattern.String())
	}

	return ServiceName(trimmed), nil
}

func MustParseServiceName(raw string) ServiceName {
	name, err := ParseServiceName(raw)
	if err != nil {
		panic(err)
	}

	return name
}

func (n ServiceName) String() string {
	return string(n)
}

type AbsolutePath string

func ParseAbsolutePath(raw string) (AbsolutePath, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("absolute path must not be empty")
	}
	if !filepath.IsAbs(trimmed) {
		return "", fmt.Errorf("path %q must be absolute", trimmed)
	}

	return AbsolutePath(filepath.Clean(trimmed)), nil
}

func MustParseAbsolutePath(raw string) AbsolutePath {
	path, err := ParseAbsolutePath(raw)
	if err != nil {
		panic(err)
	}

	return path
}

func (p AbsolutePath) String() string {
	return string(p)
}

type UserServiceInstallSpec struct {
	Name             ServiceName
	Description      string
	ProgramPath      AbsolutePath
	Arguments        []string
	WorkingDirectory AbsolutePath
	EnvironmentFile  AbsolutePath
	StdoutPath       AbsolutePath
	StderrPath       AbsolutePath
}

func NewUserServiceInstallSpec(
	name ServiceName,
	description string,
	programPath AbsolutePath,
	arguments []string,
	workingDirectory AbsolutePath,
	environmentFile AbsolutePath,
	stdoutPath AbsolutePath,
	stderrPath AbsolutePath,
) (UserServiceInstallSpec, error) {
	trimmedDescription := strings.TrimSpace(description)
	if name == "" {
		return UserServiceInstallSpec{}, fmt.Errorf("service name must not be empty")
	}
	if trimmedDescription == "" {
		return UserServiceInstallSpec{}, fmt.Errorf("service description must not be empty")
	}
	if programPath == "" {
		return UserServiceInstallSpec{}, fmt.Errorf("program path must not be empty")
	}
	if workingDirectory == "" {
		return UserServiceInstallSpec{}, fmt.Errorf("working directory must not be empty")
	}
	if environmentFile == "" {
		return UserServiceInstallSpec{}, fmt.Errorf("environment file must not be empty")
	}
	if stdoutPath == "" {
		return UserServiceInstallSpec{}, fmt.Errorf("stdout path must not be empty")
	}
	if stderrPath == "" {
		return UserServiceInstallSpec{}, fmt.Errorf("stderr path must not be empty")
	}

	return UserServiceInstallSpec{
		Name:             name,
		Description:      trimmedDescription,
		ProgramPath:      programPath,
		Arguments:        append([]string(nil), arguments...),
		WorkingDirectory: workingDirectory,
		EnvironmentFile:  environmentFile,
		StdoutPath:       stdoutPath,
		StderrPath:       stderrPath,
	}, nil
}

type UserServiceLogsOptions struct {
	Follow bool
	Lines  int
	Stdout io.Writer
	Stderr io.Writer
}

func NewUserServiceLogsOptions(lines int, follow bool, stdout io.Writer, stderr io.Writer) (UserServiceLogsOptions, error) {
	if lines <= 0 {
		return UserServiceLogsOptions{}, fmt.Errorf("log lines must be positive")
	}
	if stdout == nil {
		return UserServiceLogsOptions{}, fmt.Errorf("stdout writer must not be nil")
	}
	if stderr == nil {
		return UserServiceLogsOptions{}, fmt.Errorf("stderr writer must not be nil")
	}

	return UserServiceLogsOptions{
		Follow: follow,
		Lines:  lines,
		Stdout: stdout,
		Stderr: stderr,
	}, nil
}

type UserServiceManager interface {
	Platform() string
	Apply(context.Context, UserServiceInstallSpec) error
	Down(context.Context, ServiceName) error
	Restart(context.Context, ServiceName) error
	Logs(context.Context, ServiceName, UserServiceLogsOptions) error
}
