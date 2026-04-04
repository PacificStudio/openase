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

  const detail = mapTicketRunDetail(
    await (deps.fetchRun === getTicketRun
      ? getTicketRun(projectId, ticketId, nextState.currentRun.id)
      : deps.fetchRun(projectId, ticketId, nextState.currentRun.id)),
  )
  if (!isCurrentRequest(requestId)) {
    return
  }

  nextState = hydrateTicketRunDetail(io.getState(), detail, { select: false })
  io.setState(nextState)
}
