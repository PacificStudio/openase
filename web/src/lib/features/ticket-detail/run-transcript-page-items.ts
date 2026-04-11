import type {
  TicketRunActivityRecord,
  TicketRunDetailPayload,
  TicketRunTranscriptEntryPageRecord,
  TicketRunTranscriptEntryRecord,
  TicketRunTranscriptItemRecord,
  TicketRunTranscriptPageRecord,
} from '$lib/api/contracts'
import type { TicketRunTranscriptItem, TicketRunTranscriptPage } from './types'
import {
  buildTicketRunStepCursor,
  buildTicketRunTraceCursor,
  compareTicketRunTranscriptCursors,
  normalizeTicketRunEventCursor,
  normalizeTicketRunTranscriptCursor,
} from './run-transcript-cursor'
import { mapTicketRunStepEntry, mapTicketRunTraceEntry } from './run-transcript-entry-mappers'

export function mapProjectedTranscriptPage(
  payload: TicketRunDetailPayload,
): TicketRunTranscriptPage {
  const transcriptCursorPage = payload.transcript_page
    ? mapTranscriptPageRecord(payload.transcript_page)
    : undefined
  const transcriptItems = mapTranscriptEntriesToItems(
    payload.run.id,
    payload.transcript_entries_page,
  )
  const liveActivityItems = (payload.activities ?? [])
    .map((activity) => mapActivityToTranscriptItem(payload.run.id, activity))
    .filter((item): item is TicketRunTranscriptItem => item !== null)

  const items = [...transcriptItems, ...liveActivityItems].sort((left, right) =>
    compareTicketRunTranscriptCursors(left.cursor, right.cursor),
  )

  return {
    items,
    hasOlder: payload.transcript_entries_page?.has_older ?? false,
    hiddenOlderCount: payload.transcript_entries_page?.hidden_older_count ?? 0,
    hasNewer: payload.transcript_entries_page?.has_newer ?? false,
    hiddenNewerCount: payload.transcript_entries_page?.hidden_newer_count ?? 0,
    oldestCursor: transcriptCursorPage?.oldestCursor,
    newestCursor: transcriptCursorPage?.newestCursor,
    oldestEventCursor: normalizeTicketRunEventCursor(
      payload.transcript_entries_page?.oldest_cursor,
    ),
    newestEventCursor: normalizeTicketRunEventCursor(
      payload.transcript_entries_page?.newest_cursor,
    ),
  }
}

export function mapTranscriptPageRecord(
  record: TicketRunTranscriptPageRecord,
): TicketRunTranscriptPage {
  const items = (record.items ?? [])
    .map(mapTicketRunTranscriptItem)
    .sort((left, right) => compareTicketRunTranscriptCursors(left.cursor, right.cursor))

  return {
    items,
    hasOlder: record.has_older,
    hiddenOlderCount: record.hidden_older_count,
    hasNewer: record.has_newer,
    hiddenNewerCount: record.hidden_newer_count,
    oldestCursor: normalizeTicketRunTranscriptCursor(record.oldest_cursor) ?? items[0]?.cursor,
    newestCursor: normalizeTicketRunTranscriptCursor(record.newest_cursor) ?? items.at(-1)?.cursor,
    oldestEventCursor: undefined,
    newestEventCursor: undefined,
  }
}

function mapTranscriptEntriesToItems(
  runId: string,
  page: TicketRunTranscriptEntryPageRecord | undefined,
): TicketRunTranscriptItem[] {
  return (page?.entries ?? [])
    .map((entry) => mapTranscriptEntryToItem(runId, entry))
    .filter((item): item is TicketRunTranscriptItem => item !== null)
}

function mapTranscriptEntryToItem(
  runId: string,
  entry: TicketRunTranscriptEntryRecord,
): TicketRunTranscriptItem | null {
  const summary = entry.summary ?? entry.title ?? entry.command ?? entry.tool_name ?? ''
  const body = entry.body_text ?? entry.summary ?? ''

  switch (entry.entry_kind) {
    case 'assistant_message': {
      return mapTraceTranscriptItem(runId, entry, 'assistant_snapshot', 'assistant', body, {
        item_id: entry.activity_id ?? entry.id,
        source: 'transcript_entry',
        entry_kind: entry.entry_kind,
      })
    }
    case 'command_started': {
      const stepEntry = mapTicketRunStepEntry({
        id: `transcript:${entry.id}`,
        agent_run_id: runId,
        source_trace_event_id: null,
        step_status: 'running_command',
        summary: entry.command ?? summary,
        created_at: entry.created_at,
      })
      return {
        kind: 'step',
        cursor: buildTicketRunStepCursor(stepEntry),
        stepEntry,
      }
    }
    case 'command_completed':
    case 'command_failed': {
      return mapTraceTranscriptItem(runId, entry, 'command_output_snapshot', 'command', body, {
        item_id: entry.activity_id ?? entry.id,
        command: entry.command,
        source: 'transcript_entry',
        entry_kind: entry.entry_kind,
      })
    }
    case 'tool_call_started': {
      return mapTraceTranscriptItem(runId, entry, 'tool_call_started', 'tool', summary, {
        tool: entry.tool_name ?? summary,
        source: 'transcript_entry',
        entry_kind: entry.entry_kind,
      })
    }
    case 'tool_call_finished': {
      return mapTraceTranscriptItem(runId, entry, 'task_progress', 'task', body || summary, {
        summary,
        tool: entry.tool_name,
        source: 'transcript_entry',
        entry_kind: entry.entry_kind,
      })
    }
    case 'approval_requested': {
      return mapTraceTranscriptItem(
        runId,
        entry,
        'approval_requested',
        'task',
        summary,
        entry.metadata ?? {},
      )
    }
    case 'turn_diff': {
      return mapTraceTranscriptItem(runId, entry, 'turn_diff_updated', 'diff', body, {
        diff: body,
        source: 'transcript_entry',
      })
    }
    case 'error': {
      return mapTraceTranscriptItem(
        runId,
        entry,
        'error',
        'task',
        body || summary,
        entry.metadata ?? {},
      )
    }
    default:
      return null
  }
}

function mapTraceTranscriptItem(
  runId: string,
  entry: TicketRunTranscriptEntryRecord,
  kind: string,
  stream: string,
  output: string,
  payload: Record<string, unknown>,
): TicketRunTranscriptItem {
  const traceEntry = mapTicketRunTraceEntry({
    id: `transcript:${entry.id}`,
    agent_run_id: runId,
    sequence: 0,
    provider: entry.provider,
    kind,
    stream,
    output,
    payload,
    created_at: entry.created_at,
  })
  return {
    kind: 'trace',
    cursor: buildTicketRunTraceCursor(traceEntry),
    traceEntry,
  }
}

function mapActivityToTranscriptItem(
  runId: string,
  activity: TicketRunActivityRecord,
): TicketRunTranscriptItem | null {
  const liveText = activity.live_text?.trim()
  if (!liveText) {
    return null
  }
  if (activity.status === 'completed' || activity.status === 'failed') {
    return null
  }

  if (activity.activity_kind === 'assistant_message') {
    return mapActivityTraceItem(runId, activity, 'assistant_snapshot', 'assistant', liveText, {
      item_id: activity.activity_id,
      source: 'activity_state',
    })
  }

  if (activity.activity_kind === 'command_execution') {
    return mapActivityTraceItem(runId, activity, 'command_output_snapshot', 'command', liveText, {
      item_id: activity.activity_id,
      command: activity.command,
      source: 'activity_state',
    })
  }

  if (activity.activity_kind === 'approval') {
    return mapActivityTraceItem(
      runId,
      activity,
      'approval_requested',
      'task',
      activity.title ?? liveText,
      activity.metadata ?? {},
    )
  }

  const stepEntry = mapTicketRunStepEntry({
    id: `activity:${activity.id}`,
    agent_run_id: runId,
    source_trace_event_id: null,
    step_status: activity.status,
    summary: activity.title ?? liveText,
    created_at: activity.updated_at,
  })
  return {
    kind: 'step',
    cursor: buildTicketRunStepCursor(stepEntry),
    stepEntry,
  }
}

function mapActivityTraceItem(
  runId: string,
  activity: TicketRunActivityRecord,
  kind: string,
  stream: string,
  output: string,
  payload: Record<string, unknown>,
): TicketRunTranscriptItem {
  const traceEntry = mapTicketRunTraceEntry({
    id: `activity:${activity.id}`,
    agent_run_id: runId,
    sequence: 0,
    provider: activity.provider,
    kind,
    stream,
    output,
    payload,
    created_at: activity.updated_at,
  })
  return {
    kind: 'trace',
    cursor: buildTicketRunTraceCursor(traceEntry),
    traceEntry,
  }
}

function mapTicketRunTranscriptItem(item: TicketRunTranscriptItemRecord): TicketRunTranscriptItem {
  if (item.kind === 'step' && item.step_entry) {
    const stepEntry = mapTicketRunStepEntry(item.step_entry)
    return {
      kind: 'step',
      cursor:
        normalizeTicketRunTranscriptCursor(item.cursor) ?? buildTicketRunStepCursor(stepEntry),
      stepEntry,
    }
  }

  const traceEntry = mapTicketRunTraceEntry(
    item.trace_entry ?? {
      id: '',
      agent_run_id: '',
      sequence: 0,
      provider: '',
      kind: '',
      stream: '',
      output: '',
      payload: {},
      created_at: '',
    },
  )
  return {
    kind: 'trace',
    cursor:
      normalizeTicketRunTranscriptCursor(item.cursor) ?? buildTicketRunTraceCursor(traceEntry),
    traceEntry,
  }
}
