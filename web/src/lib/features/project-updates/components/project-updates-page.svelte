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
  import { cn } from '$lib/utils'
  import { Skeleton } from '$ui/skeleton'
  import { MessageSquare } from '@lucide/svelte'
  import { parseProjectUpdateThreads } from '../model'
  import type { ProjectUpdateStatus, ProjectUpdateThread } from '../types'
  import ProjectUpdateComposer from './project-update-composer.svelte'
  import ProjectUpdateThreadCard from './project-update-thread-card.svelte'

  let threads = $state<ProjectUpdateThread[]>([])
  let loading = $state(false)
  let error = $state('')
  let notice = $state('')
  let initialLoaded = $state(false)
  let requestVersion = 0

  let creatingThread = $state(false)

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      threads = []
      initialLoaded = false
      return
    }

    initialLoaded = false
    void loadProjectUpdates(projectId, { showLoading: true })

    return subscribeProjectEvents(projectId, (event) => {
      if (isProjectUpdateFrame(event)) {
        void loadProjectUpdates(projectId, { preserveMessages: true })
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
      if (version !== requestVersion) return
      threads = parseProjectUpdateThreads(payload.threads)
      initialLoaded = true
    } catch (caughtError) {
      if (version !== requestVersion) return
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load updates.'
    } finally {
      if (version === requestVersion) {
        loading = false
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

    {#if error}
      <div
        class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
      >
        {error}
      </div>
    {/if}

    {#if notice}
      <div
        class="rounded-md border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700"
      >
        {notice}
      </div>
    {/if}

    {#if loading && !initialLoaded}
      <div class="space-y-4">
        {#each { length: 2 } as _, i}
          <div class="border-border rounded-2xl border p-5 shadow-sm">
            <div class="space-y-3">
              <div class="flex items-center gap-2">
                <Skeleton class="h-5 w-20 rounded-full" />
                <Skeleton class="h-5 w-28 rounded-full" />
              </div>
              <Skeleton class={cn('h-6 rounded', i === 0 ? 'w-2/3' : 'w-1/2')} />
              <Skeleton class="h-4 w-48 rounded" />
              <div class="space-y-2">
                <Skeleton class="h-4 w-full rounded" />
                <Skeleton class="h-4 w-5/6 rounded" />
              </div>
            </div>
          </div>
        {/each}
      </div>
    {:else if threads.length === 0}
      <div
        class="flex flex-col items-center justify-center rounded-2xl border border-dashed py-18 text-center"
      >
        <div class="bg-muted/60 mb-4 flex size-12 items-center justify-center rounded-full">
          <MessageSquare class="text-muted-foreground size-5" />
        </div>
        <p class="text-sm font-medium">No curated updates yet</p>
        <p class="text-muted-foreground mt-1 max-w-md text-sm">
          Post the first project update to capture current delivery status. Raw agent and workflow
          events continue to appear in Activity.
        </p>
      </div>
    {:else}
      <div class="space-y-4">
        {#each threads as thread (thread.id)}
          <ProjectUpdateThreadCard
            {thread}
            onUpdateThread={handleSaveThread}
            onDeleteThread={handleDeleteThread}
            onCreateComment={handleCreateComment}
            onUpdateComment={handleSaveComment}
            onDeleteComment={handleDeleteComment}
          />
        {/each}
      </div>
    {/if}
  </div>
</PageScaffold>
