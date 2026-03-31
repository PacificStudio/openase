import type { TicketDetail } from './types'

export const dependencyRelationOptions = [
  { value: 'blocks', label: 'Blocks' },
  { value: 'blocked_by', label: 'Blocked by' },
  { value: 'sub_issue', label: 'Sub-issue' },
] as const

export const dependencyRelationActions = [
  { relation: 'sub_issue', label: 'Add parent', description: 'This ticket becomes a sub-issue' },
  {
    relation: 'blocked_by',
    label: 'Add blocked by',
    description: 'This ticket is blocked by another',
  },
  { relation: 'blocks', label: 'Mark as blocking', description: 'This ticket blocks another' },
] as const

export const repoScopePrStatusOptions = [
  { value: '', label: 'Unset' },
  { value: 'open', label: 'Open' },
  { value: 'draft', label: 'Draft' },
  { value: 'merged', label: 'Merged' },
  { value: 'closed', label: 'Closed' },
] as const

export const repoScopeCiStatusOptions = [
  { value: '', label: 'Unset' },
  { value: 'pending', label: 'Pending' },
  { value: 'running', label: 'Running' },
  { value: 'pass', label: 'Pass' },
  { value: 'fail', label: 'Fail' },
] as const

export type TicketFieldDraft = {
  title: string
  description: string
  statusId: string
}

export type DependencyDraft = {
  targetTicketId: string
  relation: string
}

export type RepoScopeDraft = {
  repoId: string
  branchName: string
  pullRequestUrl: string
  prStatus: string
  ciStatus: string
  isPrimaryScope: boolean
}

export type ExistingRepoScopeDraft = Omit<RepoScopeDraft, 'repoId'>

export type ParseResult<T> = { ok: true; value: T } | { ok: false; error: string }

export function cloneTicketDetail(source: TicketDetail): TicketDetail {
  return {
    ...source,
    status: { ...source.status },
    workflow: source.workflow ? { ...source.workflow } : undefined,
    assignedAgent: source.assignedAgent ? { ...source.assignedAgent } : undefined,
    repoScopes: source.repoScopes.map((scope) => ({ ...scope })),
    dependencies: source.dependencies.map((dependency) => ({ ...dependency })),
    externalLinks: source.externalLinks.map((link) => ({ ...link })),
    children: source.children.map((child) => ({ ...child })),
  }
}

export function nextRepoScopesForMutation(
  repoScopes: TicketDetail['repoScopes'],
  appendedScope?: TicketDetail['repoScopes'][number],
) {
  const nextScopes = appendedScope ? [...repoScopes, appendedScope] : repoScopes
  const primaryScope = nextScopes.find((scope) => scope.isPrimaryScope)
  if (!primaryScope) {
    return nextScopes
  }

  return nextScopes.map((scope) => ({
    ...scope,
    isPrimaryScope: scope.id === primaryScope.id,
  }))
}
