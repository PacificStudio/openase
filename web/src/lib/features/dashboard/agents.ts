import type { Agent } from '$lib/features/workspace'

export function upsertAgent(items: Agent[], patch: Partial<Agent> & { id: string }) {
  const index = items.findIndex((item) => item.id === patch.id)
  if (index === -1) {
    if (!patch.name || !patch.project_id || !patch.provider_id || !patch.status) {
      return items
    }

    return [
      ...items,
      {
        id: patch.id,
        provider_id: patch.provider_id,
        project_id: patch.project_id,
        name: patch.name,
        status: patch.status,
        current_ticket_id: patch.current_ticket_id ?? null,
        session_id: patch.session_id ?? '',
        runtime_phase: patch.runtime_phase ?? 'none',
        runtime_started_at: patch.runtime_started_at ?? null,
        last_error: patch.last_error ?? '',
        workspace_path: patch.workspace_path ?? '',
        capabilities: patch.capabilities ?? [],
        total_tokens_used: patch.total_tokens_used ?? 0,
        total_tickets_completed: patch.total_tickets_completed ?? 0,
        last_heartbeat_at: patch.last_heartbeat_at ?? null,
      },
    ].sort((left, right) => left.name.localeCompare(right.name))
  }

  const next = [...items]
  next[index] = { ...items[index], ...patch }
  return next
}
