import type { TicketDetail } from './types'

export const dependencyRelationOptions = [
  { value: 'blocks', label: 'Blocks' },
  { value: 'blocked_by', label: 'Blocked by' },
  { value: 'sub_issue', label: 'Sub-issue' },
] as const

export const dependencyRelationActions = [
  { relation: 'sub_issue', label: 'Add parent', description: 'This ticket becomes a sub-issue' },
  { relation: 'blocks', label: 'Mark as blocking', description: 'This ticket blocks another' },
  {
    relation: 'blocked_by',
    label: 'Mark as blocked by',
    description: 'Another ticket blocks this one',
  },
] as const

export type TicketFieldDraft = {
  title: string
  description: string
  statusId: string
}

export type DependencyDraft = {
  targetTicketId: string
  relation: TicketDetail['dependencies'][number]['relation']
}

export type RepoScopeDraft = {
  repoId: string
  branchName: string
  pullRequestUrl: string
}

export type ExistingRepoScopeDraft = Omit<RepoScopeDraft, 'repoId'>

export type ParseResult<T> = { ok: true; value: T } | { ok: false; error: string }

export function cloneTicketDetail(source: TicketDetail): TicketDetail {
  return {
    ...source,
    status: { ...source.status },
    workflow: source.workflow ? { ...source.workflow } : undefined,
    assignedAgent: source.assignedAgent ? { ...source.assignedAgent } : undefined,
    pickupDiagnosis: source.pickupDiagnosis
      ? {
          ...source.pickupDiagnosis,
          reasons: source.pickupDiagnosis.reasons.map((reason) => ({ ...reason })),
          workflow: source.pickupDiagnosis.workflow
            ? { ...source.pickupDiagnosis.workflow }
            : undefined,
          agent: source.pickupDiagnosis.agent ? { ...source.pickupDiagnosis.agent } : undefined,
          provider: source.pickupDiagnosis.provider
            ? { ...source.pickupDiagnosis.provider }
            : undefined,
          retry: { ...source.pickupDiagnosis.retry },
          capacity: {
            workflow: { ...source.pickupDiagnosis.capacity.workflow },
            project: { ...source.pickupDiagnosis.capacity.project },
            provider: { ...source.pickupDiagnosis.capacity.provider },
            status: { ...source.pickupDiagnosis.capacity.status },
          },
          blockedBy: source.pickupDiagnosis.blockedBy.map((blocker) => ({ ...blocker })),
        }
      : undefined,
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
  return appendedScope ? [...repoScopes, appendedScope] : repoScopes
}
