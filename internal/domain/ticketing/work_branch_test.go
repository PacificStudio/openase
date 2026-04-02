package ticketing

import "testing"

func TestDefaultRepoWorkBranch(t *testing.T) {
	if got := DefaultRepoWorkBranch("  OPENASE-475  "); got != "agent/OPENASE-475" {
		t.Fatalf("DefaultRepoWorkBranch() = %q, want %q", got, "agent/OPENASE-475")
	}
}

func TestNormalizeRepoWorkBranchOverride(t *testing.T) {
	if got := NormalizeRepoWorkBranchOverride("  feature/openase-475  "); got != "feature/openase-475" {
		t.Fatalf("NormalizeRepoWorkBranchOverride() = %q, want %q", got, "feature/openase-475")
	}
}

func TestResolveRepoWorkBranch(t *testing.T) {
	testCases := []struct {
		name           string
		ticketID       string
		branchOverride string
		want           string
	}{
		{
			name:           "uses trimmed override when present",
			ticketID:       "OPENASE-475",
			branchOverride: "  feature/openase-475  ",
			want:           "feature/openase-475",
		},
		{
			name:           "falls back to generated branch when override blank",
			ticketID:       " OPENASE-475 ",
			branchOverride: "   ",
			want:           "agent/OPENASE-475",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := ResolveRepoWorkBranch(testCase.ticketID, testCase.branchOverride); got != testCase.want {
				t.Fatalf("ResolveRepoWorkBranch() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestRepoWorkBranchSourceForOverride(t *testing.T) {
	t.Run("blank override means generated", func(t *testing.T) {
		if got := RepoWorkBranchSourceForOverride("   "); got != RepoWorkBranchSourceGenerated {
			t.Fatalf("RepoWorkBranchSourceForOverride(blank) = %q, want %q", got, RepoWorkBranchSourceGenerated)
		}
	})

	t.Run("non blank override means override", func(t *testing.T) {
		if got := RepoWorkBranchSourceForOverride("existing/topic"); got != RepoWorkBranchSourceOverride {
			t.Fatalf("RepoWorkBranchSourceForOverride(non-blank) = %q, want %q", got, RepoWorkBranchSourceOverride)
		}
	})
}
