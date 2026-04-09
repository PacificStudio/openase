import { ApiError } from '$lib/api/client'
import { listProjectUpdates } from '$lib/api/openase'
import { isProjectUpdateEvent, subscribeProjectEvents } from '$lib/features/project-events'
import { toastStore } from '$lib/stores/toast.svelte'
import { mergeProjectUpdateThreads, parseProjectUpdatePage } from './model'
import {
  defaultThreadPageLimit,
  persistProjectUpdatesSnapshot,
  type LoadProjectUpdatesOptions,
} from './project-updates-controller-helpers'
import { createProjectUpdateMutationHandlers } from './project-updates-controller-mutations'
import { markProjectUpdatesCacheDirty, readProjectUpdatesCache } from './project-updates-cache'
import type { ProjectUpdateThread } from './types'

type CreateProjectUpdatesControllerInput = {
  getProjectId: () => string
  threadPageLimit?: number
}

export function createProjectUpdatesController(input: CreateProjectUpdatesControllerInput) {
  let threads = $state<ProjectUpdateThread[]>([])
  let loading = $state(false)
  let loadingMoreThreads = $state(false)
  let loadError = $state('')
  let initialLoaded = $state(false)
  let creatingThread = $state(false)
  let hasMoreThreads = $state(false)
  let nextCursor = $state('')

  let requestVersion = 0
  let activeProjectId: string | null = null
  let queuedReload = false
  let reloadInFlight = false
  let loadedMorePages = false

  const threadPageLimit = Math.max(1, Math.trunc(input.threadPageLimit ?? defaultThreadPageLimit))

  $effect(() => {
    const projectId = input.getProjectId()
    activeProjectId = projectId || null
    queuedReload = false
    reloadInFlight = false
    loadedMorePages = false

    if (!projectId) {
      threads = []
      initialLoaded = false
      loading = false
      loadingMoreThreads = false
      loadError = ''
      hasMoreThreads = false
      nextCursor = ''
      return
    }

    const cachedUpdates = readProjectUpdatesCache(projectId)
    if (cachedUpdates) {
      threads = cachedUpdates.snapshot.threads
      hasMoreThreads = cachedUpdates.snapshot.hasMoreThreads
      nextCursor = cachedUpdates.snapshot.nextCursor
      loadedMorePages = cachedUpdates.snapshot.loadedMorePages
      initialLoaded = true
      loading = false
      loadError = ''
      if (cachedUpdates.dirty) {
        void refreshLatestThreads(projectId)
      }
    } else {
      initialLoaded = false
      void refreshLatestThreads(projectId, { showLoading: true })
    }

    return subscribeProjectEvents(projectId, (event) => {
      if (!isProjectUpdateEvent(event)) {
        return
      }
      markProjectUpdatesCacheDirty(projectId)
      requestReload(projectId)
    })
  })

  async function refreshLatestThreads(projectId: string, options: LoadProjectUpdatesOptions = {}) {
    const version = ++requestVersion
    if (options.showLoading) {
      loading = true
    }
    loadError = ''

    try {
      const payload = await listProjectUpdates(projectId, { limit: threadPageLimit })
      if (version !== requestVersion || activeProjectId !== projectId) {
        return
      }

      const page = parseProjectUpdatePage(payload)
      threads = loadedMorePages ? mergeProjectUpdateThreads(page.threads, threads) : page.threads
      if (!loadedMorePages) {
        hasMoreThreads = page.hasMore
        nextCursor = page.nextCursor
      }
      initialLoaded = true
      persistProjectUpdatesSnapshot(projectId, {
        threads,
        nextCursor,
        hasMoreThreads,
        loadedMorePages,
      })
    } catch (caughtError) {
      if (version !== requestVersion || activeProjectId !== projectId) {
        return
      }
      loadError = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load updates.'
    } finally {
      if (version === requestVersion && activeProjectId === projectId) {
        loading = false
      }
    }
  }

  async function handleLoadMoreThreads() {
    const projectId = input.getProjectId()
    if (!projectId || loadingMoreThreads || !hasMoreThreads || !nextCursor) {
      return false
    }

    loadingMoreThreads = true
    try {
      const payload = await listProjectUpdates(projectId, {
        limit: threadPageLimit,
        before: nextCursor,
      })
      if (activeProjectId !== projectId) {
        return false
      }

      const page = parseProjectUpdatePage(payload)
      threads = mergeProjectUpdateThreads(threads, page.threads)
      hasMoreThreads = page.hasMore
      nextCursor = page.nextCursor
      loadedMorePages = true
      persistProjectUpdatesSnapshot(projectId, {
        threads,
        nextCursor,
        hasMoreThreads,
        loadedMorePages,
      })
      return true
    } catch (caughtError) {
      if (activeProjectId === projectId) {
        toastStore.error(
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load older updates.',
        )
      }
      return false
    } finally {
      if (activeProjectId === projectId) {
        loadingMoreThreads = false
      }
    }
  }

  function requestReload(projectId: string) {
    queuedReload = true
    void drainReloadQueue(projectId)
  }

  async function drainReloadQueue(projectId: string) {
    if (!queuedReload || reloadInFlight || activeProjectId !== projectId) {
      return
    }

    reloadInFlight = true
    queuedReload = false
    try {
      await refreshLatestThreads(projectId)
    } finally {
      reloadInFlight = false
      if (queuedReload && activeProjectId === projectId) {
        void drainReloadQueue(projectId)
      }
    }
  }

  const {
    handleCreateThread,
    handleSaveThread,
    handleDeleteThread,
    handleCreateComment,
    handleSaveComment,
    handleDeleteComment,
  } = createProjectUpdateMutationHandlers({
    getProjectId: input.getProjectId,
    isCreatingThread: () => creatingThread,
    setCreatingThread: (value) => {
      creatingThread = value
    },
    refreshLatestThreads,
  })

  return {
    get threads() {
      return threads
    },
    get loading() {
      return loading
    },
    get loadingMoreThreads() {
      return loadingMoreThreads
    },
    get hasMoreThreads() {
      return hasMoreThreads
    },
    get nextCursor() {
      return nextCursor
    },
    get loadError() {
      return loadError
    },
    get initialLoaded() {
      return initialLoaded
    },
    get creatingThread() {
      return creatingThread
    },
    handleCreateThread,
    handleSaveThread,
    handleDeleteThread,
    handleCreateComment,
    handleSaveComment,
    handleDeleteComment,
    handleLoadMoreThreads,
  }
}
