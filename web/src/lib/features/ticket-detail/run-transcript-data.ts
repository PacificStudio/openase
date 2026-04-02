import type {
  TicketRunDetailPayload,
  TicketRunListPayload,
  TicketRunRecord,
  TicketRunStepRecord,
  TicketRunTraceRecord,
} from '$lib/api/contracts'
import type { TicketRun, TicketRunDetail, TicketRunStepEntry, TicketRunTraceEntry } from './types'

export function mapTicketRuns(payload: TicketRunListPayload): TicketRun[] {
  return payload.runs.map(mapTicketRun)
}

export function mapTicketRunDetail(payload: TicketRunDetailPayload): TicketRunDetail {
  return {
    run: mapTicketRun(payload.run),
    traceEntries: payload.trace_entries.map(mapTicketRunTraceEntry),
    stepEntries: payload.step_entries.map(mapTicketRunStepEntry),
  }
}

function mapTicketRun(item: TicketRunRecord): TicketRun {
  return {
    id: item.id,
    attemptNumber: item.attempt_number,
    agentId: item.agent_id,
    agentName: item.agent_name,
    provider: item.provider,
    status: normalizeRunStatus(item.status),
    currentStepStatus: item.current_step_status ?? undefined,
    currentStepSummary: item.current_step_summary ?? undefined,
    createdAt: item.created_at,
    runtimeStartedAt: item.runtime_started_at ?? undefined,
    lastHeartbeatAt: item.last_heartbeat_at ?? undefined,
    completedAt: item.completed_at ?? undefined,
    lastError: item.last_error ?? undefined,
    completionSummary: item.completion_summary
      ? {
          status: item.completion_summary.status,
          markdown: item.completion_summary.markdown ?? undefined,
          json: item.completion_summary.json ?? undefined,
          generatedAt: item.completion_summary.generated_at ?? undefined,
          error: item.completion_summary.error ?? undefined,
        }
      : undefined,
  }
}

function mapTicketRunTraceEntry(item: TicketRunTraceRecord): TicketRunTraceEntry {
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

function mapTicketRunStepEntry(item: TicketRunStepRecord): TicketRunStepEntry {
  return {
    id: item.id,
    agentRunId: item.agent_run_id,
    stepStatus: item.step_status,
    summary: item.summary,
    sourceTraceEventId: item.source_trace_event_id ?? undefined,
    createdAt: item.created_at,
  }
}

function normalizeRunStatus(status: string): TicketRun['status'] {
  if (
    status === 'launching' ||
    status === 'ready' ||
    status === 'executing' ||
    status === 'stalled' ||
    status === 'failed' ||
    status === 'completed'
  ) {
    return status
  }

  return 'launching'
}
