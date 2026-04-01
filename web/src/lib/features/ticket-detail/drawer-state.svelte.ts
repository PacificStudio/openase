import { ApiError } from '$lib/api/client'
import type { SSEFrame } from '$lib/api/sse'
import { toastStore } from '$lib/stores/toast.svelte'
import { fetchTicketDetailContext } from './context'
import {
  defaultTicketDrawerRunTranscriptDeps,
  loadTicketDrawerRunTranscript,
  type TicketDrawerRunTranscriptDeps,
} from './drawer-run-transcript'
import { applyTicketRunStreamFrame, createEmptyTicketRunTranscriptState } from './run-transcript'
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

type LoadOptions = {
  background?: boolean
  preserveMessages?: boolean
}

type TicketDrawerStateDeps = {
  fetchContext: typeof fetchTicketDetailContext
} & TicketDrawerRunTranscriptDeps

const defaultDeps: TicketDrawerStateDeps = {
  fetchContext: fetchTicketDetailContext,
  ...defaultTicketDrawerRunTranscriptDeps,
}

export function createTicketDrawerState(deps: Partial<TicketDrawerStateDeps> = {}) {
  const resolvedDeps = {
    ...defaultDeps,
    ...deps,
  }

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
    currentRun: null as TicketRun | null,
    runBlocks: [] as TicketRunTranscriptBlock[],
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
  let timelineRefreshQueued = false
  let timelineRefreshLoop: Promise<void> | null = null

  function applyFullContext(detailContext: Awaited<ReturnType<typeof fetchTicketDetailContext>>) {
    state.ticket = detailContext.ticket
    state.timeline = detailContext.timeline
    state.hooks = detailContext.hooks
    state.statuses = detailContext.statuses
    state.dependencyCandidates = detailContext.dependencyCandidates
    state.repoOptions = detailContext.repoOptions
  }

  function applyTimelineRefresh(
    detailContext: Awaited<ReturnType<typeof fetchTicketDetailContext>>,
  ) {
    state.ticket = detailContext.ticket
    state.timeline = detailContext.timeline
    state.hooks = detailContext.hooks
  }

  function applyRunTranscriptState(
    nextState: ReturnType<typeof createEmptyTicketRunTranscriptState>,
  ) {
    state.runs = nextState.runs
    state.currentRun = nextState.currentRun
    state.runBlocks = nextState.blocks
  }

  function getRunTranscriptState() {
    return {
      runs: state.runs,
      currentRun: state.currentRun,
      blocks: state.runBlocks,
    }
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
          const detailContext = await resolvedDeps.fetchContext(projectId, ticketId)
          if (requestId !== loadRequestId || !state.ticket) {
            continue
          }
          applyTimelineRefresh(detailContext)
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
      if (!options.background) {
        state.loading = true
        state.error = ''
      }
      try {
        const detailContext = await resolvedDeps.fetchContext(projectId, ticketId)
        if (requestId !== loadRequestId) return

        applyFullContext(detailContext)
        await loadTicketDrawerRunTranscript(
          resolvedDeps,
          { getState: getRunTranscriptState, setState: applyRunTranscriptState },
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
    applyRunStreamFrame(frame: Pick<SSEFrame, 'event' | 'data'>) {
      const nextState = applyTicketRunStreamFrame(getRunTranscriptState(), frame)
      applyRunTranscriptState(nextState)
    },
    reset() {
      loadRequestId += 1
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
      state.currentRun = null
      state.runBlocks = []
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
