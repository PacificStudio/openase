package projectrepomirror

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	rootent "github.com/BetterAndBetterII/openase/ent"
	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/uuid"
)

func TestResolveRepositorySyncAuthUsesUnifiedGitHubCredential(t *testing.T) {
	projectID := uuid.New()
	svc := NewService(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.ConfigureGitHubCredentials(projectRepoMirrorStubTokenResolver{
		projectID: projectID,
		resolved: githubauthdomain.ResolvedCredential{
			Scope: githubauthdomain.ScopeProject,
			Token: "ghu_platform_token",
		},
	})

	auth, err := svc.resolveRepositorySyncAuth(context.Background(), &rootent.ProjectRepo{
		ProjectID:     projectID,
		RepositoryURL: "https://github.com/acme/backend.git",
	})
	if err != nil {
		t.Fatalf("resolveRepositorySyncAuth() error = %v", err)
	}

	basicAuth, ok := auth.gitAuth.(*githttp.BasicAuth)
	if !ok {
		t.Fatalf("resolveRepositorySyncAuth() gitAuth type = %T, want *http.BasicAuth", auth.gitAuth)
	}
	if basicAuth.Username != "x-access-token" || basicAuth.Password != "ghu_platform_token" {
		t.Fatalf("resolveRepositorySyncAuth() gitAuth = %+v", basicAuth)
	}
	if auth.githubToken != "ghu_platform_token" {
		t.Fatalf("resolveRepositorySyncAuth() githubToken = %q", auth.githubToken)
	}
}

func TestBuildRemoteSyncMirrorScriptProjectsGitHubTokenIntoControlledSession(t *testing.T) {
	script := buildRemoteSyncMirrorScript(
		"/srv/openase/mirrors/backend",
		"https://github.com/acme/backend.git",
		"main",
		"ghu_platform_token",
	)

	if !strings.Contains(script, "export GH_TOKEN='ghu_platform_token'") {
		t.Fatalf("expected GH_TOKEN projection in remote script, got %q", script)
	}
	if !strings.Contains(script, "OPENASE_GITHUB_AUTH_HEADER=") {
		t.Fatalf("expected transient GitHub auth header in remote script, got %q", script)
	}
	if !strings.Contains(script, "git -c http.https://github.com/.extraheader=\"$OPENASE_GITHUB_AUTH_HEADER\" clone --branch 'main' --single-branch 'https://github.com/acme/backend.git' '/srv/openase/mirrors/backend'") {
		t.Fatalf("expected authenticated clone command, got %q", script)
	}
	if !strings.Contains(script, "git -c http.https://github.com/.extraheader=\"$OPENASE_GITHUB_AUTH_HEADER\" -C '/srv/openase/mirrors/backend' fetch origin") {
		t.Fatalf("expected authenticated fetch command, got %q", script)
	}
}

type projectRepoMirrorStubTokenResolver struct {
	projectID uuid.UUID
	resolved  githubauthdomain.ResolvedCredential
}

func (s projectRepoMirrorStubTokenResolver) ResolveProjectCredential(_ context.Context, projectID uuid.UUID) (githubauthdomain.ResolvedCredential, error) {
	if projectID != s.projectID {
		return githubauthdomain.ResolvedCredential{}, nil
	}
	return s.resolved, nil
}
