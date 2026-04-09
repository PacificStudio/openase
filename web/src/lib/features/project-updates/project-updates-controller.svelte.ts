import { ApiError } from '$lib/api/client'
import {
  createProjectUpdateComment,
  createProjectUpdateThread,
  deleteProjectUpdateComment,
  deleteProjectUpdateThread,
  listProjectUpdates,
  updateProjectUpdateComment,
  updateProjectUpdateThread,
} from '$lib/api/openase'
import { isProjectUpdateEvent, subscribeProjectEvents } from '$lib/features/project-events'
import { toastStore } from '$lib/stores/toast.svelte'
import { mergeProjectUpdateThreads, parseProjectUpdatePage } from './model'
import {
  markProjectUpdatesCacheDirty,
  readProjectUpdatesCache,
  writeProjectUpdatesCache,
} from './project-updates-cache'
import type { ProjectUpdateStatus, ProjectUpdateThread } from './types'

const defaultThreadPageLimit = 10

type LoadProjectUpdatesOptions = {
  showLoading?: boolean
}

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
      writeSnapshot(projectId)
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
      writeSnapshot(projectId)
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

  async function handleCreateThread(draft: { status: ProjectUpdateStatus; body: string }) {
    const projectId = input.getProjectId()
    if (!projectId || creatingThread) {
      return false
    }

    creatingThread = true

    try {
      await createProjectUpdateThread(projectId, draft)
      toastStore.success('Update posted.')
      await refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to post update.',
      )
      return false
    } finally {
      creatingThread = false
    }
  }

  async function handleSaveThread(
    threadId: string,
    draft: { status: ProjectUpdateStatus; body: string },
  ) {
    const projectId = input.getProjectId()
    if (!projectId) {
      return false
    }
    try {
      await updateProjectUpdateThread(projectId, threadId, draft)
      toastStore.success('Update edited.')
      await refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to edit update.',
      )
      return false
    }
  }

  async function handleDeleteThread(threadId: string) {
    const projectId = input.getProjectId()
    if (!projectId) {
      return false
    }

    try {
      await deleteProjectUpdateThread(projectId, threadId)
      toastStore.success('Update deleted.')
      await refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete update.',
      )
      await refreshLatestThreads(projectId)
      return false
    }
  }

  async function handleCreateComment(threadId: string, body: string) {
    const projectId = input.getProjectId()
    if (!projectId) {
      return false
    }

    try {
      await createProjectUpdateComment(projectId, threadId, { body })
      toastStore.success('Comment added.')
      await refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to add comment.',
      )
      return false
    }
  }

  async function handleSaveComment(threadId: string, commentId: string, body: string) {
    const projectId = input.getProjectId()
    if (!projectId || !body) {
      return false
    }

    try {
      await updateProjectUpdateComment(projectId, threadId, commentId, { body })
      toastStore.success('Comment edited.')
      await refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to edit comment.',
      )
      return false
    }
  }

  async function handleDeleteComment(threadId: string, commentId: string) {
    const projectId = input.getProjectId()
    if (!projectId) {
      return false
    }

    try {
      await deleteProjectUpdateComment(projectId, threadId, commentId)
      toastStore.success('Comment deleted.')
      await refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete comment.',
      )
      await refreshLatestThreads(projectId)
      return false
    }
  }

  function writeSnapshot(projectId: string) {
    writeProjectUpdatesCache(projectId, {
      threads,
      nextCursor,
      hasMoreThreads,
      loadedMorePages,
    })
  }

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
