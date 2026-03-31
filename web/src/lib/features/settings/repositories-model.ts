import type { ProjectRepoRecord } from '$lib/api/contracts'

export type RepositoryDraft = {
  name: string
  repositoryURL: string
  defaultBranch: string
  workspaceDirname: string
  labels: string
}

export type RepositoryMutationInput = {
  name: string
  repository_url: string
  default_branch: string
  workspace_dirname: string | null
  labels: string[]
}

export type RepositoryDraftParseResult =
  | { ok: true; value: RepositoryMutationInput }
  | { ok: false; error: string }

export type RepositoryEditorMode = 'create' | 'edit'

export function createEmptyRepositoryDraft(): RepositoryDraft {
  return {
    name: '',
    repositoryURL: '',
    defaultBranch: 'main',
    workspaceDirname: '',
    labels: '',
  }
}

export function projectRepoToDraft(repo: ProjectRepoRecord): RepositoryDraft {
  const labels = Array.isArray((repo as { labels?: unknown }).labels) ? repo.labels : []

  return {
    name: repo.name,
    repositoryURL: repo.repository_url,
    defaultBranch: repo.default_branch || 'main',
    workspaceDirname: repo.workspace_dirname ?? '',
    labels: labels.join(', '),
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
      workspace_dirname: draft.workspaceDirname.trim() || null,
      labels: splitLabels(draft.labels),
    },
  }
}

export function sortProjectRepos(repos: ProjectRepoRecord[]): ProjectRepoRecord[] {
  return repos
    .slice()
    .sort((left, right) => left.name.localeCompare(right.name, undefined, { sensitivity: 'base' }))
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
