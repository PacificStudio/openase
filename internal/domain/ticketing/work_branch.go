package ticketing

import (
	"fmt"
	"strings"
)

type RepoWorkBranchSource string

const (
	RepoWorkBranchSourceGenerated RepoWorkBranchSource = "generated"
	RepoWorkBranchSourceOverride  RepoWorkBranchSource = "override"
)

func DefaultRepoWorkBranch(ticketIdentifier string) string {
	return fmt.Sprintf("agent/%s", strings.TrimSpace(ticketIdentifier))
}

func NormalizeRepoWorkBranchOverride(raw string) string {
	return strings.TrimSpace(raw)
}

func ResolveRepoWorkBranch(ticketIdentifier string, branchOverride string) string {
	normalized := NormalizeRepoWorkBranchOverride(branchOverride)
	if normalized != "" {
		return normalized
	}
	return DefaultRepoWorkBranch(ticketIdentifier)
}

func RepoWorkBranchSourceForOverride(branchOverride string) RepoWorkBranchSource {
	if NormalizeRepoWorkBranchOverride(branchOverride) == "" {
		return RepoWorkBranchSourceGenerated
	}
	return RepoWorkBranchSourceOverride
}
