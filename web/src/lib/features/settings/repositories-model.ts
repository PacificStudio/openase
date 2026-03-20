import type { ProjectRepoRecord } from '$lib/api/contracts'

export type RepositoryDraft = {
  name: string
  repositoryURL: string
  defaultBranch: string
  clonePath: string
  labels: string
  isPrimary: boolean
}

export type RepositoryMutationInput = {
  name: string
  repository_url: string
  default_branch: string
  clone_path: string | null
  labels: string[]
  is_primary: boolean
}

export type RepositoryDraftParseResult =
  | { ok: true; value: RepositoryMutationInput }
  | { ok: false; error: string }

export type RepositoryEditorMode = 'create' | 'edit'

export function createEmptyRepositoryDraft(options: { isPrimary?: boolean } = {}): RepositoryDraft {
  return {
    name: '',
    repositoryURL: '',
    defaultBranch: 'main',
    clonePath: '',
    labels: '',
    isPrimary: options.isPrimary ?? false,
  }
}

export function projectRepoToDraft(repo: ProjectRepoRecord): RepositoryDraft {
  return {
    name: repo.name,
    repositoryURL: repo.repository_url,
    defaultBranch: repo.default_branch || 'main',
    clonePath: repo.clone_path ?? '',
    labels: repo.labels.join(', '),
    isPrimary: repo.is_primary,
  }
}

export function parseRepositoryDraft(draft: RepositoryDraft): RepositoryDraftParseResult {
  const name = draft.name.trim()
  if (!name) {
    return { ok: false, error: 'Repository name is required.' }
  }

  const repositoryURL = draft.repositoryURL.trim()
  if (!repositoryURL) {
    return { ok: false, error: 'Repository URL is required.' }
  }

  return {
    ok: true,
    value: {
      name,
      repository_url: repositoryURL,
      default_branch: draft.defaultBranch.trim() || 'main',
      clone_path: draft.clonePath.trim() || null,
      labels: splitLabels(draft.labels),
      is_primary: draft.isPrimary,
    },
  }
}

export function sortProjectRepos(repos: ProjectRepoRecord[]): ProjectRepoRecord[] {
  return repos
    .slice()
    .sort(
      (left, right) =>
        Number(right.is_primary) - Number(left.is_primary) ||
        left.name.localeCompare(right.name, undefined, { sensitivity: 'base' }),
    )
}

function splitLabels(raw: string): string[] {
  const labels: string[] = []
  const seen = new Set<string>()

  for (const item of raw.split(/[\n,]/)) {
    const trimmed = item.trim()
    if (!trimmed || seen.has(trimmed)) {
      continue
    }

    seen.add(trimmed)
    labels.push(trimmed)
  }

  return labels
}
