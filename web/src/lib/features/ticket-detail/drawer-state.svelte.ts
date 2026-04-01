import { ApiError } from '$lib/api/client'
import { toastStore } from '$lib/stores/toast.svelte'
import { fetchTicketDetailContext } from './context'
import type {
  HookExecution,
  TicketDetail,
  TicketReferenceOption,
  TicketRepoOption,
  TicketStatusOption,
  TicketTimelineItem,
} from './types'

type LoadOptions = {
  background?: boolean
  preserveMessages?: boolean
}

type TicketDrawerStateDeps = {
  fetchContext: typeof fetchTicketDetailContext
}

const defaultDeps: TicketDrawerStateDeps = {
  fetchContext: fetchTicketDetailContext,
}

export function createTicketDrawerState(deps: TicketDrawerStateDeps = defaultDeps) {
  let loading = $state(false)
  let error = $state('')
  let ticket = $state<TicketDetail | null>(null)
  let timeline = $state<TicketTimelineItem[]>([])
  let hooks = $state<HookExecution[]>([])
  let statuses = $state<TicketStatusOption[]>([])
  let dependencyCandidates = $state<TicketReferenceOption[]>([])
  let repoOptions = $state<TicketRepoOption[]>([])
  let savingFields = $state(false)
  let creatingDependency = $state(false)
  let deletingDependencyId = $state<string | null>(null)
  let creatingExternalLink = $state(false)
  let deletingExternalLinkId = $state<string | null>(null)
  let creatingRepoScope = $state(false)
  let updatingRepoScopeId = $state<string | null>(null)
  let deletingRepoScopeId = $state<string | null>(null)
  let creatingComment = $state(false)
  let updatingCommentId = $state<string | null>(null)
  let deletingCommentId = $state<string | null>(null)
  let resumingRetry = $state(false)
  let loadRequestId = 0
  let timelineRefreshQueued = false
  let timelineRefreshLoop: Promise<void> | null = null

  function applyFullContext(detailContext: Awaited<ReturnType<typeof fetchTicketDetailContext>>) {
    ticket = detailContext.ticket
    timeline = detailContext.timeline
    hooks = detailContext.hooks
    statuses = detailContext.statuses
    dependencyCandidates = detailContext.dependencyCandidates
    repoOptions = detailContext.repoOptions
  }

  function applyTimelineRefresh(
    detailContext: Awaited<ReturnType<typeof fetchTicketDetailContext>>,
  ) {
    ticket = detailContext.ticket
    timeline = detailContext.timeline
    hooks = detailContext.hooks
  }

  async function runTimelineRefresh(projectId: string, ticketId: string) {
    if (loading || !ticket) {
      return
    }
    if (timelineRefreshLoop) {
      await timelineRefreshLoop
      return
    }

    timelineRefreshLoop = (async () => {
      while (timelineRefreshQueued && !loading && ticket) {
        timelineRefreshQueued = false
        const requestId = loadRequestId
        try {
          const detailContext = await deps.fetchContext(projectId, ticketId)
          if (requestId !== loadRequestId || !ticket) {
            continue
          }
          applyTimelineRefresh(detailContext)
        } catch (caughtError) {
          if (requestId !== loadRequestId || !ticket) {
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

  return {
    get loading() {
      return loading
    },
    get error() {
      return error
    },
    get ticket() {
      return ticket
    },
    set ticket(value) {
      ticket = value
    },
    get hooks() {
      return hooks
    },
    get timeline() {
      return timeline
    },
    set timeline(value) {
      timeline = value
    },
    get statuses() {
      return statuses
    },
    get dependencyCandidates() {
      return dependencyCandidates
    },
    get repoOptions() {
      return repoOptions
    },
    get savingFields() {
      return savingFields
    },
    set savingFields(value) {
      savingFields = value
    },
    get creatingDependency() {
      return creatingDependency
    },
    set creatingDependency(value) {
      creatingDependency = value
    },
    get deletingDependencyId() {
      return deletingDependencyId
    },
    set deletingDependencyId(value) {
      deletingDependencyId = value
    },
    get creatingExternalLink() {
      return creatingExternalLink
    },
    set creatingExternalLink(value) {
      creatingExternalLink = value
    },
    get deletingExternalLinkId() {
      return deletingExternalLinkId
    },
    set deletingExternalLinkId(value) {
      deletingExternalLinkId = value
    },
    get creatingRepoScope() {
      return creatingRepoScope
    },
    set creatingRepoScope(value) {
      creatingRepoScope = value
    },
    get updatingRepoScopeId() {
      return updatingRepoScopeId
    },
    set updatingRepoScopeId(value) {
      updatingRepoScopeId = value
    },
    get deletingRepoScopeId() {
      return deletingRepoScopeId
    },
    set deletingRepoScopeId(value) {
      deletingRepoScopeId = value
    },
    get creatingComment() {
      return creatingComment
    },
    set creatingComment(value) {
      creatingComment = value
    },
    get updatingCommentId() {
      return updatingCommentId
    },
    set updatingCommentId(value) {
      updatingCommentId = value
    },
    get deletingCommentId() {
      return deletingCommentId
    },
    set deletingCommentId(value) {
      deletingCommentId = value
    },
    get resumingRetry() {
      return resumingRetry
    },
    set resumingRetry(value) {
      resumingRetry = value
    },
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
        loading = true
        error = ''
      }
      try {
        const detailContext = await deps.fetchContext(projectId, ticketId)
        if (requestId !== loadRequestId) return

        applyFullContext(detailContext)
      } catch (caughtError) {
        if (requestId !== loadRequestId) return
        const message =
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load ticket detail.'
        if (options.background) {
          toastStore.error(message)
        } else {
          error = message
        }
      } finally {
        if (requestId === loadRequestId && !options.background) {
          loading = false
        }
      }
    },
    async refreshTimeline(projectId: string, ticketId: string) {
      if (loading || !ticket) {
        return
      }
      timelineRefreshQueued = true
      await runTimelineRefresh(projectId, ticketId)
      if (timelineRefreshQueued) {
        await runTimelineRefresh(projectId, ticketId)
      }
    },
    reset() {
      loadRequestId += 1
      timelineRefreshQueued = false
      timelineRefreshLoop = null
      loading = false
      error = ''
      ticket = null
      timeline = []
      hooks = []
      statuses = []
      dependencyCandidates = []
      repoOptions = []
      savingFields = false
      creatingDependency = false
      deletingDependencyId = null
      creatingExternalLink = false
      deletingExternalLinkId = null
      creatingRepoScope = false
      updatingRepoScopeId = null
      deletingRepoScopeId = null
      creatingComment = false
      updatingCommentId = null
      deletingCommentId = null
      resumingRetry = false
    },
  }
}
