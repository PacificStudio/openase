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
