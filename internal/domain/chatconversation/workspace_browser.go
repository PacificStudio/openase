package chatconversation

import (
	"fmt"
	"path"
	"strings"
)

type WorkspaceRepoName string

func (n WorkspaceRepoName) String() string {
	return string(n)
}

type WorkspaceRelativePath string

func (p WorkspaceRelativePath) String() string {
	return string(p)
}

func (p WorkspaceRelativePath) IsRoot() bool {
	return strings.TrimSpace(string(p)) == ""
}

func (p WorkspaceRelativePath) Segments() []string {
	if p.IsRoot() {
		return nil
	}
	return strings.Split(string(p), "/")
}

type RawWorkspaceTreeInput struct {
	Repo string
	Path string
}

type WorkspaceTreeInput struct {
	RepoName WorkspaceRepoName
	Path     WorkspaceRelativePath
}

func ParseWorkspaceTreeInput(raw RawWorkspaceTreeInput) (WorkspaceTreeInput, error) {
	repoName, err := parseWorkspaceRepoName("repo", raw.Repo)
	if err != nil {
		return WorkspaceTreeInput{}, err
	}
	repoPath, err := parseWorkspaceRelativePath("path", raw.Path, true)
	if err != nil {
		return WorkspaceTreeInput{}, err
	}
	return WorkspaceTreeInput{
		RepoName: repoName,
		Path:     repoPath,
	}, nil
}

type RawWorkspaceFileInput struct {
	Repo string
	Path string
}

type WorkspaceFileInput struct {
	RepoName WorkspaceRepoName
	Path     WorkspaceRelativePath
}

func ParseWorkspaceFileInput(raw RawWorkspaceFileInput) (WorkspaceFileInput, error) {
	repoName, err := parseWorkspaceRepoName("repo", raw.Repo)
	if err != nil {
		return WorkspaceFileInput{}, err
	}
	repoPath, err := parseWorkspaceRelativePath("path", raw.Path, false)
	if err != nil {
		return WorkspaceFileInput{}, err
	}
	return WorkspaceFileInput{
		RepoName: repoName,
		Path:     repoPath,
	}, nil
}

func parseWorkspaceRepoName(field string, raw string) (WorkspaceRepoName, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%w: %s must not be empty", ErrInvalidInput, field)
	}
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return "", fmt.Errorf("%w: %s must be a repo name", ErrInvalidInput, field)
	}
	return WorkspaceRepoName(trimmed), nil
}

func parseWorkspaceRelativePath(
	field string,
	raw string,
	allowEmpty bool,
) (WorkspaceRelativePath, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		if allowEmpty {
			return "", nil
		}
		return "", fmt.Errorf("%w: %s must not be empty", ErrInvalidInput, field)
	}
	if strings.Contains(trimmed, "\\") {
		return "", fmt.Errorf("%w: %s must use forward slashes", ErrInvalidInput, field)
	}
	if strings.HasPrefix(trimmed, "/") {
		return "", fmt.Errorf("%w: %s must be repo-relative", ErrInvalidInput, field)
	}

	cleaned := path.Clean(trimmed)
	switch {
	case cleaned == ".":
		if allowEmpty {
			return "", nil
		}
		return "", fmt.Errorf("%w: %s must not be empty", ErrInvalidInput, field)
	case cleaned == "..", strings.HasPrefix(cleaned, "../"):
		return "", fmt.Errorf("%w: %s must stay inside the repo", ErrInvalidInput, field)
	}

	for _, segment := range strings.Split(cleaned, "/") {
		if segment == ".git" {
			return "", fmt.Errorf("%w: %s must not access .git", ErrInvalidInput, field)
		}
	}

	return WorkspaceRelativePath(cleaned), nil
}
