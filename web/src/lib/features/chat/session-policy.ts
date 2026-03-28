import type { ChatDonePayload } from '$lib/api/chat'

export const EPHEMERAL_CHAT_MAX_TURNS = 10
export const EPHEMERAL_CHAT_MAX_BUDGET_USD = 2

export function formatEphemeralChatUsageSummary(payload: ChatDonePayload) {
  const usage = `Session budget: ${payload.turnsUsed}/${EPHEMERAL_CHAT_MAX_TURNS} turns used, ${payload.turnsRemaining} remaining.`
  if (payload.costUSD === undefined) {
    return `${usage} Spend unavailable for this provider; the chat budget cap remains $${EPHEMERAL_CHAT_MAX_BUDGET_USD.toFixed(2)}.`
  }

  return `${usage} Current spend $${payload.costUSD.toFixed(2)} of $${EPHEMERAL_CHAT_MAX_BUDGET_USD.toFixed(2)}.`
}
