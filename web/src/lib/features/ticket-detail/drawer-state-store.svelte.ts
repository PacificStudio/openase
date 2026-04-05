import type { StreamConnectionState } from '$lib/api/sse'
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

type TicketDrawerMutableState = {
  loading: boolean
  error: string
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
  runStepEntriesByRun: ReturnType<typeof createEmptyTicketRunTranscriptState>['stepEntriesByRun']
  runTraceEntriesByRun: ReturnType<typeof createEmptyTicketRunTranscriptState>['traceEntriesByRun']
  runLifecycleBlocksByRun: ReturnType<
    typeof createEmptyTicketRunTranscriptState
  >['lifecycleBlocksByRun']
  runPageInfoByRun: ReturnType<typeof createEmptyTicketRunTranscriptState>['pageInfoByRun']
  loadingRunId: string | null
  loadingOlderRunId: string | null
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

export function createTicketDrawerMutableState(): TicketDrawerMutableState {
  return {
    loading: false,
    error: '',
    ticket: null as TicketDetail | null,
    timeline: [] as TicketTimelineItem[],
    hooks: [] as HookExecution[],
    statuses: [] as TicketStatusOption[],
    dependencyCandidates: [] as TicketReferenceOption[],
    repoOptions: [] as TicketRepoOption[],
    runs: [] as TicketRun[],
    selectedRunId: null as string | null,
    followLatest: true,
    currentRun: null as TicketRun | null,
    runBlocks: [] as TicketRunTranscriptBlock[],
    runBlockCache: {} as Record<string, TicketRunTranscriptBlock[]>,
    runStepEntriesByRun: {} as ReturnType<
      typeof createEmptyTicketRunTranscriptState
    >['stepEntriesByRun'],
    runTraceEntriesByRun: {} as ReturnType<
      typeof createEmptyTicketRunTranscriptState
    >['traceEntriesByRun'],
    runLifecycleBlocksByRun: {} as ReturnType<
      typeof createEmptyTicketRunTranscriptState
    >['lifecycleBlocksByRun'],
    runPageInfoByRun: {} as ReturnType<typeof createEmptyTicketRunTranscriptState>['pageInfoByRun'],
    loadingRunId: null as string | null,
    loadingOlderRunId: null as string | null,
    runStreamState: 'idle' as TicketDrawerMutableState['runStreamState'],
    recoveringRunTranscript: false,
    savingFields: false,
    creatingDependency: false,
    deletingDependencyId: null as string | null,
    creatingExternalLink: false,
    deletingExternalLinkId: null as string | null,
    creatingRepoScope: false,
    updatingRepoScopeId: null as string | null,
    deletingRepoScopeId: null as string | null,
    creatingComment: false,
    updatingCommentId: null as string | null,
    deletingCommentId: null as string | null,
    resumingRetry: false,
    resettingWorkspace: false,
    archiving: false,
  }
}

export function resetTicketDrawerMutableState(state: TicketDrawerMutableState): void {
  state.loading = false
  state.error = ''
  state.ticket = null
  state.timeline = []
  state.hooks = []
  state.statuses = []
  state.dependencyCandidates = []
  state.repoOptions = []
  state.runs = []
  state.selectedRunId = null
  state.followLatest = true
  state.currentRun = null
  state.runBlocks = []
  state.runBlockCache = {}
  state.runStepEntriesByRun = {}
  state.runTraceEntriesByRun = {}
  state.runLifecycleBlocksByRun = {}
  state.runPageInfoByRun = {}
  state.loadingRunId = null
  state.loadingOlderRunId = null
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
