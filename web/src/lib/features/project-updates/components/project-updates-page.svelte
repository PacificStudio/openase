<script lang="ts">
  import { PageScaffold } from '$lib/components/layout'
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
  import {
    isProjectUpdateEvent,
    subscribeProjectEvents,
    type ProjectEventEnvelope,
  } from '$lib/features/project-events'
  import { appStore } from '$lib/stores/app.svelte'
  import { parseProjectUpdateThreads } from '../model'
  import type { ProjectUpdateStatus, ProjectUpdateThread } from '../types'
  import {
    markProjectUpdatesCacheDirty,
    readProjectUpdatesCache,
    writeProjectUpdatesCache,
  } from '../project-updates-cache'
  import ProjectUpdateComposer from './project-update-composer.svelte'
  import ProjectUpdatesFeed from './project-updates-feed.svelte'

  let threads = $state<ProjectUpdateThread[]>([])
  let loading = $state(false)
  let error = $state('')
  let notice = $state('')
  let initialLoaded = $state(false)
  let requestVersion = 0
  let activeProjectId: string | null = null
  let queuedReload = false
  let reloadInFlight = false

  let creatingThread = $state(false)

  $effect(() => {
    const projectId = appStore.currentProject?.id
    activeProjectId = projectId ?? null
    queuedReload = false
    reloadInFlight = false
    if (!projectId) {
      threads = []
      initialLoaded = false
      loading = false
      error = ''
      return
    }

    const cachedUpdates = readProjectUpdatesCache(projectId)
    if (cachedUpdates) {
      threads = cachedUpdates.snapshot.threads
      initialLoaded = true
      loading = false
      error = ''
      if (cachedUpdates.dirty) {
        void loadProjectUpdates(projectId, { preserveMessages: true })
      }
    } else {
      initialLoaded = false
      void loadProjectUpdates(projectId, { showLoading: true })
    }

    return subscribeProjectEvents(projectId, (event) => {
      if (isProjectUpdateFrame(event)) {
        markProjectUpdatesCacheDirty(projectId)
        requestReload(projectId)
      }
    })
  })

  async function loadProjectUpdates(
    projectId: string,
    options: { showLoading?: boolean; preserveMessages?: boolean } = {},
  ) {
    const version = ++requestVersion
    if (options.showLoading) {
      loading = true
    }
    error = ''
    if (!options.preserveMessages) {
      notice = ''
    }

    try {
      const payload = await listProjectUpdates(projectId)
      if (version !== requestVersion || activeProjectId !== projectId) return
      const nextThreads = parseProjectUpdateThreads(payload.threads)
      threads = nextThreads
      initialLoaded = true
      writeProjectUpdatesCache(projectId, nextThreads)
    } catch (caughtError) {
      if (version !== requestVersion || activeProjectId !== projectId) return
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load updates.'
    } finally {
      if (version === requestVersion && activeProjectId === projectId) {
        loading = false
      }
    }
  }

  const requestReload = (projectId: string) => {
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
      await loadProjectUpdates(projectId, { preserveMessages: true })
    } finally {
      reloadInFlight = false
      if (queuedReload && activeProjectId === projectId) {
        void drainReloadQueue(projectId)
      }
    }
  }

  async function handleCreateThread(draft: {
    status: ProjectUpdateStatus
    title: string
    body: string
  }) {
    const projectId = appStore.currentProject?.id
    if (!projectId || creatingThread) return false

    creatingThread = true
    error = ''
    notice = ''

    try {
      await createProjectUpdateThread(projectId, draft)
      notice = 'Update posted.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
      return true
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to post update.'
      return false
    } finally {
      creatingThread = false
    }
  }

  async function handleSaveThread(
    threadId: string,
    draft: { status: ProjectUpdateStatus; title: string; body: string },
  ) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return false
    error = ''
    notice = ''

    try {
      await updateProjectUpdateThread(projectId, threadId, draft)
      notice = 'Update edited.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
      return true
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to edit update.'
      return false
    }
  }

  async function handleDeleteThread(threadId: string) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return false
    error = ''
    notice = ''

    try {
      await deleteProjectUpdateThread(projectId, threadId)
      notice = 'Update deleted.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
      return true
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete update.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
      return false
    }
  }

  async function handleCreateComment(threadId: string, body: string) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return false
    error = ''
    notice = ''

    try {
      await createProjectUpdateComment(projectId, threadId, { body })
      notice = 'Comment added.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
      return true
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to add comment.'
      return false
    }
  }

  async function handleSaveComment(threadId: string, commentId: string, body: string) {
    const projectId = appStore.currentProject?.id
    if (!projectId || !body) return false
    error = ''
    notice = ''

    try {
      await updateProjectUpdateComment(projectId, threadId, commentId, { body })
      notice = 'Comment edited.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
      return true
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to edit comment.'
      return false
    }
  }

  async function handleDeleteComment(threadId: string, commentId: string) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return false
    error = ''
    notice = ''

    try {
      await deleteProjectUpdateComment(projectId, threadId, commentId)
      notice = 'Comment deleted.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
      return true
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete comment.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
      return false
    }
  }

  function isProjectUpdateFrame(event: ProjectEventEnvelope) {
    return isProjectUpdateEvent(event)
  }
</script>

<PageScaffold
  title="Updates"
  description="Curated project progress threads with status and discussion. Raw runtime logs stay in Activity."
>
  <div class="w-full space-y-5">
    <ProjectUpdateComposer creating={creatingThread} onSubmit={handleCreateThread} />
    <ProjectUpdatesFeed
      {threads}
      {loading}
      {initialLoaded}
      {error}
      {notice}
      onUpdateThread={handleSaveThread}
      onDeleteThread={handleDeleteThread}
      onCreateComment={handleCreateComment}
      onUpdateComment={handleSaveComment}
      onDeleteComment={handleDeleteComment}
    />
  </div>
</PageScaffold>
