import type { TicketRunStepRecord, TicketRunTraceRecord } from '$lib/api/contracts'
import type { TicketRunStepEntry, TicketRunTraceEntry } from './types'

export function mapTicketRunTraceEntry(item: TicketRunTraceRecord): TicketRunTraceEntry {
  return {
    id: item.id,
    agentRunId: item.agent_run_id,
    sequence: item.sequence,
    provider: item.provider,
    kind: item.kind,
    stream: item.stream,
    output: item.output,
    payload: item.payload ?? {},
    createdAt: item.created_at,
  }
}

export function mapTicketRunStepEntry(item: TicketRunStepRecord): TicketRunStepEntry {
  return {
    id: item.id,
    agentRunId: item.agent_run_id,
    stepStatus: item.step_status,
    summary: item.summary,
    sourceTraceEventId: item.source_trace_event_id ?? undefined,
    createdAt: item.created_at,
  }
}
