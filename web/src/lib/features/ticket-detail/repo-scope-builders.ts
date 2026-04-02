import type { TicketDetail, TicketRepoOption } from './types'
import {
  nextRepoScopesForMutation,
  type ExistingRepoScopeDraft,
  type ParseResult,
  type RepoScopeDraft,
} from './mutation-shared'

export function buildCreateRepoScopeMutation(
  currentTicket: Pick<TicketDetail, 'identifier'>,
  availableRepos: TicketRepoOption[],
  draft: RepoScopeDraft,
): ParseResult<{
  body: {
    repo_id: string
    branch_name: string | null
    pull_request_url: string | null
  }
  optimisticScope: TicketDetail['repoScopes'][number]
  successMessage: string
}> {
  const repo = availableRepos.find((item) => item.id === draft.repoId)
  if (!repo) {
    return { ok: false, error: 'Select a valid repository.' }
  }

  const branchName = normalizeOptionalText(draft.branchName)
  const pullRequestUrl = normalizeOptionalText(draft.pullRequestUrl)

  return {
    ok: true,
    value: {
      body: {
        repo_id: repo.id,
        branch_name: branchName,
        pull_request_url: pullRequestUrl,
      },
      optimisticScope: {
        id: `pending-${repo.id}`,
        repoId: repo.id,
        repoName: repo.name,
        branchName: branchName ?? '',
        defaultBranch: repo.defaultBranch,
        effectiveBranchName: branchName ?? generatedTicketBranchName(currentTicket.identifier),
        branchSource: branchName ? 'override' : 'generated',
        prUrl: pullRequestUrl ?? undefined,
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
    branch_name: string | null
    pull_request_url: string | null
  }
  optimisticUpdate: (ticket: TicketDetail) => TicketDetail
  successMessage: string
}> {
  const scope = currentTicket.repoScopes.find((item) => item.id === scopeId)
  if (!scope) {
    return { ok: false, error: 'Repo scope no longer exists in the current ticket state.' }
  }

  const branchName = normalizeOptionalText(draft.branchName)
  const pullRequestUrl = normalizeOptionalText(draft.pullRequestUrl)

  return {
    ok: true,
    value: {
      body: {
        branch_name: branchName,
        pull_request_url: pullRequestUrl,
      },
      optimisticUpdate: (ticket) => ({
        ...ticket,
        repoScopes: nextRepoScopesForMutation(
          ticket.repoScopes.map((item) =>
            item.id === scopeId
              ? {
                  ...item,
                  branchName: branchName ?? '',
                  effectiveBranchName:
                    branchName ?? generatedTicketBranchName(currentTicket.identifier),
                  branchSource: branchName ? 'override' : 'generated',
                  prUrl: pullRequestUrl ?? undefined,
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

function normalizeOptionalText(value: string) {
  const normalized = value.trim()
  return normalized ? normalized : null
}

function generatedTicketBranchName(ticketIdentifier: string) {
  return `agent/${ticketIdentifier.trim()}`
}
