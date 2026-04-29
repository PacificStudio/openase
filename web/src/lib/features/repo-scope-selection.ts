export type RepoScopeOption = {
  id: string
  label: string
  defaultBranch: string
}

export type RepoScopePayload = Array<{
  repo_id: string
  branch_name?: string | null
}>

export function mapProjectRepoOptions(
  repos: Array<{
    id: string
    name: string
    default_branch: string
  }>,
): RepoScopeOption[] {
  return repos
    .slice()
    .sort((left, right) => left.name.localeCompare(right.name))
    .map((repo) => ({
      id: repo.id,
      label: repo.name,
      defaultBranch: repo.default_branch || 'main',
    }))
}

export function defaultRepoScopeSelection(repoOptions: RepoScopeOption[]): string[] {
  return repoOptions.length === 1 ? [repoOptions[0].id] : []
}

export function buildRepoScopePayload(
  repoOptions: RepoScopeOption[],
  selectedRepoIds: string[],
  repoBranchOverrides: Record<string, string>,
  emptySelectionError: string,
): { value: RepoScopePayload | undefined } | { error: string } {
  if (repoOptions.length === 0) {
    return { value: undefined }
  }

  if (repoOptions.length === 1) {
    const repo = repoOptions[0]
    const branchOverride = repoBranchOverrides[repo.id]?.trim()
    return {
      value: [
        {
          repo_id: repo.id,
          branch_name: branchOverride || undefined,
        },
      ],
    }
  }

  const selectedRepos = repoOptions.filter((repo) => selectedRepoIds.includes(repo.id))
  if (selectedRepos.length === 0) {
    return { error: emptySelectionError }
  }

  return {
    value: selectedRepos.map((repo) => ({
      repo_id: repo.id,
      branch_name: repoBranchOverrides[repo.id]?.trim() || undefined,
    })),
  }
}
