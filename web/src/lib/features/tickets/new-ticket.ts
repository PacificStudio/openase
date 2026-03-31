import type { TicketStatus, Workflow } from '$lib/api/contracts'

export const ticketPriorityOptions = ['urgent', 'high', 'medium', 'low'] as const

export type TicketPriorityOption = (typeof ticketPriorityOptions)[number]

export type TicketOption = {
  id: string
  label: string
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
  workflowId: string
  repoId: string
}

export type NewTicketPayload = {
  title: string
  description?: string
  status_id?: string | null
  priority: TicketPriorityOption
  workflow_id?: string | null
  repo_scopes?: Array<{
    repo_id: string
    branch_name?: string | null
  }>
}

type ParsedDraft = { ok: true; payload: NewTicketPayload } | { ok: false; error: string }

export function mapTicketStatusOptions(statuses: TicketStatus[]): TicketOption[] {
  return statuses
    .slice()
    .sort((left, right) => left.position - right.position)
    .map((status) => ({
      id: status.id,
      label: status.name,
    }))
}

export function mapWorkflowOptions(workflows: Workflow[]): TicketOption[] {
  return workflows
    .slice()
    .sort((left, right) => left.name.localeCompare(right.name))
    .map((workflow) => ({
      id: workflow.id,
      label:
        workflow.name === workflow.type ? workflow.name : `${workflow.name} (${workflow.type})`,
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
  statusOptions: TicketOption[],
  workflowOptions: TicketOption[],
  repoOptions: TicketRepoOption[],
): NewTicketDraft {
  return {
    title: '',
    description: '',
    statusId: statusOptions[0]?.id ?? '',
    priority: 'medium',
    workflowId: workflowOptions[0]?.id ?? '',
    repoId: repoOptions.length === 1 ? repoOptions[0].id : '',
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
  const repoScopes = buildRepoScopes(repoOptions, draft.repoId)
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
      priority: draft.priority,
      workflow_id: draft.workflowId || null,
      repo_scopes: repoScopes.value,
    },
  }
}

function buildRepoScopes(
  repoOptions: TicketRepoOption[],
  selectedRepoId: string,
): { value: NewTicketPayload['repo_scopes'] } | { error: string } {
  if (repoOptions.length === 0) {
    return { value: undefined }
  }

  const selectedRepo =
    repoOptions.length === 1
      ? repoOptions[0]
      : (repoOptions.find((repo) => repo.id === selectedRepoId) ?? null)
  if (!selectedRepo) {
    return { error: 'Select a repository for this ticket.' }
  }

  return {
    value: [
      {
        repo_id: selectedRepo.id,
        branch_name: selectedRepo.defaultBranch || 'main',
      },
    ],
  }
}
