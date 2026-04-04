import type { TicketDetailLiveContext } from './context'
import { createEmptyTicketRunTranscriptState } from './run-transcript'
import type {
  HookExecution,
  TicketDetail,
  TicketReferenceOption,
  TicketRepoOption,
  TicketRun,
  TicketRunTranscriptBlock,
  TicketStatusOption,
  TicketTimelineItem,
} from './types'

type TicketDrawerRunTranscriptState = ReturnType<typeof createEmptyTicketRunTranscriptState>

type TicketDrawerMutableState = {
  ticket: TicketDetail | null
  timeline: TicketTimelineItem[]
  hooks: HookExecution[]
  statuses: TicketStatusOption[]
  dependencyCandidates: TicketReferenceOption[]
  repoOptions: TicketRepoOption[]
  runs: TicketRun[]
  selectedRunId: string | null
  followLatest: boolean
  currentRun: TicketRun | null
  runBlocks: TicketRunTranscriptBlock[]
  runBlockCache: Record<string, TicketRunTranscriptBlock[]>
}

export function applyTicketDrawerContext(
  state: TicketDrawerMutableState,
  detailContext: TicketDetailLiveContext,
) {
  state.ticket = detailContext.ticket
  state.timeline = detailContext.timeline
  state.hooks = detailContext.hooks
}

export function applyTicketDrawerTimelineRefresh(
  state: TicketDrawerMutableState,
  detailContext: TicketDetailLiveContext,
) {
  state.ticket = detailContext.ticket
  state.timeline = detailContext.timeline
  state.hooks = detailContext.hooks
}

export function applyTicketDrawerRunTranscriptState(
  state: TicketDrawerMutableState,
  nextState: TicketDrawerRunTranscriptState,
) {
  state.runs = nextState.runs
  state.selectedRunId = nextState.selectedRunId
  state.followLatest = nextState.followLatest
  state.currentRun = nextState.currentRun
  state.runBlocks = nextState.blocks
  state.runBlockCache = nextState.blockCache
}

export function readTicketDrawerRunTranscriptState(
  state: TicketDrawerMutableState,
): TicketDrawerRunTranscriptState {
  return {
    runs: state.runs,
    selectedRunId: state.selectedRunId,
    followLatest: state.followLatest,
    currentRun: state.currentRun,
    blocks: state.runBlocks,
    blockCache: state.runBlockCache,
  }
}
