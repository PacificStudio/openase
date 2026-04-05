import { getTicketRun, listTicketRuns } from '$lib/api/openase'
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
}

export const defaultTicketDrawerRunTranscriptDeps: TicketDrawerRunTranscriptDeps = {
  fetchRuns: listTicketRuns,
  fetchRun: getTicketRun,
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

  const detail = mapTicketRunDetail(
    await (deps.fetchRun === getTicketRun
      ? getTicketRun(projectId, ticketId, nextState.currentRun.id)
      : deps.fetchRun(projectId, ticketId, nextState.currentRun.id)),
  )
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
  while (afterCursor) {
    const backfillDetail = mapTicketRunDetail(
      await (deps.fetchRun === getTicketRun
        ? getTicketRun(projectId, ticketId, currentRunId, { after: afterCursor })
        : deps.fetchRun(projectId, ticketId, currentRunId, { after: afterCursor })),
    )
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
  }
}

export async function loadOlderTicketDrawerRunTranscript(
  deps: TicketDrawerRunTranscriptDeps,
  io: TicketDrawerRunTranscriptIO,
  projectId: string,
  ticketId: string,
  runId: string,
  oldestCursor: string,
) {
  const detail = mapTicketRunDetail(
    await (deps.fetchRun === getTicketRun
      ? getTicketRun(projectId, ticketId, runId, { before: oldestCursor })
      : deps.fetchRun(projectId, ticketId, runId, { before: oldestCursor })),
  )
  if (detail.transcriptPage.items.length === 0) {
    return
  }

  io.setState(hydrateTicketRunDetail(io.getState(), detail, { select: false }))
}
