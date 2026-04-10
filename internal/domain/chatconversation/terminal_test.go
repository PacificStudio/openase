package chatconversation

import "testing"

func TestParseTerminalMode(t *testing.T) {
	t.Parallel()

	mode, err := ParseTerminalMode("  shell  ")
	if err != nil {
		t.Fatalf("ParseTerminalMode() error = %v", err)
	}
	if mode != TerminalModeShell {
		t.Fatalf("ParseTerminalMode() = %q, want %q", mode, TerminalModeShell)
	}
}

func TestParseTerminalModeRejectsUnsupportedValues(t *testing.T) {
	t.Parallel()

	if _, err := ParseTerminalMode("tmux"); err == nil {
		t.Fatal("ParseTerminalMode() error = nil, want error")
	}
}

func TestParseOpenTerminalSessionInput(t *testing.T) {
	t.Parallel()

	repoPath := "  backend  "
	cwdPath := "  src  "
	cols := 132
	rows := 44
	parsed, err := ParseOpenTerminalSessionInput(OpenTerminalSessionRawInput{
		Mode:     " shell ",
		RepoPath: &repoPath,
		CWDPath:  &cwdPath,
		Cols:     &cols,
		Rows:     &rows,
	})
	if err != nil {
		t.Fatalf("ParseOpenTerminalSessionInput() error = %v", err)
	}
	if parsed.Mode != TerminalModeShell || parsed.RepoPath == nil || *parsed.RepoPath != "backend" || parsed.CWDPath == nil || *parsed.CWDPath != "src" || parsed.Cols != 132 || parsed.Rows != 44 {
		t.Fatalf("ParseOpenTerminalSessionInput() = %+v", parsed)
	}
}

func TestParseOpenTerminalSessionInputDefaultsSizeAndClearsBlankPaths(t *testing.T) {
	t.Parallel()

	blank := "   "
	parsed, err := ParseOpenTerminalSessionInput(OpenTerminalSessionRawInput{
		Mode:     "shell",
		RepoPath: &blank,
		CWDPath:  &blank,
	})
	if err != nil {
		t.Fatalf("ParseOpenTerminalSessionInput() error = %v", err)
	}
	if parsed.RepoPath != nil || parsed.CWDPath != nil || parsed.Cols != DefaultTerminalCols || parsed.Rows != DefaultTerminalRows {
		t.Fatalf("ParseOpenTerminalSessionInput() = %+v", parsed)
	}
}

func TestParseOpenTerminalSessionInputDefaultsSizeWithNilPaths(t *testing.T) {
	t.Parallel()

	parsed, err := ParseOpenTerminalSessionInput(OpenTerminalSessionRawInput{
		Mode: "shell",
	})
	if err != nil {
		t.Fatalf("ParseOpenTerminalSessionInput() error = %v", err)
	}
	if parsed.RepoPath != nil || parsed.CWDPath != nil || parsed.Cols != DefaultTerminalCols || parsed.Rows != DefaultTerminalRows {
		t.Fatalf("ParseOpenTerminalSessionInput() = %+v", parsed)
	}
}

func TestParseOpenTerminalSessionInputRejectsInvalidSize(t *testing.T) {
	t.Parallel()

	zero := 0
	tooLarge := MaxTerminalSize + 1
	if _, err := ParseOpenTerminalSessionInput(OpenTerminalSessionRawInput{Mode: "shell", Cols: &zero}); err == nil {
		t.Fatal("expected cols validation error")
	}
	if _, err := ParseOpenTerminalSessionInput(OpenTerminalSessionRawInput{Mode: "shell", Rows: &zero}); err == nil {
		t.Fatal("expected rows validation error")
	}
	if _, err := ParseOpenTerminalSessionInput(OpenTerminalSessionRawInput{Mode: "shell", Cols: &tooLarge}); err == nil {
		t.Fatal("expected oversized cols validation error")
	}
	if _, err := ParseOpenTerminalSessionInput(OpenTerminalSessionRawInput{Mode: "shell", Rows: &tooLarge}); err == nil {
		t.Fatal("expected oversized rows validation error")
	}
}

func TestParseOpenTerminalSessionInputRejectsUnsupportedMode(t *testing.T) {
	t.Parallel()

	if _, err := ParseOpenTerminalSessionInput(OpenTerminalSessionRawInput{Mode: "tmux"}); err == nil {
		t.Fatal("expected mode validation error")
	}
}

func TestTrimOptionalTerminalPathTrimsNonEmptyValues(t *testing.T) {
	t.Parallel()

	raw := "  nested/path  "
	trimmed := trimOptionalTerminalPath(&raw)
	if trimmed == nil || *trimmed != "nested/path" {
		t.Fatalf("trimOptionalTerminalPath() = %v", trimmed)
	}
}
