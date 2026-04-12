import type { TicketStatus } from '$lib/api/contracts'
import { boardPriorityValues, type BoardPriority } from '$lib/features/board/public'
import {
  buildRepoScopePayload,
  defaultRepoScopeSelection,
  mapProjectRepoOptions,
  type RepoScopeOption,
} from '$lib/features/repo-scope-selection'

export { mapProjectRepoOptions }

export const ticketPriorityOptions = boardPriorityValues

export type TicketPriorityOption = BoardPriority

export type TicketStatusOption = {
  id: string
  label: string
  color: string
  stage: 'backlog' | 'unstarted' | 'started' | 'completed' | 'canceled'
}

export type TicketRepoOption = RepoScopeOption

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

export function createNewTicketDraft(
  statusOptions: TicketStatusOption[],
  repoOptions: TicketRepoOption[],
): NewTicketDraft {
  return {
    title: '',
    description: '',
    statusId: statusOptions[0]?.id ?? '',
    priority: '',
    repoIds: defaultRepoScopeSelection(repoOptions),
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
  const repoScopes = buildRepoScopePayload(
    repoOptions,
    draft.repoIds,
    draft.repoBranchOverrides,
    'Select at least one repository scope for this ticket.',
  )
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
