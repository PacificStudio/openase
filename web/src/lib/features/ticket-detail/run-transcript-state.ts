import type { TicketRun, TicketRunTranscriptState } from './types'
import { sortTicketRuns } from './run-transcript-blocks'
import { syncSelectedBlocks, syncSelectedRun } from './run-transcript-selection'

export function createEmptyTicketRunTranscriptState(): TicketRunTranscriptState {
  return {
    runs: [],
    selectedRunId: null,
    followLatest: true,
    currentRun: null,
    blocks: [],
    blockCache: {},
  }
}

export function setTicketRunList(
  state: TicketRunTranscriptState,
  runs: TicketRun[],
): TicketRunTranscriptState {
  const nextRuns = sortTicketRuns(runs)
  const blockCache = syncSelectedBlocks(state)

  return syncSelectedRun({
    ...state,
    runs: nextRuns,
    blockCache,
  })
}

export function selectTicketRun(
  state: TicketRunTranscriptState,
  runId: string,
): TicketRunTranscriptState {
  if (!state.runs.some((run) => run.id === runId)) {
    return state
  }

  return syncSelectedRun({
    ...state,
    selectedRunId: runId,
    followLatest: state.runs[0]?.id === runId,
    blockCache: syncSelectedBlocks(state),
  })
}
