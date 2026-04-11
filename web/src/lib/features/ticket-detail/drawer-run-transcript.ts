import {
  getTicketRun,
  listTicketRunActivities,
  listTicketRunRawEvents,
  listTicketRunTranscriptEntries,
  listTicketRuns,
} from '$lib/api/openase'
import { mapTicketRunDetail, mapTicketRuns } from './run-transcript-data'
import {
  createEmptyTicketRunTranscriptState,
  hydrateTicketRunDetail,
  setTicketRunList,
} from './run-transcript'
import type { TicketRunTranscriptState } from './types'

export type TicketDrawerRunTranscriptDeps = {
  fetchRuns: typeof listTicketRuns
  fetchRun: typeof getTicketRun
  fetchRunActivities: typeof listTicketRunActivities
  fetchRunRawEvents: typeof listTicketRunRawEvents
  fetchRunTranscriptEntries: typeof listTicketRunTranscriptEntries
}

export const defaultTicketDrawerRunTranscriptDeps: TicketDrawerRunTranscriptDeps = {
  fetchRuns: listTicketRuns,
  fetchRun: getTicketRun,
  fetchRunActivities: listTicketRunActivities,
  fetchRunRawEvents: listTicketRunRawEvents,
  fetchRunTranscriptEntries: listTicketRunTranscriptEntries,
}

type TicketDrawerRunTranscriptIO = {
  getState: () => TicketRunTranscriptState
  setState: (state: TicketRunTranscriptState) => void
}

export async function loadTicketDrawerRunTranscript(
  deps: TicketDrawerRunTranscriptDeps,
  io: TicketDrawerRunTranscriptIO,
  projectId: string,
  ticketId: string,
  requestId: number,
  isCurrentRequest: (requestId: number) => boolean,
) {
  const runList = mapTicketRuns(
    await (deps.fetchRuns === listTicketRuns
      ? listTicketRuns(projectId, ticketId)
      : deps.fetchRuns(projectId, ticketId)),
  )
  if (!isCurrentRequest(requestId)) {
    return
  }

  let nextState = setTicketRunList(createEmptyTicketRunTranscriptState(), runList)
  io.setState(nextState)

  if (!nextState.currentRun) {
    return
  }

  const runId = nextState.currentRun.id
  const [detailPayload, activitiesPayload, transcriptEntriesPayload, rawEventsPayload] =
    await Promise.all([
      deps.fetchRun === getTicketRun
        ? getTicketRun(projectId, ticketId, runId)
        : deps.fetchRun(projectId, ticketId, runId),
      deps.fetchRunActivities === listTicketRunActivities
        ? listTicketRunActivities(projectId, ticketId, runId)
        : deps.fetchRunActivities(projectId, ticketId, runId),
      deps.fetchRunTranscriptEntries === listTicketRunTranscriptEntries
        ? listTicketRunTranscriptEntries(projectId, ticketId, runId)
        : deps.fetchRunTranscriptEntries(projectId, ticketId, runId),
      deps.fetchRunRawEvents === listTicketRunRawEvents
        ? listTicketRunRawEvents(projectId, ticketId, runId, { limit: 50 })
        : deps.fetchRunRawEvents(projectId, ticketId, runId, { limit: 50 }),
    ])
  const detail = mapTicketRunDetail({
    ...detailPayload,
    activities: activitiesPayload.activities,
    transcript_entries_page: transcriptEntriesPayload.transcript_entries_page,
    raw_events_page: rawEventsPayload.raw_events_page,
  })
  if (!isCurrentRequest(requestId)) {
    return
  }

  nextState = hydrateTicketRunDetail(io.getState(), detail)
  io.setState(nextState)
}

export async function recoverTicketDrawerRunTranscript(
  deps: TicketDrawerRunTranscriptDeps,
  io: TicketDrawerRunTranscriptIO,
  projectId: string,
  ticketId: string,
  requestId: number,
  isCurrentRequest: (requestId: number) => boolean,
) {
  const runList = mapTicketRuns(
    await (deps.fetchRuns === listTicketRuns
      ? listTicketRuns(projectId, ticketId)
      : deps.fetchRuns(projectId, ticketId)),
  )
  if (!isCurrentRequest(requestId)) {
    return
  }

  let nextState = setTicketRunList(io.getState(), runList)
  io.setState(nextState)

  if (!nextState.currentRun) {
    return
  }
  const currentRunId = nextState.currentRun.id
  let afterCursor = nextState.pageInfoByRun[currentRunId]?.newestCursor
  let afterEventCursor = nextState.pageInfoByRun[currentRunId]?.newestEventCursor
  while (afterCursor) {
    const [detailPayload, transcriptEntriesPayload] = await Promise.all([
      deps.fetchRun === getTicketRun
        ? getTicketRun(projectId, ticketId, currentRunId, { after: afterCursor })
        : deps.fetchRun(projectId, ticketId, currentRunId, { after: afterCursor }),
      afterEventCursor
        ? deps.fetchRunTranscriptEntries === listTicketRunTranscriptEntries
          ? listTicketRunTranscriptEntries(projectId, ticketId, currentRunId, {
              after: afterEventCursor,
            })
          : deps.fetchRunTranscriptEntries(projectId, ticketId, currentRunId, {
              after: afterEventCursor,
            })
        : Promise.resolve(undefined),
    ])
    const backfillDetail = mapTicketRunDetail({
      ...detailPayload,
      transcript_entries_page:
        transcriptEntriesPayload?.transcript_entries_page ?? detailPayload.transcript_entries_page,
    })
    if (!isCurrentRequest(requestId)) {
      return
    }
    if (backfillDetail.transcriptPage.items.length === 0) {
      break
    }

    nextState = hydrateTicketRunDetail(io.getState(), backfillDetail, { select: false })
    io.setState(nextState)

    if (!backfillDetail.transcriptPage.hasNewer) {
      break
    }
    afterCursor = backfillDetail.transcriptPage.newestCursor
    afterEventCursor = backfillDetail.transcriptPage.newestEventCursor
  }
}

export async function loadOlderTicketDrawerRunTranscript(
  deps: TicketDrawerRunTranscriptDeps,
  io: TicketDrawerRunTranscriptIO,
  projectId: string,
  ticketId: string,
  runId: string,
  oldestCursor: string,
  oldestEventCursor?: string,
) {
  const [detailPayload, transcriptEntriesPayload] = await Promise.all([
    deps.fetchRun === getTicketRun
      ? getTicketRun(projectId, ticketId, runId, { before: oldestCursor })
      : deps.fetchRun(projectId, ticketId, runId, { before: oldestCursor }),
    oldestEventCursor
      ? deps.fetchRunTranscriptEntries === listTicketRunTranscriptEntries
        ? listTicketRunTranscriptEntries(projectId, ticketId, runId, { before: oldestEventCursor })
        : deps.fetchRunTranscriptEntries(projectId, ticketId, runId, { before: oldestEventCursor })
      : Promise.resolve(undefined),
  ])
  const detail = mapTicketRunDetail({
    ...detailPayload,
    transcript_entries_page:
      transcriptEntriesPayload?.transcript_entries_page ?? detailPayload.transcript_entries_page,
  })
  if (detail.transcriptPage.items.length === 0) {
    return
  }

  io.setState(hydrateTicketRunDetail(io.getState(), detail, { select: false }))
}
