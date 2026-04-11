import type { TicketRunDetailPayload } from '$lib/api/contracts'
import type { TicketRunTranscriptPage } from './types'
import {
  buildTicketRunStepCursor,
  buildTicketRunTraceCursor,
  compareTicketRunTranscriptCursors,
} from './run-transcript-cursor'
import { mapTicketRunStepEntry, mapTicketRunTraceEntry } from './run-transcript-entry-mappers'
import { mapProjectedTranscriptPage, mapTranscriptPageRecord } from './run-transcript-page-items'

export { mapTicketRunStepEntry, mapTicketRunTraceEntry } from './run-transcript-entry-mappers'

export function mapTicketRunTranscriptPage(
  payload: TicketRunDetailPayload,
): TicketRunTranscriptPage {
  if (payload.transcript_entries_page || payload.activities?.length) {
    return mapProjectedTranscriptPage(payload)
  }

  if (payload.transcript_page) {
    return mapTranscriptPageRecord(payload.transcript_page)
  }

  const items = [
    ...(payload.step_entries ?? []).map((item) => {
      const stepEntry = mapTicketRunStepEntry(item)
      return {
        kind: 'step' as const,
        cursor: buildTicketRunStepCursor(stepEntry),
        stepEntry,
      }
    }),
    ...(payload.trace_entries ?? []).map((item) => {
      const traceEntry = mapTicketRunTraceEntry(item)
      return {
        kind: 'trace' as const,
        cursor: buildTicketRunTraceCursor(traceEntry),
        traceEntry,
      }
    }),
  ].sort((left, right) => compareTicketRunTranscriptCursors(left.cursor, right.cursor))

  return {
    items,
    hasOlder: false,
    hiddenOlderCount: 0,
    hasNewer: false,
    hiddenNewerCount: 0,
    oldestCursor: items[0]?.cursor,
    newestCursor: items.at(-1)?.cursor,
  }
}
