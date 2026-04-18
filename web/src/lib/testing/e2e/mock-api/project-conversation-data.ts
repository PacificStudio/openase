import { asObject, asString } from './helpers'

export function buildMockProjectConversationReply(
  message: string,
  ticketFocus: {
    identifier: string
    status: string
    retryPaused: boolean
    pauseReason: string
    repoScopes: unknown[]
    hookHistory: unknown[]
    currentRun: Record<string, unknown> | null
  } | null,
) {
  if (!ticketFocus) {
    return `Mock assistant reply for: ${message}`
  }

  const normalized = message.toLowerCase()
  if (normalized.includes('why is this ticket not running')) {
    const currentRunStatus = asString(ticketFocus.currentRun?.status) ?? 'unknown'
    const lastError = asString(ticketFocus.currentRun?.last_error) ?? 'unknown'
    return `${ticketFocus.identifier} is currently ${ticketFocus.status}. Retries are paused=${ticketFocus.retryPaused} because "${ticketFocus.pauseReason}". The latest run status was ${currentRunStatus} and the latest failure was "${lastError}".`
  }

  if (normalized.includes('which repos does this ticket currently affect')) {
    const scopes = ticketFocus.repoScopes
      .map((scope) => asObject(scope))
      .filter((scope): scope is Record<string, unknown> => scope !== null)
      .map((scope) =>
        [
          asString(scope.repo_name) ?? asString(scope.repo_id) ?? 'unknown-repo',
          asString(scope.branch_name),
        ]
          .filter(Boolean)
          .join(' @ '),
      )
      .filter(Boolean)
    return `${ticketFocus.identifier} currently affects ${scopes.join(', ')}.`
  }

  if (normalized.includes('what hook failed most recently')) {
    const latestHook = ticketFocus.hookHistory
      .map((hook) => asObject(hook))
      .filter((hook): hook is Record<string, unknown> => hook !== null)
      .at(-1)
    return latestHook
      ? `The latest hook was ${asString(latestHook.hook_name) ?? 'unknown'} and it reported "${asString(latestHook.output) ?? ''}".`
      : `No hook history is available for ${ticketFocus.identifier}.`
  }

  return `Mock assistant reply for ${ticketFocus.identifier}: ${message}`
}

export function buildMockProjectConversationWorkspaceDiff(conversationId: string) {
  return {
    conversation_id: conversationId,
    workspace_path: `/tmp/${conversationId}`,
    preparing: false,
    dirty: false,
    repos_changed: 0,
    files_changed: 0,
    added: 0,
    removed: 0,
    repos: [],
  }
}
