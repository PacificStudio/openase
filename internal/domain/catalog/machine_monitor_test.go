package catalog

import (
	"testing"
	"time"
)

func TestParseMachineAgentEnvironment(t *testing.T) {
	collectedAt := time.Date(2026, 3, 20, 18, 30, 0, 0, time.UTC)
	raw := "claude_code\tfalse\t\tunknown\tunknown\n" +
		"codex\ttrue\t0.0.1\tunknown\tapi_key\n" +
		"gemini\ttrue\t1.2.3\tunknown\tunknown\n"

	environment, err := ParseMachineAgentEnvironment(raw, collectedAt)
	if err != nil {
		t.Fatalf("parse agent environment: %v", err)
	}
	if !environment.Dispatchable {
		t.Fatalf("expected environment to be dispatchable, got %+v", environment)
	}
	if len(environment.CLIs) != 3 {
		t.Fatalf("expected three cli snapshots, got %+v", environment.CLIs)
	}
	if environment.CLIs[1].Name != "codex" || environment.CLIs[1].Version != "0.0.1" || environment.CLIs[1].AuthMode != MachineAgentAuthModeAPIKey || !environment.CLIs[1].Ready {
		t.Fatalf("expected codex snapshot to be parsed, got %+v", environment.CLIs[1])
	}
	if environment.CLIs[2].Name != "gemini" || !environment.CLIs[2].Ready {
		t.Fatalf("expected gemini installed with unknown auth to remain usable, got %+v", environment.CLIs[2])
	}
}

func TestParseMachineFullAudit(t *testing.T) {
	collectedAt := time.Date(2026, 3, 20, 18, 45, 0, 0, time.UTC)
	raw := "git\ttrue\tOpenASE\topenase@example.com\n" +
		"gh_cli\ttrue\tnot_logged_in\n" +
		"github_token_probe\ttrue\tvalid\ttrue\trepo,read:org\tgranted\t\n" +
		"network\ttrue\tfalse\ttrue\n"

	audit, err := ParseMachineFullAudit(raw, collectedAt)
	if err != nil {
		t.Fatalf("parse full audit: %v", err)
	}
	if !audit.Git.Installed || audit.Git.UserEmail != "openase@example.com" {
		t.Fatalf("expected git audit to parse, got %+v", audit.Git)
	}
	if audit.GitHubCLI.AuthStatus != MachineAgentAuthStatusNotLoggedIn {
		t.Fatalf("expected gh auth status to parse, got %+v", audit.GitHubCLI)
	}
	if audit.GitHubTokenProbe.State != "valid" || !audit.GitHubTokenProbe.Valid || audit.GitHubTokenProbe.RepoAccess != "granted" {
		t.Fatalf("expected github token probe to parse, got %+v", audit.GitHubTokenProbe)
	}
	if !audit.Network.GitHubReachable || audit.Network.PyPIReachable || !audit.Network.NPMReachable {
		t.Fatalf("expected network audit to parse, got %+v", audit.Network)
	}
}

func TestParseMachineFullAuditDefaultsMissingGitHubTokenProbe(t *testing.T) {
	collectedAt := time.Date(2026, 3, 21, 8, 15, 0, 0, time.UTC)
	raw := "git\ttrue\tOpenASE\topenase@example.com\n" +
		"gh_cli\tfalse\tlogged_in\n" +
		"network\ttrue\ttrue\ttrue\n"

	audit, err := ParseMachineFullAudit(raw, collectedAt)
	if err != nil {
		t.Fatalf("parse full audit without github token probe: %v", err)
	}
	if audit.GitHubTokenProbe.State != "missing" || audit.GitHubTokenProbe.Configured || audit.GitHubTokenProbe.Valid {
		t.Fatalf("expected missing github token probe, got %+v", audit.GitHubTokenProbe)
	}
	if audit.GitHubTokenProbe.CheckedAt == nil || !audit.GitHubTokenProbe.CheckedAt.Equal(collectedAt.UTC()) {
		t.Fatalf("expected missing probe checked_at to use collection time, got %+v", audit.GitHubTokenProbe.CheckedAt)
	}
}

func TestParseMachineFullAuditRejectsInvalidGitHubTokenProbeState(t *testing.T) {
	collectedAt := time.Date(2026, 3, 21, 8, 30, 0, 0, time.UTC)
	raw := "git\ttrue\tOpenASE\topenase@example.com\n" +
		"gh_cli\ttrue\tnot_logged_in\n" +
		"github_token_probe\ttrue\tbogus\ttrue\trepo\tgranted\t\n" +
		"network\ttrue\ttrue\ttrue\n"

	_, err := ParseMachineFullAudit(raw, collectedAt)
	if err == nil || err.Error() != "parse github_token_probe state on row 2: invalid state \"bogus\"" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMachineFullAuditRejectsInvalidGitHubTokenProbeRepoAccess(t *testing.T) {
	collectedAt := time.Date(2026, 3, 21, 8, 45, 0, 0, time.UTC)
	raw := "git\ttrue\tOpenASE\topenase@example.com\n" +
		"gh_cli\ttrue\tnot_logged_in\n" +
		"github_token_probe\ttrue\tvalid\ttrue\trepo\tbogus\t\n" +
		"network\ttrue\ttrue\ttrue\n"

	_, err := ParseMachineFullAudit(raw, collectedAt)
	if err == nil || err.Error() != "parse github_token_probe repo access on row 2: invalid repo access \"bogus\"" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMachineFullAuditRejectsInvalidGitHubTokenProbeShape(t *testing.T) {
	collectedAt := time.Date(2026, 3, 21, 9, 0, 0, 0, time.UTC)
	raw := "git\ttrue\tOpenASE\topenase@example.com\n" +
		"gh_cli\ttrue\tnot_logged_in\n" +
		"github_token_probe\ttrue\tvalid\ttrue\trepo\tgranted\n" +
		"network\ttrue\ttrue\ttrue\n"

	_, err := ParseMachineFullAudit(raw, collectedAt)
	if err == nil || err.Error() != "github_token_probe row 2 must have 7 columns" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMachineFullAuditRejectsInvalidGitHubTokenProbeConfiguredFlag(t *testing.T) {
	collectedAt := time.Date(2026, 3, 21, 9, 15, 0, 0, time.UTC)
	raw := "git\ttrue\tOpenASE\topenase@example.com\n" +
		"gh_cli\ttrue\tnot_logged_in\n" +
		"github_token_probe\tnope\tvalid\ttrue\trepo\tgranted\t\n" +
		"network\ttrue\ttrue\ttrue\n"

	_, err := ParseMachineFullAudit(raw, collectedAt)
	if err == nil || err.Error() != "parse github_token_probe configured on row 2: strconv.ParseBool: parsing \"nope\": invalid syntax" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMachineFullAuditRejectsInvalidGitHubTokenProbeValidFlag(t *testing.T) {
	collectedAt := time.Date(2026, 3, 21, 9, 30, 0, 0, time.UTC)
	raw := "git\ttrue\tOpenASE\topenase@example.com\n" +
		"gh_cli\ttrue\tnot_logged_in\n" +
		"github_token_probe\ttrue\tvalid\tnope\trepo\tgranted\t\n" +
		"network\ttrue\ttrue\ttrue\n"

	_, err := ParseMachineFullAudit(raw, collectedAt)
	if err == nil || err.Error() != "parse github_token_probe valid on row 2: strconv.ParseBool: parsing \"nope\": invalid syntax" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseDelimitedMachinePermissions(t *testing.T) {
	if got := parseDelimitedMachinePermissions(""); got != nil {
		t.Fatalf("expected empty permissions for blank input, got %#v", got)
	}
	if got := parseDelimitedMachinePermissions("-"); got != nil {
		t.Fatalf("expected empty permissions for sentinel input, got %#v", got)
	}

	got := parseDelimitedMachinePermissions("repo, read:org, , issues:write ")
	if len(got) != 3 || got[0] != "repo" || got[1] != "read:org" || got[2] != "issues:write" {
		t.Fatalf("unexpected parsed permissions: %#v", got)
	}
}
