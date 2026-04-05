import type { StreamConnectionState } from '$lib/api/sse'
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

export type TicketDrawerMutableState = {
  loading: boolean
  error: string
  ticket: TicketDetail | null
  timeline: TicketTimelineItem[]
  hooks: HookExecution[]
  statuses: TicketStatusOption[]
  dependencyCandidates: TicketReferenceOption[]
  repoOptions: TicketRepoOption[]
  runs: TicketRun[]
  runsLoaded: boolean
  loadingRuns: boolean
  runsError: string
  selectedRunId: string | null
  followLatest: boolean
  currentRun: TicketRun | null
  runBlocks: TicketRunTranscriptBlock[]
  runBlockCache: Record<string, TicketRunTranscriptBlock[]>
  loadingRunId: string | null
  runStreamState: StreamConnectionState
  recoveringRunTranscript: boolean
  savingFields: boolean
  creatingDependency: boolean
  deletingDependencyId: string | null
  creatingExternalLink: boolean
  deletingExternalLinkId: string | null
  creatingRepoScope: boolean
  updatingRepoScopeId: string | null
  deletingRepoScopeId: string | null
  creatingComment: boolean
  updatingCommentId: string | null
  deletingCommentId: string | null
  resumingRetry: boolean
  resettingWorkspace: boolean
  archiving: boolean
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

export function resetTicketDrawerState(state: TicketDrawerMutableState) {
  state.loading = false
  state.error = ''
  state.ticket = null
  state.timeline = []
  state.hooks = []
  state.statuses = []
  state.dependencyCandidates = []
  state.repoOptions = []
  state.runsLoaded = false
  state.loadingRuns = false
  state.runsError = ''
  applyTicketDrawerRunTranscriptState(state, createEmptyTicketRunTranscriptState())
  state.loadingRunId = null
  state.runStreamState = 'idle'
  state.recoveringRunTranscript = false
  state.savingFields = false
  state.creatingDependency = false
  state.deletingDependencyId = null
  state.creatingExternalLink = false
  state.deletingExternalLinkId = null
  state.creatingRepoScope = false
  state.updatingRepoScopeId = null
  state.deletingRepoScopeId = null
  state.creatingComment = false
  state.updatingCommentId = null
  state.deletingCommentId = null
  state.resumingRetry = false
  state.resettingWorkspace = false
  state.archiving = false
}
