package chatconversation

import (
	"errors"
	"testing"
)

func TestParseWorkspaceTreeInput(t *testing.T) {
	t.Parallel()

	input, err := ParseWorkspaceTreeInput(RawWorkspaceTreeInput{
		Repo: "backend",
		Path: "src/lib",
	})
	if err != nil {
		t.Fatalf("ParseWorkspaceTreeInput() error = %v", err)
	}
	if input.RepoName != "backend" || input.Path.String() != "src/lib" {
		t.Fatalf("unexpected parsed tree input: %+v", input)
	}

	root, err := ParseWorkspaceTreeInput(RawWorkspaceTreeInput{
		Repo: "backend",
		Path: "  ",
	})
	if err != nil {
		t.Fatalf("ParseWorkspaceTreeInput(root) error = %v", err)
	}
	if !root.Path.IsRoot() {
		t.Fatalf("expected root path, got %q", root.Path)
	}
	if got := root.RepoName.String(); got != "backend" {
		t.Fatalf("RepoName.String() = %q, want backend", got)
	}
}

func TestParseWorkspaceFileInputRejectsUnsafePaths(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		raw  RawWorkspaceFileInput
	}{
		{
			name: "empty repo",
			raw: RawWorkspaceFileInput{
				Repo: " ",
				Path: "README.md",
			},
		},
		{
			name: "empty path",
			raw: RawWorkspaceFileInput{
				Repo: "backend",
				Path: " ",
			},
		},
		{
			name: "absolute path",
			raw: RawWorkspaceFileInput{
				Repo: "backend",
				Path: "/etc/passwd",
			},
		},
		{
			name: "dot dot",
			raw: RawWorkspaceFileInput{
				Repo: "backend",
				Path: "../secret",
			},
		},
		{
			name: "escaped path",
			raw: RawWorkspaceFileInput{
				Repo: "backend",
				Path: "docs/../../secret",
			},
		},
		{
			name: "git path",
			raw: RawWorkspaceFileInput{
				Repo: "backend",
				Path: ".git/config",
			},
		},
		{
			name: "git segment",
			raw: RawWorkspaceFileInput{
				Repo: "backend",
				Path: "src/.git/config",
			},
		},
		{
			name: "backslash path",
			raw: RawWorkspaceFileInput{
				Repo: "backend",
				Path: "src\\main.go",
			},
		},
		{
			name: "repo path",
			raw: RawWorkspaceFileInput{
				Repo: "services/backend",
				Path: "README.md",
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseWorkspaceFileInput(tc.raw)
			if err == nil {
				t.Fatal("ParseWorkspaceFileInput() error = nil, want error")
			}
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("ParseWorkspaceFileInput() error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestWorkspaceRelativePathSegments(t *testing.T) {
	t.Parallel()

	input, err := ParseWorkspaceFileInput(RawWorkspaceFileInput{
		Repo: "backend",
		Path: "src/api/server.go",
	})
	if err != nil {
		t.Fatalf("ParseWorkspaceFileInput() error = %v", err)
	}
	segments := input.Path.Segments()
	if len(segments) != 3 || segments[0] != "src" || segments[1] != "api" || segments[2] != "server.go" {
		t.Fatalf("unexpected path segments: %#v", segments)
	}

	root := WorkspaceRelativePath("")
	if segments := root.Segments(); segments != nil {
		t.Fatalf("root segments = %#v, want nil", segments)
	}
}

func TestParseWorkspaceTreeInputRejectsInvalidPath(t *testing.T) {
	t.Parallel()

	_, err := ParseWorkspaceTreeInput(RawWorkspaceTreeInput{
		Repo: "backend",
		Path: "/abs/path",
	})
	if err == nil || !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("ParseWorkspaceTreeInput() error = %v, want ErrInvalidInput", err)
	}

	_, err = ParseWorkspaceTreeInput(RawWorkspaceTreeInput{
		Repo: "services/backend",
		Path: "",
	})
	if err == nil || !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("ParseWorkspaceTreeInput(repo) error = %v, want ErrInvalidInput", err)
	}
}

func TestParseWorkspaceRelativePathNormalizesDotRoot(t *testing.T) {
	t.Parallel()

	input, err := ParseWorkspaceTreeInput(RawWorkspaceTreeInput{
		Repo: "backend",
		Path: ".",
	})
	if err != nil {
		t.Fatalf("ParseWorkspaceTreeInput(.) error = %v", err)
	}
	if !input.Path.IsRoot() {
		t.Fatalf("expected dot path to normalize to root, got %q", input.Path)
	}

	_, err = ParseWorkspaceFileInput(RawWorkspaceFileInput{
		Repo: "backend",
		Path: ".",
	})
	if err == nil || !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("ParseWorkspaceFileInput(.) error = %v, want ErrInvalidInput", err)
	}
}
