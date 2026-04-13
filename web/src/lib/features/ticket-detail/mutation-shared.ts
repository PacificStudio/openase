import type { TicketDetail } from './types'
import { i18nStore } from '$lib/i18n/store.svelte'

export const dependencyRelationOptions = [
  {
    value: 'blocks',
    get label() {
      return i18nStore.t('ticketDetail.dependencies.blocks')
    },
  },
  {
    value: 'blocked_by',
    get label() {
      return i18nStore.t('ticketDetail.dependencies.blockedBy')
    },
  },
  {
    value: 'sub_issue',
    get label() {
      return i18nStore.t('ticketDetail.dependencies.subIssue')
    },
  },
] as const

export const dependencyRelationActions = [
  {
    relation: 'sub_issue',
    get label() {
      return i18nStore.t('ticketDetail.dependencies.addParent')
    },
    get description() {
      return i18nStore.t('ticketDetail.dependencies.addParentDescription')
    },
  },
  {
    relation: 'blocks',
    get label() {
      return i18nStore.t('ticketDetail.dependencies.markAsBlocking')
    },
    get description() {
      return i18nStore.t('ticketDetail.dependencies.markAsBlockingDescription')
    },
  },
  {
    relation: 'blocked_by',
    get label() {
      return i18nStore.t('ticketDetail.dependencies.markAsBlockedBy')
    },
    get description() {
      return i18nStore.t('ticketDetail.dependencies.markAsBlockedByDescription')
    },
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
