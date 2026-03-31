import type { TicketDetail, TicketRepoOption } from './types'
import {
  nextRepoScopesForMutation,
  repoScopeCiStatusOptions,
  repoScopePrStatusOptions,
  type ExistingRepoScopeDraft,
  type ParseResult,
  type RepoScopeDraft,
} from './mutation-shared'

export function buildCreateRepoScopeMutation(
  availableRepos: TicketRepoOption[],
  draft: RepoScopeDraft,
): ParseResult<{
  body: {
    repo_id: string
    branch_name: string
    pull_request_url: string | null
    pr_status?: string
    ci_status?: string
  }
  optimisticScope: TicketDetail['repoScopes'][number]
  successMessage: string
}> {
  const repo = availableRepos.find((item) => item.id === draft.repoId)
  if (!repo) {
    return { ok: false, error: 'Select a valid repository.' }
  }

  const branchName = draft.branchName.trim() || repo.defaultBranch.trim()
  if (!branchName) {
    return { ok: false, error: 'Branch name is required.' }
  }

  const prStatus = parseOptionalSelection(
    draft.prStatus,
    repoScopePrStatusOptions.map((option) => option.value),
    'pull request status',
  )
  if (!prStatus.ok) {
    return prStatus
  }

  const ciStatus = parseOptionalSelection(
    draft.ciStatus,
    repoScopeCiStatusOptions.map((option) => option.value),
    'CI status',
  )
  if (!ciStatus.ok) {
    return ciStatus
  }

  const pullRequestUrl = normalizeOptionalText(draft.pullRequestUrl)

  return {
    ok: true,
    value: {
      body: {
        repo_id: repo.id,
        branch_name: branchName,
        pull_request_url: pullRequestUrl,
        pr_status: prStatus.value ?? undefined,
        ci_status: ciStatus.value ?? undefined,
      },
      optimisticScope: {
        id: `pending-${repo.id}`,
        repoId: repo.id,
        repoName: repo.name,
        branchName,
        prUrl: pullRequestUrl ?? undefined,
        prStatus: prStatus.value ?? undefined,
        ciStatus: ciStatus.value ?? undefined,
      },
      successMessage: `Repo scope added for ${repo.name}.`,
    },
  }
}

export function buildUpdateRepoScopeMutation(
  currentTicket: TicketDetail,
  scopeId: string,
  draft: ExistingRepoScopeDraft,
): ParseResult<{
  body: {
    branch_name: string
    pull_request_url: string | null
    pr_status: string | null
    ci_status: string | null
  }
  optimisticUpdate: (ticket: TicketDetail) => TicketDetail
  successMessage: string
}> {
  const scope = currentTicket.repoScopes.find((item) => item.id === scopeId)
  if (!scope) {
    return { ok: false, error: 'Repo scope no longer exists in the current ticket state.' }
  }

  const branchName = draft.branchName.trim()
  if (!branchName) {
    return { ok: false, error: 'Branch name is required.' }
  }

  const prStatus = parseOptionalSelection(
    draft.prStatus,
    repoScopePrStatusOptions.map((option) => option.value),
    'pull request status',
  )
  if (!prStatus.ok) {
    return prStatus
  }

  const ciStatus = parseOptionalSelection(
    draft.ciStatus,
    repoScopeCiStatusOptions.map((option) => option.value),
    'CI status',
  )
  if (!ciStatus.ok) {
    return ciStatus
  }

  const pullRequestUrl = normalizeOptionalText(draft.pullRequestUrl)

  return {
    ok: true,
    value: {
      body: {
        branch_name: branchName,
        pull_request_url: pullRequestUrl,
        pr_status: prStatus.value,
        ci_status: ciStatus.value,
      },
      optimisticUpdate: (ticket) => ({
        ...ticket,
        repoScopes: nextRepoScopesForMutation(
          ticket.repoScopes.map((item) =>
            item.id === scopeId
              ? {
                  ...item,
                  branchName,
                  prUrl: pullRequestUrl ?? undefined,
                  prStatus: prStatus.value ?? undefined,
                  ciStatus: ciStatus.value ?? undefined,
                }
              : item,
          ),
        ),
      }),
      successMessage: `Repo scope updated for ${scope.repoName}.`,
    },
  }
}

export function buildDeleteRepoScopeMutation(
  currentTicket: TicketDetail,
  scopeId: string,
): ParseResult<{
  optimisticUpdate: (ticket: TicketDetail) => TicketDetail
  successMessage: string
}> {
  const scope = currentTicket.repoScopes.find((item) => item.id === scopeId)
  if (!scope) {
    return { ok: false, error: 'Repo scope no longer exists in the current ticket state.' }
  }

  return {
    ok: true,
    value: {
      optimisticUpdate: (ticket) => ({
        ...ticket,
        repoScopes: ticket.repoScopes.filter((item) => item.id !== scopeId),
      }),
      successMessage: `Repo scope removed for ${scope.repoName}.`,
    },
  }
}

function parseOptionalSelection<T extends string>(
  rawValue: string,
  allowedValues: readonly T[],
  label: string,
): ParseResult<T | null> {
  if (!rawValue) {
    return { ok: true, value: null }
  }
  if (!allowedValues.includes(rawValue as T)) {
    return { ok: false, error: `Select a valid ${label}.` }
  }

  return { ok: true, value: rawValue as T }
}

function normalizeOptionalText(value: string) {
  const normalized = value.trim()
  return normalized ? normalized : null
}
