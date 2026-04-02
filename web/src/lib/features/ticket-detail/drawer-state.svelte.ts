import { ApiError } from '$lib/api/client'
import type { SSEFrame, StreamConnectionState } from '$lib/api/sse'
import { toastStore } from '$lib/stores/toast.svelte'
import {
  fetchTicketDetailLiveContext,
  fetchTicketDetailProjectReferenceData,
  selectTicketDetailReferenceData,
  type TicketDetailProjectReferenceData,
} from './context'
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
  fetchLiveContext: typeof fetchTicketDetailLiveContext
  fetchReferenceData: typeof fetchTicketDetailProjectReferenceData
} & TicketDrawerRunTranscriptDeps

const defaultDeps: TicketDrawerStateDeps = {
  fetchLiveContext: fetchTicketDetailLiveContext,
  fetchReferenceData: fetchTicketDetailProjectReferenceData,
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
  let referenceData: TicketDetailProjectReferenceData | null = null
  let referenceProjectId: string | null = null
  let referenceRefreshQueued = false
  let referenceRefreshLoop: Promise<void> | null = null

  const hasCachedReferenceData = (projectId: string) =>
    referenceProjectId === projectId && referenceData !== null

  async function ensureReferenceData(projectId: string) {
    if (hasCachedReferenceData(projectId)) {
      return referenceData
    }

    const nextReferenceData = await resolvedDeps.fetchReferenceData(projectId)
    referenceProjectId = projectId
    referenceData = nextReferenceData
    return nextReferenceData
  }

  function applyReferenceData(
    projectId: string,
    ticketId: string,
    nextReferenceData: TicketDetailProjectReferenceData,
  ) {
    referenceProjectId = projectId
    referenceData = nextReferenceData
    const selectedReferenceData = selectTicketDetailReferenceData(nextReferenceData, ticketId)
    state.statuses = selectedReferenceData.statuses
    state.dependencyCandidates = selectedReferenceData.dependencyCandidates
    state.repoOptions = selectedReferenceData.repoOptions
  }

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
          const currentReferenceData = await ensureReferenceData(projectId)
          if (requestId !== loadRequestId || !state.ticket) {
            continue
          }
          const detailContext = await resolvedDeps.fetchLiveContext(
            projectId,
            ticketId,
            currentReferenceData!,
          )
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

  async function runReferenceRefresh(projectId: string, ticketId: string) {
    if (state.loading) {
      return
    }
    if (referenceRefreshLoop) {
      await referenceRefreshLoop
      return
    }

    referenceRefreshLoop = (async () => {
      while (referenceRefreshQueued && !state.loading) {
        referenceRefreshQueued = false
        const requestId = loadRequestId
        try {
          const nextReferenceData = await resolvedDeps.fetchReferenceData(projectId)
          if (requestId !== loadRequestId || activeTicketChanged(ticketId)) {
            continue
          }
          applyReferenceData(projectId, ticketId, nextReferenceData)
        } catch (caughtError) {
          if (requestId !== loadRequestId || activeTicketChanged(ticketId)) {
            continue
          }
          const message =
            caughtError instanceof ApiError
              ? caughtError.detail
              : 'Failed to refresh ticket references.'
          toastStore.error(message)
        }
      }
    })().finally(() => {
      referenceRefreshLoop = null
    })

    await referenceRefreshLoop
  }

  const activeTicketChanged = (ticketId: string) => state.ticket?.id !== ticketId

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
        const referencePromise = ensureReferenceData(projectId)
        const referenceSnapshot = hasCachedReferenceData(projectId) ? referenceData : null
        const detailContextPromise = referenceSnapshot
          ? resolvedDeps.fetchLiveContext(projectId, ticketId, referenceSnapshot)
          : referencePromise.then((currentReferenceData) =>
              resolvedDeps.fetchLiveContext(projectId, ticketId, currentReferenceData!),
            )
        const [nextReferenceData, detailContext] = await Promise.all([
          referencePromise,
          detailContextPromise,
        ])
        if (requestId !== loadRequestId) return

        applyReferenceData(projectId, ticketId, nextReferenceData!)
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
    async refreshReferences(projectId: string, ticketId: string) {
      if (state.loading) {
        return
      }
      referenceRefreshQueued = true
      await runReferenceRefresh(projectId, ticketId)
      if (referenceRefreshQueued) {
        await runReferenceRefresh(projectId, ticketId)
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
      referenceRefreshQueued = false
      referenceRefreshLoop = null
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
    invalidateReferences(projectId?: string) {
      if (!projectId || referenceProjectId === projectId) {
        referenceProjectId = null
        referenceData = null
      }
    },
  })
}
