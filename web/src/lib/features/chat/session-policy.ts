import type { ChatDonePayload, ChatSource } from '$lib/api/chat'

export const EPHEMERAL_CHAT_MAX_TURNS = 10
export const EPHEMERAL_CHAT_MAX_BUDGET_USD = 2

export function describeEphemeralChatSessionPolicy(source: ChatSource, hasSession: boolean) {
  if (source === 'project_sidebar') {
    if (hasSession) {
      return 'Follow-up prompts reuse the current project conversation until you close the sheet or switch providers.'
    }

    return 'The first reply starts a grouped project conversation. Assistant markdown streams into a single reply block per turn.'
  }

  return hasSession
    ? `Session cap: ${EPHEMERAL_CHAT_MAX_TURNS} turns / $${EPHEMERAL_CHAT_MAX_BUDGET_USD.toFixed(2)}. Follow-up prompts reuse the current chat context until you close the panel, switch providers, or hit the limit.`
    : `Session cap: ${EPHEMERAL_CHAT_MAX_TURNS} turns / $${EPHEMERAL_CHAT_MAX_BUDGET_USD.toFixed(2)}. The first reply starts a new ephemeral chat session.`
}

export function formatEphemeralChatUsageSummary(source: ChatSource, payload: ChatDonePayload) {
  if (source === 'project_sidebar') {
    const usage = `Project conversation: ${payload.turnsUsed} turn${payload.turnsUsed === 1 ? '' : 's'} so far.`
    if (payload.costUSD === undefined) {
      return `${usage} Spend unavailable for this provider.`
    }

    return `${usage} Current spend $${payload.costUSD.toFixed(2)}.`
  }

  const turnsRemaining = payload.turnsRemaining ?? 0
  const usage = `Session budget: ${payload.turnsUsed}/${EPHEMERAL_CHAT_MAX_TURNS} turns used, ${turnsRemaining} remaining.`
  if (payload.costUSD === undefined) {
    return `${usage} Spend unavailable for this provider; the chat budget cap remains $${EPHEMERAL_CHAT_MAX_BUDGET_USD.toFixed(2)}.`
  }

  return `${usage} Current spend $${payload.costUSD.toFixed(2)} of $${EPHEMERAL_CHAT_MAX_BUDGET_USD.toFixed(2)}.`
}
