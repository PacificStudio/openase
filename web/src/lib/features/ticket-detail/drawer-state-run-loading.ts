import { ApiError } from '$lib/api/client'
import {
  loadTicketDrawerRunTranscript,
  type TicketDrawerRunTranscriptDeps,
} from './drawer-run-transcript'
import {
  applyTicketDrawerRunTranscriptState,
  readTicketDrawerRunTranscriptState,
  type TicketDrawerMutableState,
} from './drawer-state-mutators'
import { createEmptyTicketRunTranscriptState } from './run-transcript'

export async function ensureTicketDrawerRunsLoaded(
  deps: TicketDrawerRunTranscriptDeps,
  state: TicketDrawerMutableState,
  projectId: string,
  ticketId: string,
  requestId: number,
  activeRequestIdRef: { current: number },
  options: { force?: boolean } = {},
) {
  if (state.loading || !state.ticket || state.ticket.id !== ticketId || state.loadingRuns) {
    return
  }
  if (state.runsLoaded && !options.force) {
    return
  }

  activeRequestIdRef.current = requestId
  state.loadingRuns = true
  state.runsError = ''

  if (options.force) {
    applyTicketDrawerRunTranscriptState(state, createEmptyTicketRunTranscriptState())
  }

  try {
    await loadTicketDrawerRunTranscript(
      deps,
      {
        getState: () => readTicketDrawerRunTranscriptState(state),
        setState: (nextState) => applyTicketDrawerRunTranscriptState(state, nextState),
      },
      projectId,
      ticketId,
      requestId,
      (activeRequestID) =>
        activeRequestID === activeRequestIdRef.current &&
        !state.loading &&
        state.ticket?.id === ticketId,
    )
    if (requestId === activeRequestIdRef.current) {
      state.runsLoaded = true
    }
  } catch (caughtError) {
    if (requestId !== activeRequestIdRef.current) {
      return
    }
    state.runsLoaded = false
    state.runsError =
      caughtError instanceof ApiError ? caughtError.detail : 'Failed to load ticket runs.'
  } finally {
    if (requestId === activeRequestIdRef.current) {
      state.loadingRuns = false
    }
  }
}
