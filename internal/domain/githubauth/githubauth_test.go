package githubauth

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEnumValidity(t *testing.T) {
	t.Parallel()

	if !ScopeOrganization.IsValid() || !ScopeProject.IsValid() || Scope("invalid").IsValid() {
		t.Fatalf("unexpected scope validity")
	}
	if !SourceDeviceFlow.IsValid() || !SourceGHCLIImport.IsValid() || !SourceManualPaste.IsValid() || Source("invalid").IsValid() {
		t.Fatalf("unexpected source validity")
	}
	if !ProbeStateMissing.IsValid() || !ProbeStateConfigured.IsValid() || !ProbeStateProbing.IsValid() ||
		!ProbeStateValid.IsValid() || !ProbeStateInsufficientPermissions.IsValid() || !ProbeStateRevoked.IsValid() ||
		!ProbeStateError.IsValid() || ProbeState("invalid").IsValid() {
		t.Fatalf("unexpected probe state validity")
	}
	if !RepoAccessNotChecked.IsValid() || !RepoAccessGranted.IsValid() || !RepoAccessDenied.IsValid() || RepoAccess("invalid").IsValid() {
		t.Fatalf("unexpected repo access validity")
	}
}

func TestRepositoryRefString(t *testing.T) {
	t.Parallel()

	if got := (RepositoryRef{Owner: "octo", Name: "repo"}).String(); got != "octo/repo" {
		t.Fatalf("unexpected repository ref: %s", got)
	}
}

func TestMissingAndConfiguredProbe(t *testing.T) {
	t.Parallel()

	missing := MissingProbe()
	if missing.State != ProbeStateMissing || missing.Configured || missing.Valid || missing.RepoAccess != RepoAccessNotChecked {
		t.Fatalf("unexpected missing probe: %#v", missing)
	}

	configured := ConfiguredProbe()
	if configured.State != ProbeStateConfigured || !configured.Configured || configured.Valid || configured.RepoAccess != RepoAccessNotChecked {
		t.Fatalf("unexpected configured probe: %#v", configured)
	}
}

func TestNormalizeProbe(t *testing.T) {
	t.Parallel()

	assertProbe(t, NormalizeProbe(nil, false), MissingProbe())
	assertProbe(t, NormalizeProbe(nil, true), ConfiguredProbe())

	checkedAt := time.Unix(1710000000, 0).UTC()
	raw := &TokenProbe{
		State:       ProbeState("bad"),
		Configured:  false,
		Valid:       true,
		Login:       "  octocat  ",
		Permissions: []string{"issues:write", "contents:read", "issues:write"},
		RepoAccess:  RepoAccess("bad"),
		CheckedAt:   &checkedAt,
		LastError:   "  revoked  ",
	}
	got := NormalizeProbe(raw, true)

	if got.State != ProbeStateConfigured {
		t.Fatalf("unexpected normalized state: %s", got.State)
	}
	if !got.Configured || !got.Valid {
		t.Fatalf("unexpected normalized config flags: %#v", got)
	}
	if got.Login != "octocat" {
		t.Fatalf("unexpected login: %q", got.Login)
	}
	if got.RepoAccess != RepoAccessNotChecked {
		t.Fatalf("unexpected repo access: %s", got.RepoAccess)
	}
	if got.CheckedAt == nil || !got.CheckedAt.Equal(checkedAt) {
		t.Fatalf("unexpected checked_at: %#v", got.CheckedAt)
	}
	if got.LastError != "revoked" {
		t.Fatalf("unexpected last error: %q", got.LastError)
	}
	if len(got.Permissions) != 2 || got.Permissions[0] != "contents:read" || got.Permissions[1] != "issues:write" {
		t.Fatalf("unexpected permissions: %#v", got.Permissions)
	}

	raw.Permissions[0] = "changed"
	if got.Permissions[0] != "contents:read" {
		t.Fatalf("permissions were not copied")
	}
	if got.CheckedAt == raw.CheckedAt {
		t.Fatalf("checked_at pointer should be cloned")
	}
}

func TestParseGitHubRepositoryURL(t *testing.T) {
	t.Parallel()

	validCases := []struct {
		raw  string
		want RepositoryRef
	}{
		{raw: "https://github.com/PacificStudio/openase.git", want: RepositoryRef{Owner: "pacificstudio", Name: "openase"}},
		{raw: " https://github.com/PacificStudio/OpenASE ", want: RepositoryRef{Owner: "pacificstudio", Name: "openase"}},
		{raw: "git@github.com:PacificStudio/openase.git", want: RepositoryRef{Owner: "pacificstudio", Name: "openase"}},
		{raw: "ssh://git@github.com/PacificStudio/openase.git", want: RepositoryRef{Owner: "pacificstudio", Name: "openase"}},
	}
	for _, tc := range validCases {
		got, ok := ParseGitHubRepositoryURL(tc.raw)
		if !ok || got != tc.want {
			t.Fatalf("expected %q to parse as %#v, got %#v ok=%v", tc.raw, tc.want, got, ok)
		}
	}

	for _, raw := range []string{
		"",
		"https://gitlab.com/PacificStudio/openase.git",
		"http://github.com/PacificStudio/openase.git",
		"https://github.com/PacificStudio",
		"https://github.com/PacificStudio/.git",
		"https://github.com/PacificStudio/openase/extra",
		"https://github.com//openase.git",
		"://bad",
	} {
		if got, ok := ParseGitHubRepositoryURL(raw); ok {
			t.Fatalf("expected %q to be rejected, got %#v", raw, got)
		}
	}
}

func TestNormalizeGitHubRepositoryURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw  string
		want string
		ok   bool
	}{
		{raw: "https://github.com/PacificStudio/openase.git", want: "https://github.com/pacificstudio/openase.git", ok: true},
		{raw: "git@github.com:PacificStudio/openase.git", want: "https://github.com/pacificstudio/openase.git", ok: true},
		{raw: "ssh://git@github.com/PacificStudio/openase.git", want: "https://github.com/pacificstudio/openase.git", ok: true},
		{raw: "https://gitlab.com/PacificStudio/openase.git", ok: false},
	}

	for _, tc := range tests {
		got, ok := NormalizeGitHubRepositoryURL(tc.raw)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("NormalizeGitHubRepositoryURL(%q) = %q, %v; want %q, %v", tc.raw, got, ok, tc.want, tc.ok)
		}
	}
}

func TestRedactToken(t *testing.T) {
	t.Parallel()

	if got := RedactToken(""); got != "" {
		t.Fatalf("expected empty redaction, got %q", got)
	}
	if got := RedactToken(" abcd "); got != "****" {
		t.Fatalf("unexpected short token redaction: %q", got)
	}
	if got := RedactToken("ghp_example_token"); got != "ghp_exa...oken" {
		t.Fatalf("unexpected token redaction: %q", got)
	}
}

func TestDefaultCipherSeed(t *testing.T) {
	t.Parallel()

	first := DefaultCipherSeed(" postgres://user:pass@host/db ")
	second := DefaultCipherSeed("postgres://user:pass@host/db")
	third := DefaultCipherSeed("postgres://user:pass@host/other")

	if first == "" || first != second {
		t.Fatalf("expected deterministic seed, got %q and %q", first, second)
	}
	if first == third {
		t.Fatalf("expected distinct seeds for distinct DSNs")
	}
}

func TestDeriveCipherSeed(t *testing.T) {
	t.Parallel()

	first := DeriveCipherSeed(" shared-seed ")
	second := DeriveCipherSeed("shared-seed")
	third := DeriveCipherSeed("other-seed")

	if first == "" || first != second {
		t.Fatalf("expected deterministic derived seed, got %q and %q", first, second)
	}
	if first == third {
		t.Fatalf("expected distinct derived seeds for distinct inputs")
	}
}

func TestResolveProjectCredential(t *testing.T) {
	t.Parallel()

	orgCredential := &StoredCredential{
		TokenPreview: "test-org-preview",
		Source:       SourceDeviceFlow,
	}
	projectCredential := &StoredCredential{
		TokenPreview: "test-project-preview",
		Source:       SourceManualPaste,
	}
	projectProbe := &TokenProbe{
		State:       ProbeStateValid,
		Configured:  true,
		Valid:       true,
		RepoAccess:  RepoAccessGranted,
		Permissions: []string{"contents:read"},
	}

	decrypt := func(credential StoredCredential) (string, error) {
		switch credential.TokenPreview {
		case "test-org-preview":
			return "org-token", nil
		case "test-project-preview":
			return "project-token", nil
		default:
			return "", errors.New("unknown credential")
		}
	}

	missing, err := ResolveProjectCredential(ProjectContext{}, decrypt)
	if err != nil {
		t.Fatalf("unexpected missing credential error: %v", err)
	}
	if missing.Token != "" {
		t.Fatalf("unexpected missing resolution: %#v", missing)
	}
	assertProbe(t, missing.Probe, MissingProbe())

	context := ProjectContext{
		ProjectID:              uuid.New(),
		OrganizationID:         uuid.New(),
		OrganizationCredential: orgCredential,
		OrganizationProbe:      &TokenProbe{State: ProbeStateProbing, RepoAccess: RepoAccessDenied},
		ProjectCredential:      projectCredential,
		ProjectProbe:           projectProbe,
	}
	got, err := ResolveProjectCredential(context, decrypt)
	if err != nil {
		t.Fatalf("unexpected project resolution error: %v", err)
	}
	if got.Scope != ScopeProject || got.Source != SourceManualPaste || got.Token != "project-token" || got.TokenPreview != "test-project-preview" {
		t.Fatalf("unexpected project resolution: %#v", got)
	}
	if got.Probe.State != ProbeStateValid || got.Probe.RepoAccess != RepoAccessGranted {
		t.Fatalf("unexpected project probe: %#v", got.Probe)
	}

	context.ProjectCredential = nil
	context.ProjectProbe = nil
	got, err = ResolveProjectCredential(context, decrypt)
	if err != nil {
		t.Fatalf("unexpected organization resolution error: %v", err)
	}
	if got.Scope != ScopeOrganization || got.Source != SourceDeviceFlow || got.Token != "org-token" || got.TokenPreview != "test-org-preview" {
		t.Fatalf("unexpected organization resolution: %#v", got)
	}
	if got.Probe.State != ProbeStateProbing || got.Probe.RepoAccess != RepoAccessDenied {
		t.Fatalf("unexpected organization probe: %#v", got.Probe)
	}
}

func TestResolveProjectCredentialDecryptError(t *testing.T) {
	t.Parallel()

	_, err := ResolveProjectCredential(ProjectContext{
		OrganizationCredential: &StoredCredential{TokenPreview: "test-org-preview"},
	}, func(StoredCredential) (string, error) {
		return "", errors.New("boom")
	})
	if err == nil || err.Error() != "decrypt organization GitHub credential: boom" {
		t.Fatalf("unexpected decrypt error: %v", err)
	}

	_, err = ResolveProjectCredential(ProjectContext{
		ProjectCredential: &StoredCredential{TokenPreview: "test-project-preview"},
	}, func(StoredCredential) (string, error) {
		return "", errors.New("boom")
	})
	if err == nil || err.Error() != "decrypt project GitHub credential: boom" {
		t.Fatalf("unexpected project decrypt error: %v", err)
	}
}

func TestProjectContextCredentialForScope(t *testing.T) {
	t.Parallel()

	orgCredential := &StoredCredential{TokenPreview: "org"}
	orgProbe := &TokenProbe{State: ProbeStateConfigured}
	projectCredential := &StoredCredential{TokenPreview: "project"}
	projectProbe := &TokenProbe{State: ProbeStateValid}
	projectContext := ProjectContext{
		OrganizationCredential: orgCredential,
		OrganizationProbe:      orgProbe,
		ProjectCredential:      projectCredential,
		ProjectProbe:           projectProbe,
	}

	gotCredential, gotProbe, err := projectContext.CredentialForScope(ScopeOrganization)
	if err != nil || gotCredential != orgCredential || gotProbe != orgProbe {
		t.Fatalf("CredentialForScope(organization) = %#v, %#v, %v", gotCredential, gotProbe, err)
	}

	gotCredential, gotProbe, err = projectContext.CredentialForScope(ScopeProject)
	if err != nil || gotCredential != projectCredential || gotProbe != projectProbe {
		t.Fatalf("CredentialForScope(project) = %#v, %#v, %v", gotCredential, gotProbe, err)
	}

	if _, _, err := projectContext.CredentialForScope(Scope("invalid")); err == nil {
		t.Fatal("CredentialForScope(invalid) expected error")
	}
}

func assertProbe(t *testing.T, got TokenProbe, want TokenProbe) {
	t.Helper()

	if got.State != want.State || got.Configured != want.Configured || got.Valid != want.Valid || got.RepoAccess != want.RepoAccess || got.LastError != want.LastError {
		t.Fatalf("unexpected probe: got %#v want %#v", got, want)
	}
	if len(got.Permissions) != len(want.Permissions) {
		t.Fatalf("unexpected permissions: got %#v want %#v", got.Permissions, want.Permissions)
	}
	for i := range want.Permissions {
		if got.Permissions[i] != want.Permissions[i] {
			t.Fatalf("unexpected permissions: got %#v want %#v", got.Permissions, want.Permissions)
		}
	}
	if (got.CheckedAt == nil) != (want.CheckedAt == nil) {
		t.Fatalf("unexpected checked_at nilness: got %#v want %#v", got.CheckedAt, want.CheckedAt)
	}
	if got.CheckedAt != nil && !got.CheckedAt.Equal(*want.CheckedAt) {
		t.Fatalf("unexpected checked_at: got %#v want %#v", got.CheckedAt, want.CheckedAt)
	}
}
