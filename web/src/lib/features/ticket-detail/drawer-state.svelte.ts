import { ApiError } from '$lib/api/client'
import type { SSEFrame, StreamConnectionState } from '$lib/api/sse'
import { toastStore } from '$lib/stores/toast.svelte'
import { fetchTicketDetailLiveContext, fetchTicketDetailProjectReferenceData } from './context'
import {
  recoverTicketDrawerRunTranscript,
  defaultTicketDrawerRunTranscriptDeps,
  type TicketDrawerRunTranscriptDeps,
} from './drawer-run-transcript'
import { ensureTicketDrawerRunsLoaded } from './drawer-state-run-loading'
import {
  createEmptyTicketRunTranscriptState,
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
  resetTicketDrawerState,
} from './drawer-state-mutators'
import { createTicketDrawerReferenceController } from './drawer-state-reference'
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
    runsLoaded: false,
    loadingRuns: false,
    runsError: '',
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
    resettingWorkspace: false,
    archiving: false,
  })
  let loadRequestId = 0
  let runDetailRequestId = 0
  let runRecoveryRequestId = 0
  let runTranscriptRequestId = 0
  const referenceController = createTicketDrawerReferenceController({
    fetchLiveContext: resolvedDeps.fetchLiveContext,
    fetchReferenceData: resolvedDeps.fetchReferenceData,
    getLoadRequestId: () => loadRequestId,
    isLoading: () => state.loading,
    getTicket: () => state.ticket,
    applyReferenceSelection: (selected) => {
      state.statuses = selected.statuses
      state.dependencyCandidates = selected.dependencyCandidates
      state.repoOptions = selected.repoOptions
    },
    applyTimelineRefresh: (detailContext) => applyTicketDrawerTimelineRefresh(state, detailContext),
    notifyError: (message) => toastStore.error(message),
  })

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
      runTranscriptRequestId += 1
      state.loadingRunId = null
      if (!options.background || state.ticket?.id !== ticketId) {
        state.loadingRuns = false
        state.runsLoaded = false
        state.runsError = ''
        applyTicketDrawerRunTranscriptState(state, createEmptyTicketRunTranscriptState())
      }
      if (!options.background) {
        state.loading = true
        state.error = ''
      }
      try {
        const referencePromise = referenceController.ensureReferenceData(projectId)
        const referenceSnapshot = referenceController.hasCachedReferenceData(projectId)
          ? await referencePromise
          : null
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

        referenceController.applyReferenceData(projectId, ticketId, nextReferenceData!)
        applyTicketDrawerContext(state, detailContext)
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
      await referenceController.refreshTimeline(projectId, ticketId)
    },
    async refreshReferences(projectId: string, ticketId: string) {
      await referenceController.refreshReferences(projectId, ticketId)
    },
    setRunStreamState(nextState: StreamConnectionState) {
      state.runStreamState = nextState
    },
    async ensureRunsLoaded(projectId: string, ticketId: string, options: { force?: boolean } = {}) {
      await ensureTicketDrawerRunsLoaded(
        resolvedDeps,
        state,
        projectId,
        ticketId,
        ++runTranscriptRequestId,
        {
          get current() {
            return runTranscriptRequestId
          },
          set current(value: number) {
            runTranscriptRequestId = value
          },
        },
        options,
      )
    },
    applyRunStreamFrame(frame: Pick<SSEFrame, 'event' | 'data'>) {
      const nextState = applyTicketRunStreamFrame(readTicketDrawerRunTranscriptState(state), frame)
      applyTicketDrawerRunTranscriptState(state, nextState)
    },
    async recoverRunTranscript(projectId: string, ticketId: string) {
      if (state.loading || !state.ticket || !state.runsLoaded) {
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
      runTranscriptRequestId += 1
      referenceController.resetQueues()
      resetTicketDrawerState(state)
    },
    invalidateReferences(projectId?: string) {
      referenceController.invalidateReferences(projectId)
    },
  })
}

export type TicketDrawerState = ReturnType<typeof createTicketDrawerState>
