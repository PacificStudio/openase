import { ApiError } from '$lib/api/client'
import type { SSEFrame, StreamConnectionState } from '$lib/api/sse'
import { toastStore } from '$lib/stores/toast.svelte'
import { fetchTicketDetailContext } from './context'
import {
  recoverTicketDrawerRunTranscript,
  defaultTicketDrawerRunTranscriptDeps,
  loadTicketDrawerRunTranscript,
  type TicketDrawerRunTranscriptDeps,
} from './drawer-run-transcript'
import {
  applyTicketRunStreamFrame,
  hydrateTicketRunDetail,
  selectTicketRun,
} from './run-transcript'
import { mapTicketRunDetail } from './run-transcript-data'
import {
  applyTicketDrawerContext,
  applyTicketDrawerRunTranscriptState,
  applyTicketDrawerTimelineRefresh,
  readTicketDrawerRunTranscriptState,
} from './drawer-state-mutators'
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

type LoadOptions = { background?: boolean; preserveMessages?: boolean }

type TicketDrawerStateDeps = {
  fetchContext: typeof fetchTicketDetailContext
} & TicketDrawerRunTranscriptDeps

const defaultDeps: TicketDrawerStateDeps = {
  fetchContext: fetchTicketDetailContext,
  ...defaultTicketDrawerRunTranscriptDeps,
}

export function createTicketDrawerState(deps: Partial<TicketDrawerStateDeps> = {}) {
  const resolvedDeps = { ...defaultDeps, ...deps }

  const state = $state({
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
    loadingRunId: null as string | null,
    runStreamState: 'idle' as StreamConnectionState,
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
  })
  let loadRequestId = 0
  let runDetailRequestId = 0
  let runRecoveryRequestId = 0
  let timelineRefreshQueued = false
  let timelineRefreshLoop: Promise<void> | null = null

  async function runTimelineRefresh(projectId: string, ticketId: string) {
    if (state.loading || !state.ticket) {
      return
    }
    if (timelineRefreshLoop) {
      await timelineRefreshLoop
      return
    }

    timelineRefreshLoop = (async () => {
      while (timelineRefreshQueued && !state.loading && state.ticket) {
        timelineRefreshQueued = false
        const requestId = loadRequestId
        try {
          const detailContext = await resolvedDeps.fetchContext(projectId, ticketId)
          if (requestId !== loadRequestId || !state.ticket) {
            continue
          }
          applyTicketDrawerTimelineRefresh(state, detailContext)
        } catch (caughtError) {
          if (requestId !== loadRequestId || !state.ticket) {
            continue
          }
          const message =
            caughtError instanceof ApiError
              ? caughtError.detail
              : 'Failed to refresh ticket timeline.'
          toastStore.error(message)
        }
      }
    })().finally(() => {
      timelineRefreshLoop = null
    })

    await timelineRefreshLoop
  }

  return Object.assign(state, {
    clearMutationMessages() {
      // no-op: toasts auto-dismiss
    },
    setMutationError(message: string) {
      toastStore.error(message)
    },
    setMutationNotice(message: string) {
      toastStore.success(message)
    },
    async load(projectId: string, ticketId: string, options: LoadOptions = {}) {
      const requestId = ++loadRequestId
      runDetailRequestId += 1
      state.loadingRunId = null
      if (!options.background) {
        state.loading = true
        state.error = ''
      }
      try {
        const detailContext = await resolvedDeps.fetchContext(projectId, ticketId)
        if (requestId !== loadRequestId) return

        applyTicketDrawerContext(state, detailContext)
        await loadTicketDrawerRunTranscript(
          resolvedDeps,
          {
            getState: () => readTicketDrawerRunTranscriptState(state),
            setState: (nextState) => applyTicketDrawerRunTranscriptState(state, nextState),
          },
          projectId,
          ticketId,
          requestId,
          (activeRequestID) => activeRequestID === loadRequestId,
        )
      } catch (caughtError) {
        if (requestId !== loadRequestId) return
        const message =
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load ticket detail.'
        if (options.background) {
          toastStore.error(message)
        } else {
          state.error = message
        }
      } finally {
        if (requestId === loadRequestId && !options.background) {
          state.loading = false
        }
      }
    },
    async refreshTimeline(projectId: string, ticketId: string) {
      if (state.loading || !state.ticket) {
        return
      }
      timelineRefreshQueued = true
      await runTimelineRefresh(projectId, ticketId)
      if (timelineRefreshQueued) {
        await runTimelineRefresh(projectId, ticketId)
      }
    },
    setRunStreamState(nextState: StreamConnectionState) {
      state.runStreamState = nextState
    },
    applyRunStreamFrame(frame: Pick<SSEFrame, 'event' | 'data'>) {
      const nextState = applyTicketRunStreamFrame(readTicketDrawerRunTranscriptState(state), frame)
      applyTicketDrawerRunTranscriptState(state, nextState)
    },
    async recoverRunTranscript(projectId: string, ticketId: string) {
      if (state.loading || !state.ticket) {
        return
      }

      const requestId = ++runRecoveryRequestId
      state.recoveringRunTranscript = true

      try {
        await recoverTicketDrawerRunTranscript(
          resolvedDeps,
          {
            getState: () => readTicketDrawerRunTranscriptState(state),
            setState: (nextState) => applyTicketDrawerRunTranscriptState(state, nextState),
          },
          projectId,
          ticketId,
          requestId,
          (activeRequestID) =>
            activeRequestID === runRecoveryRequestId &&
            !state.loading &&
            state.ticket?.id === ticketId,
        )
      } catch (caughtError) {
        if (requestId !== runRecoveryRequestId) {
          return
        }
        const message =
          caughtError instanceof ApiError
            ? caughtError.detail
            : 'Failed to recover ticket run transcript.'
        toastStore.error(message)
      } finally {
        if (requestId === runRecoveryRequestId) {
          state.recoveringRunTranscript = false
        }
      }
    },
    async selectRun(projectId: string, ticketId: string, runId: string) {
      const optimisticState = selectTicketRun(readTicketDrawerRunTranscriptState(state), runId)
      applyTicketDrawerRunTranscriptState(state, optimisticState)

      if (optimisticState.currentRun?.id !== runId) {
        return
      }

      const requestId = ++runDetailRequestId
      state.loadingRunId = runId

      try {
        const detail = mapTicketRunDetail(await resolvedDeps.fetchRun(projectId, ticketId, runId))
        if (requestId !== runDetailRequestId) {
          return
        }

        applyTicketDrawerRunTranscriptState(
          state,
          hydrateTicketRunDetail(readTicketDrawerRunTranscriptState(state), detail),
        )
      } catch (caughtError) {
        if (requestId !== runDetailRequestId) {
          return
        }
        const message =
          caughtError instanceof ApiError
            ? caughtError.detail
            : 'Failed to load ticket run transcript.'
        toastStore.error(message)
      } finally {
        if (requestId === runDetailRequestId) {
          state.loadingRunId = null
        }
      }
    },
    reset() {
      loadRequestId += 1
      runDetailRequestId += 1
      timelineRefreshQueued = false
      timelineRefreshLoop = null
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
    },
  })
}
