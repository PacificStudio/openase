import { seedRunBlocks } from './run-transcript-blocks'
import type { TicketRunTranscriptBlock, TicketRunTranscriptState } from './types'

export function syncSelectedRun(state: TicketRunTranscriptState): TicketRunTranscriptState {
  const latestRun = state.runs[0] ?? null
  if (!latestRun) {
    return {
      ...state,
      selectedRunId: null,
      followLatest: true,
      currentRun: null,
      blocks: [],
    }
  }

  let followLatest = state.followLatest
  let selectedRunId = state.selectedRunId
  if (!selectedRunId) {
    selectedRunId = latestRun.id
    followLatest = true
  }
  if (followLatest) {
    selectedRunId = latestRun.id
  }

  const selectedRun = state.runs.find((run) => run.id === selectedRunId) ?? latestRun
  return {
    ...state,
    selectedRunId,
    followLatest,
    currentRun: selectedRun,
    blocks: state.blockCache[selectedRun.id] ?? seedRunBlocks(selectedRun),
  }
}

export function syncSelectedBlocks(
  state: TicketRunTranscriptState,
): Record<string, TicketRunTranscriptBlock[]> {
  if (!state.currentRun || state.blocks.length === 0) {
    return state.blockCache
  }
  return {
    ...state.blockCache,
    [state.currentRun.id]: state.blocks,
  }
}

export function cacheSelectedState(state: TicketRunTranscriptState): TicketRunTranscriptState {
  return {
    ...state,
    blockCache: syncSelectedBlocks(state),
  }
}
