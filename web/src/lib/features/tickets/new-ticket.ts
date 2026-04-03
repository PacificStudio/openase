import type { TicketStatus } from '$lib/api/contracts'
import { boardPriorityValues, type BoardPriority } from '$lib/features/board/public'

export const ticketPriorityOptions = boardPriorityValues

export type TicketPriorityOption = BoardPriority

export type TicketStatusOption = {
  id: string
  label: string
  color: string
  stage: 'backlog' | 'unstarted' | 'started' | 'completed' | 'canceled'
}

export type TicketRepoOption = {
  id: string
  label: string
  defaultBranch: string
}

export type NewTicketDraft = {
  title: string
  description: string
  statusId: string
  priority: TicketPriorityOption
  repoIds: string[]
  repoBranchOverrides: Record<string, string>
}

export type NewTicketPayload = {
  title: string
  description?: string
  status_id?: string | null
  priority?: Exclude<TicketPriorityOption, ''> | null
  repo_scopes?: Array<{
    repo_id: string
    branch_name?: string | null
  }>
}

type ParsedDraft = { ok: true; payload: NewTicketPayload } | { ok: false; error: string }

export function mapTicketStatusOptions(statuses: TicketStatus[]): TicketStatusOption[] {
  return statuses
    .slice()
    .sort((left, right) => left.position - right.position)
    .map((status) => ({
      id: status.id,
      label: status.name,
      color: status.color || '#94a3b8',
      stage: (status.stage || 'unstarted') as TicketStatusOption['stage'],
    }))
}

export function mapProjectRepoOptions(
  repos: Array<{
    id: string
    name: string
    default_branch: string
  }>,
): TicketRepoOption[] {
  return repos
    .slice()
    .sort((left, right) => left.name.localeCompare(right.name))
    .map((repo) => ({
      id: repo.id,
      label: repo.name,
      defaultBranch: repo.default_branch || 'main',
    }))
}

export function createNewTicketDraft(
  statusOptions: TicketStatusOption[],
  repoOptions: TicketRepoOption[],
): NewTicketDraft {
  return {
    title: '',
    description: '',
    statusId: statusOptions[0]?.id ?? '',
    priority: '',
    repoIds: repoOptions.length === 1 ? [repoOptions[0].id] : [],
    repoBranchOverrides: {},
  }
}

export function parseNewTicketDraft(
  draft: NewTicketDraft,
  repoOptions: TicketRepoOption[],
): ParsedDraft {
  const title = draft.title.trim()
  if (!title) {
    return {
      ok: false,
      error: 'Title is required.',
    }
  }

  const description = draft.description.trim()
  const repoScopes = buildRepoScopes(repoOptions, draft.repoIds, draft.repoBranchOverrides)
  if ('error' in repoScopes) {
    return {
      ok: false,
      error: repoScopes.error,
    }
  }

  return {
    ok: true,
    payload: {
      title,
      description: description || undefined,
      status_id: draft.statusId || null,
      priority: draft.priority || null,
      repo_scopes: repoScopes.value,
    },
  }
}

function buildRepoScopes(
  repoOptions: TicketRepoOption[],
  selectedRepoIds: string[],
  repoBranchOverrides: Record<string, string>,
): { value: NewTicketPayload['repo_scopes'] } | { error: string } {
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
    return { error: 'Select at least one repository scope for this ticket.' }
  }

  return {
    value: selectedRepos.map((repo) => ({
      repo_id: repo.id,
      branch_name: repoBranchOverrides[repo.id]?.trim() || undefined,
    })),
  }
}
