<script lang="ts">
  import { PageScaffold } from '$lib/components/layout'
  import { ApiError } from '$lib/api/client'
  import { connectEventStream, type SSEFrame } from '$lib/api/sse'
  import {
    createProjectUpdateComment,
    createProjectUpdateThread,
    deleteProjectUpdateComment,
    deleteProjectUpdateThread,
    listProjectUpdates,
    updateProjectUpdateComment,
    updateProjectUpdateThread,
  } from '$lib/api/openase'
  import { appStore } from '$lib/stores/app.svelte'
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Skeleton } from '$ui/skeleton'
  import { Textarea } from '$ui/textarea'
  import { MessageSquare, Pencil, Trash2 } from '@lucide/svelte'
  import { parseProjectUpdateThreads } from '../model'
  import ProjectUpdateMarkdownContent from './project-update-markdown-content.svelte'
  import {
    projectUpdateStatusBadgeClass,
    projectUpdateStatusLabel,
    projectUpdateStatusOptions,
  } from '../status'
  import type { ProjectUpdateComment, ProjectUpdateStatus, ProjectUpdateThread } from '../types'

  let threads = $state<ProjectUpdateThread[]>([])
  let loading = $state(false)
  let error = $state('')
  let notice = $state('')
  let initialLoaded = $state(false)
  let requestVersion = 0

  let composerStatus = $state<ProjectUpdateStatus>('on_track')
  let composerTitle = $state('')
  let composerBody = $state('')
  let creatingThread = $state(false)

  let editingThreadId = $state<string | null>(null)
  let editingThreadStatus = $state<ProjectUpdateStatus>('on_track')
  let editingThreadTitle = $state('')
  let editingThreadBody = $state('')
  let updatingThreadId = $state<string | null>(null)
  let deletingThreadId = $state<string | null>(null)

  let commentDraftByThreadId = $state<Record<string, string>>({})
  let creatingCommentThreadId = $state<string | null>(null)
  let editingCommentId = $state<string | null>(null)
  let editingCommentBody = $state('')
  let updatingCommentId = $state<string | null>(null)
  let deletingCommentId = $state<string | null>(null)

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      threads = []
      initialLoaded = false
      return
    }

    initialLoaded = false
    void loadProjectUpdates(projectId, { showLoading: true })

    return connectEventStream(`/api/v1/projects/${projectId}/activity/stream`, {
      onEvent: (frame) => {
        if (isProjectUpdateFrame(frame)) {
          void loadProjectUpdates(projectId, { preserveMessages: true })
        }
      },
      onError: (streamError) => {
        console.error('Project updates stream error:', streamError)
      },
    })
  })

  $effect(() => {
    if (!editingThreadId) return
    const current = threads.find((thread) => thread.id === editingThreadId)
    if (!current || current.isDeleted) {
      cancelThreadEdit()
    }
  })

  $effect(() => {
    if (!editingCommentId) return
    const current = findComment(editingCommentId)
    if (!current || current.isDeleted) {
      cancelCommentEdit()
    }
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

  async function handleCreateThread() {
    const projectId = appStore.currentProject?.id
    if (!projectId || creatingThread) return

    creatingThread = true
    error = ''
    notice = ''

    try {
      await createProjectUpdateThread(projectId, {
        status: composerStatus,
        title: composerTitle.trim(),
        body: composerBody.trim(),
      })
      composerStatus = 'on_track'
      composerTitle = ''
      composerBody = ''
      notice = 'Update posted.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to post update.'
    } finally {
      creatingThread = false
    }
  }

  function beginThreadEdit(thread: ProjectUpdateThread) {
    editingThreadId = thread.id
    editingThreadStatus = thread.status
    editingThreadTitle = thread.title
    editingThreadBody = thread.bodyMarkdown
  }

  function cancelThreadEdit() {
    editingThreadId = null
    editingThreadStatus = 'on_track'
    editingThreadTitle = ''
    editingThreadBody = ''
  }

  async function handleSaveThread(threadId: string) {
    const projectId = appStore.currentProject?.id
    if (!projectId || updatingThreadId === threadId) return

    updatingThreadId = threadId
    error = ''
    notice = ''

    try {
      await updateProjectUpdateThread(projectId, threadId, {
        status: editingThreadStatus,
        title: editingThreadTitle.trim(),
        body: editingThreadBody.trim(),
      })
      notice = 'Update edited.'
      cancelThreadEdit()
      await loadProjectUpdates(projectId, { preserveMessages: true })
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to edit update.'
    } finally {
      updatingThreadId = null
    }
  }

  async function handleDeleteThread(threadId: string) {
    const projectId = appStore.currentProject?.id
    if (!projectId || deletingThreadId === threadId) return
    if (!window.confirm('Delete this update thread?')) return

    deletingThreadId = threadId
    error = ''
    notice = ''

    try {
      await deleteProjectUpdateThread(projectId, threadId)
      notice = 'Update deleted.'
      if (editingThreadId === threadId) {
        cancelThreadEdit()
      }
      await loadProjectUpdates(projectId, { preserveMessages: true })
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete update.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
    } finally {
      deletingThreadId = null
    }
  }

  async function handleCreateComment(threadId: string) {
    const projectId = appStore.currentProject?.id
    const body = commentDraftByThreadId[threadId]?.trim() ?? ''
    if (!projectId || !body || creatingCommentThreadId === threadId) return

    creatingCommentThreadId = threadId
    error = ''
    notice = ''

    try {
      await createProjectUpdateComment(projectId, threadId, { body })
      commentDraftByThreadId = { ...commentDraftByThreadId, [threadId]: '' }
      notice = 'Comment added.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to add comment.'
    } finally {
      creatingCommentThreadId = null
    }
  }

  function beginCommentEdit(comment: ProjectUpdateComment) {
    editingCommentId = comment.id
    editingCommentBody = comment.bodyMarkdown
  }

  function cancelCommentEdit() {
    editingCommentId = null
    editingCommentBody = ''
  }

  async function handleSaveComment(threadId: string, commentId: string) {
    const projectId = appStore.currentProject?.id
    if (!projectId || updatingCommentId === commentId) return

    updatingCommentId = commentId
    error = ''
    notice = ''

    try {
      await updateProjectUpdateComment(projectId, threadId, commentId, {
        body: editingCommentBody.trim(),
      })
      notice = 'Comment edited.'
      cancelCommentEdit()
      await loadProjectUpdates(projectId, { preserveMessages: true })
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to edit comment.'
    } finally {
      updatingCommentId = null
    }
  }

  async function handleDeleteComment(threadId: string, commentId: string) {
    const projectId = appStore.currentProject?.id
    if (!projectId || deletingCommentId === commentId) return
    if (!window.confirm('Delete this comment?')) return

    deletingCommentId = commentId
    error = ''
    notice = ''

    try {
      await deleteProjectUpdateComment(projectId, threadId, commentId)
      notice = 'Comment deleted.'
      if (editingCommentId === commentId) {
        cancelCommentEdit()
      }
      await loadProjectUpdates(projectId, { preserveMessages: true })
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete comment.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
    } finally {
      deletingCommentId = null
    }
  }

  function setCommentDraft(threadId: string, value: string) {
    commentDraftByThreadId = { ...commentDraftByThreadId, [threadId]: value }
  }

  function findComment(commentId: string) {
    for (const thread of threads) {
      const match = thread.comments.find((comment) => comment.id === commentId)
      if (match) return match
    }
    return null
  }

  function isEdited(createdAt: string, updatedAt: string, editedAt?: string) {
    return Boolean(editedAt) || updatedAt !== createdAt
  }

  function editedLabel(createdAt: string, updatedAt: string, editedAt?: string) {
    const effective = editedAt ?? updatedAt
    return isEdited(createdAt, updatedAt, editedAt) ? `edited ${formatRelativeTime(effective)}` : ''
  }

  function threadCommentDraft(threadId: string) {
    return commentDraftByThreadId[threadId] ?? ''
  }

  function isProjectUpdateFrame(frame: SSEFrame) {
    if (frame.event !== 'message') return false

    try {
      const payload = JSON.parse(frame.data) as { type?: string }
      return typeof payload.type === 'string' && payload.type.startsWith('project_update_')
    } catch {
      return false
    }
  }
</script>

<PageScaffold
  title="Updates"
  description="Curated project progress threads with status and discussion. Raw runtime logs stay in Activity."
>
  <div class="w-full space-y-5">
    <section class="border-border bg-background rounded-2xl border shadow-sm">
      <div class="border-border border-b px-5 py-4">
        <h2 class="text-base font-semibold">Post a project update</h2>
        <p class="text-muted-foreground mt-1 text-sm">
          Use Updates for the latest human-authored project status. Runtime event logs remain in
          Activity.
        </p>
      </div>
      <div class="space-y-4 px-5 py-4">
        <div class="grid gap-3 md:grid-cols-[180px_minmax(0,1fr)]">
          <label class="space-y-1.5 text-sm">
            <span class="text-muted-foreground">Delivery status</span>
            <select
              bind:value={composerStatus}
              aria-label="New update status"
              class="border-input bg-background ring-offset-background focus-visible:ring-ring flex h-10 w-full rounded-md border px-3 py-2 text-sm focus-visible:ring-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
            >
              {#each projectUpdateStatusOptions as option (option.value)}
                <option value={option.value}>{option.label}</option>
              {/each}
            </select>
          </label>
          <label class="space-y-1.5 text-sm">
            <span class="text-muted-foreground">Title</span>
            <Input
              bind:value={composerTitle}
              aria-label="New update title"
              placeholder="Sprint 2 rollout"
            />
          </label>
        </div>
        <label class="space-y-1.5 text-sm">
          <span class="text-muted-foreground">Body</span>
          <Textarea
            bind:value={composerBody}
            aria-label="New update body"
            rows={5}
            placeholder="Summarize the latest delivery signal, risks, and next checkpoint."
          />
        </label>
        <div class="flex items-center justify-between gap-3">
          <p class="text-muted-foreground text-xs">
            This timeline is separate from system Activity and is ordered by the latest discussion.
          </p>
          <Button
            onclick={handleCreateThread}
            disabled={!composerTitle.trim() || !composerBody.trim() || creatingThread}
          >
            {creatingThread ? 'Posting…' : 'Post update'}
          </Button>
        </div>
      </div>
    </section>

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
          <article class="border-border bg-background rounded-2xl border shadow-sm">
            <div class="border-border flex items-start justify-between gap-3 border-b px-5 py-4">
              <div class="min-w-0 flex-1">
                <div class="mb-2 flex flex-wrap items-center gap-2">
                  <Badge
                    variant="outline"
                    class={cn('font-medium', projectUpdateStatusBadgeClass(thread.status))}
                  >
                    {projectUpdateStatusLabel(thread.status)}
                  </Badge>
                  {#if thread.isDeleted}
                    <Badge variant="outline">Deleted</Badge>
                  {/if}
                  <span class="text-muted-foreground text-xs">
                    {thread.commentCount}
                    {thread.commentCount === 1 ? 'comment' : 'comments'}
                  </span>
                </div>
                <h2
                  class={cn('text-lg font-semibold', thread.isDeleted && 'text-muted-foreground')}
                >
                  {thread.title}
                </h2>
                <div class="text-muted-foreground mt-1 flex flex-wrap items-center gap-2 text-xs">
                  <span>{thread.createdBy}</span>
                  <span>{formatRelativeTime(thread.createdAt)}</span>
                  {#if isEdited(thread.createdAt, thread.updatedAt, thread.editedAt)}
                    <span>{editedLabel(thread.createdAt, thread.updatedAt, thread.editedAt)}</span>
                  {/if}
                  <span>last activity {formatRelativeTime(thread.lastActivityAt)}</span>
                </div>
              </div>
              <div class="flex items-center gap-2">
                {#if editingThreadId !== thread.id}
                  <Button
                    size="icon-sm"
                    variant="ghost"
                    aria-label={`Edit update ${thread.title}`}
                    onclick={() => beginThreadEdit(thread)}
                    disabled={thread.isDeleted || deletingThreadId === thread.id}
                  >
                    <Pencil class="size-4" />
                  </Button>
                {/if}
                <Button
                  size="icon-sm"
                  variant="ghost"
                  aria-label={`Delete update ${thread.title}`}
                  onclick={() => handleDeleteThread(thread.id)}
                  disabled={thread.isDeleted || deletingThreadId === thread.id}
                >
                  <Trash2 class="size-4" />
                </Button>
              </div>
            </div>

            <div class="space-y-4 px-5 py-4">
              {#if editingThreadId === thread.id}
                <div class="space-y-3">
                  <div class="grid gap-3 md:grid-cols-[180px_minmax(0,1fr)]">
                    <label class="space-y-1.5 text-sm">
                      <span class="text-muted-foreground">Delivery status</span>
                      <select
                        bind:value={editingThreadStatus}
                        aria-label={`Edit status for ${thread.title}`}
                        class="border-input bg-background ring-offset-background focus-visible:ring-ring flex h-10 w-full rounded-md border px-3 py-2 text-sm focus-visible:ring-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                      >
                        {#each projectUpdateStatusOptions as option (option.value)}
                          <option value={option.value}>{option.label}</option>
                        {/each}
                      </select>
                    </label>
                    <label class="space-y-1.5 text-sm">
                      <span class="text-muted-foreground">Title</span>
                      <Input
                        bind:value={editingThreadTitle}
                        aria-label={`Edit title for ${thread.title}`}
                      />
                    </label>
                  </div>
                  <label class="space-y-1.5 text-sm">
                    <span class="text-muted-foreground">Body</span>
                    <Textarea
                      bind:value={editingThreadBody}
                      aria-label={`Edit body for ${thread.title}`}
                      rows={6}
                    />
                  </label>
                  <div class="flex justify-end gap-2">
                    <Button
                      size="sm"
                      variant="outline"
                      onclick={cancelThreadEdit}
                      disabled={updatingThreadId === thread.id}
                    >
                      Cancel
                    </Button>
                    <Button
                      size="sm"
                      onclick={() => handleSaveThread(thread.id)}
                      disabled={!editingThreadTitle.trim() ||
                        !editingThreadBody.trim() ||
                        updatingThreadId === thread.id}
                    >
                      {updatingThreadId === thread.id ? 'Saving…' : 'Save'}
                    </Button>
                  </div>
                </div>
              {:else if thread.isDeleted}
                <p class="text-muted-foreground text-sm italic">
                  This update was deleted. Existing discussion remains visible for timeline
                  continuity.
                </p>
              {:else}
                <ProjectUpdateMarkdownContent source={thread.bodyMarkdown} />
              {/if}
            </div>

            <div class="border-border border-t px-5 py-4">
              <div class="space-y-3">
                {#each thread.comments as comment (comment.id)}
                  <div class="bg-muted/35 rounded-xl border px-4 py-3">
                    <div class="flex items-start justify-between gap-3">
                      <div class="min-w-0 flex-1">
                        <div class="flex flex-wrap items-center gap-2 text-sm">
                          <span class="font-medium">{comment.createdBy}</span>
                          {#if comment.isDeleted}
                            <Badge variant="outline" class="h-5 px-2 text-[10px]">Deleted</Badge>
                          {/if}
                        </div>
                        <div
                          class="text-muted-foreground mt-1 flex flex-wrap items-center gap-2 text-xs"
                        >
                          <span>{formatRelativeTime(comment.createdAt)}</span>
                          {#if isEdited(comment.createdAt, comment.updatedAt, comment.editedAt)}
                            <span
                              >{editedLabel(
                                comment.createdAt,
                                comment.updatedAt,
                                comment.editedAt,
                              )}</span
                            >
                          {/if}
                        </div>
                      </div>
                      <div class="flex items-center gap-2">
                        {#if editingCommentId !== comment.id}
                          <Button
                            size="icon-sm"
                            variant="ghost"
                            aria-label={`Edit comment ${comment.id}`}
                            onclick={() => beginCommentEdit(comment)}
                            disabled={comment.isDeleted || deletingCommentId === comment.id}
                          >
                            <Pencil class="size-4" />
                          </Button>
                        {/if}
                        <Button
                          size="icon-sm"
                          variant="ghost"
                          aria-label={`Delete comment ${comment.id}`}
                          onclick={() => handleDeleteComment(thread.id, comment.id)}
                          disabled={comment.isDeleted || deletingCommentId === comment.id}
                        >
                          <Trash2 class="size-4" />
                        </Button>
                      </div>
                    </div>

                    <div class="mt-3">
                      {#if editingCommentId === comment.id}
                        <div class="space-y-3">
                          <Textarea
                            bind:value={editingCommentBody}
                            aria-label={`Edit comment ${comment.id}`}
                            rows={4}
                          />
                          <div class="flex justify-end gap-2">
                            <Button
                              size="sm"
                              variant="outline"
                              onclick={cancelCommentEdit}
                              disabled={updatingCommentId === comment.id}
                            >
                              Cancel
                            </Button>
                            <Button
                              size="sm"
                              onclick={() => handleSaveComment(thread.id, comment.id)}
                              disabled={!editingCommentBody.trim() ||
                                updatingCommentId === comment.id}
                            >
                              {updatingCommentId === comment.id ? 'Saving…' : 'Save'}
                            </Button>
                          </div>
                        </div>
                      {:else if comment.isDeleted}
                        <p class="text-muted-foreground text-sm italic">
                          This comment was deleted.
                        </p>
                      {:else}
                        <ProjectUpdateMarkdownContent source={comment.bodyMarkdown} />
                      {/if}
                    </div>
                  </div>
                {/each}

                {#if !thread.isDeleted}
                  <div class="rounded-xl border border-dashed px-4 py-4">
                    <label class="space-y-1.5 text-sm">
                      <span class="text-muted-foreground">Reply</span>
                      <Textarea
                        value={threadCommentDraft(thread.id)}
                        aria-label={`Reply to ${thread.title}`}
                        rows={3}
                        placeholder="Add a progress note, question, or follow-up."
                        oninput={(event) => {
                          setCommentDraft(
                            thread.id,
                            (event.currentTarget as HTMLTextAreaElement).value,
                          )
                        }}
                      />
                    </label>
                    <div class="mt-3 flex justify-end">
                      <Button
                        size="sm"
                        onclick={() => handleCreateComment(thread.id)}
                        disabled={!threadCommentDraft(thread.id).trim() ||
                          creatingCommentThreadId === thread.id}
                      >
                        {creatingCommentThreadId === thread.id ? 'Posting…' : 'Add comment'}
                      </Button>
                    </div>
                  </div>
                {/if}
              </div>
            </div>
          </article>
        {/each}
      </div>
    {/if}
  </div>
</PageScaffold>
