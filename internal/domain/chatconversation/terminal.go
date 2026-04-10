package chatconversation

import (
	"fmt"
	"strings"
)

const (
	DefaultTerminalCols = 120
	DefaultTerminalRows = 30
	MaxTerminalSize     = 65535
)

type TerminalMode string

const (
	TerminalModeShell TerminalMode = "shell"
)

type OpenTerminalSessionRawInput struct {
	Mode     string
	RepoPath *string
	CWDPath  *string
	Cols     *int
	Rows     *int
}

type OpenTerminalSessionInput struct {
	Mode     TerminalMode
	RepoPath *string
	CWDPath  *string
	Cols     int
	Rows     int
}

func ParseTerminalMode(raw string) (TerminalMode, error) {
	switch TerminalMode(strings.TrimSpace(raw)) {
	case TerminalModeShell:
		return TerminalModeShell, nil
	default:
		return "", fmt.Errorf("mode must be %q", TerminalModeShell)
	}
}

func ParseOpenTerminalSessionInput(raw OpenTerminalSessionRawInput) (OpenTerminalSessionInput, error) {
	mode, err := ParseTerminalMode(raw.Mode)
	if err != nil {
		return OpenTerminalSessionInput{}, err
	}
	cols, err := parseTerminalSize(raw.Cols, "cols", DefaultTerminalCols)
	if err != nil {
		return OpenTerminalSessionInput{}, err
	}
	rows, err := parseTerminalSize(raw.Rows, "rows", DefaultTerminalRows)
	if err != nil {
		return OpenTerminalSessionInput{}, err
	}
	return OpenTerminalSessionInput{
		Mode:     mode,
		RepoPath: trimOptionalTerminalPath(raw.RepoPath),
		CWDPath:  trimOptionalTerminalPath(raw.CWDPath),
		Cols:     cols,
		Rows:     rows,
	}, nil
}

func parseTerminalSize(raw *int, name string, fallback int) (int, error) {
	if raw == nil {
		return fallback, nil
	}
	if *raw <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}
	if *raw > MaxTerminalSize {
		return 0, fmt.Errorf("%s must be less than or equal to %d", name, MaxTerminalSize)
	}
	return *raw, nil
}

func trimOptionalTerminalPath(raw *string) *string {
	if raw == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
