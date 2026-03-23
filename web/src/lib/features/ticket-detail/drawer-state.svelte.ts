import { ApiError } from '$lib/api/client'
import { fetchTicketDetailContext } from './context'
import type {
  HookExecution,
  TicketActivity,
  TicketComment,
  TicketDetail,
  TicketReferenceOption,
  TicketRepoOption,
  TicketStatusOption,
} from './types'

type LoadOptions = {
  background?: boolean
  preserveMessages?: boolean
}

export function createTicketDrawerState() {
  let loading = $state(false)
  let error = $state('')
  let mutationError = $state('')
  let mutationNotice = $state('')
  let ticket = $state<TicketDetail | null>(null)
  let comments = $state<TicketComment[]>([])
  let hooks = $state<HookExecution[]>([])
  let activities = $state<TicketActivity[]>([])
  let statuses = $state<TicketStatusOption[]>([])
  let dependencyCandidates = $state<TicketReferenceOption[]>([])
  let repoOptions = $state<TicketRepoOption[]>([])
  let savingFields = $state(false)
  let creatingDependency = $state(false)
  let deletingDependencyId = $state<string | null>(null)
  let creatingRepoScope = $state(false)
  let updatingRepoScopeId = $state<string | null>(null)
  let deletingRepoScopeId = $state<string | null>(null)
  let creatingComment = $state(false)
  let updatingCommentId = $state<string | null>(null)
  let deletingCommentId = $state<string | null>(null)
  let loadRequestId = 0

  return {
    get loading() {
      return loading
    },
    get error() {
      return error
    },
    get mutationError() {
      return mutationError
    },
    get mutationNotice() {
      return mutationNotice
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
    get comments() {
      return comments
    },
    set comments(value) {
      comments = value
    },
    get activities() {
      return activities
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
    clearMutationMessages() {
      mutationError = ''
      mutationNotice = ''
    },
    setMutationError(message: string) {
      mutationNotice = ''
      mutationError = message
    },
    setMutationNotice(message: string) {
      mutationError = ''
      mutationNotice = message
    },
    async load(projectId: string, ticketId: string, options: LoadOptions = {}) {
      const requestId = ++loadRequestId
      if (!options.background) {
        loading = true
        error = ''
      }
      if (!options.preserveMessages) {
        mutationError = ''
        mutationNotice = ''
      }

      try {
        const detailContext = await fetchTicketDetailContext(projectId, ticketId)
        if (requestId !== loadRequestId) return

        ticket = detailContext.ticket
        comments = detailContext.comments
        hooks = detailContext.hooks
        activities = detailContext.activities
        statuses = detailContext.statuses
        dependencyCandidates = detailContext.dependencyCandidates
        repoOptions = detailContext.repoOptions
      } catch (caughtError) {
        if (requestId !== loadRequestId) return
        const message =
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load ticket detail.'
        if (options.background) {
          mutationError = message
        } else {
          error = message
        }
      } finally {
        if (requestId === loadRequestId && !options.background) {
          loading = false
        }
      }
    },
    reset() {
      loadRequestId += 1
      loading = false
      error = ''
      mutationError = ''
      mutationNotice = ''
      ticket = null
      comments = []
      hooks = []
      activities = []
      statuses = []
      dependencyCandidates = []
      repoOptions = []
      savingFields = false
      creatingDependency = false
      deletingDependencyId = null
      creatingRepoScope = false
      updatingRepoScopeId = null
      deletingRepoScopeId = null
      creatingComment = false
      updatingCommentId = null
      deletingCommentId = null
    },
  }
}
