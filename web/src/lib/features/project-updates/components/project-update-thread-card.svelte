<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { AlertTriangle, CircleCheck, CircleX, Pencil, Send, Trash2, X } from '@lucide/svelte'
  import { isProjectUpdateEdited, projectUpdateEditedLabel } from '../metadata'
  import { projectUpdateStatusLabel } from '../status'
  import type { ProjectUpdateStatus, ProjectUpdateThread } from '../types'
  import ProjectUpdateCommentItem from './project-update-comment-item.svelte'

  let {
    thread,
    onUpdateThread,
    onDeleteThread,
    onCreateComment,
    onUpdateComment,
    onDeleteComment,
  }: {
    thread: ProjectUpdateThread
    onUpdateThread?: (
      threadId: string,
      draft: { status: ProjectUpdateStatus; title: string; body: string },
    ) => Promise<boolean> | boolean
    onDeleteThread?: (threadId: string) => Promise<boolean> | boolean
    onCreateComment?: (threadId: string, body: string) => Promise<boolean> | boolean
    onUpdateComment?: (
      threadId: string,
      commentId: string,
      body: string,
    ) => Promise<boolean> | boolean
    onDeleteComment?: (threadId: string, commentId: string) => Promise<boolean> | boolean
  } = $props()

  let editingThread = $state(false)
  let editingStatus = $state<ProjectUpdateStatus>('on_track')
  let editingTitle = $state('')
  let commentDraft = $state('')
  let savingThread = $state(false)
  let deletingThread = $state(false)
  let creatingComment = $state(false)
  let showComments = $state(false)

  const statusConfig: Record<
    ProjectUpdateStatus,
    { icon: typeof CircleCheck; dotClass: string; textClass: string }
  > = {
    on_track: {
      icon: CircleCheck,
      dotClass: 'text-emerald-500',
      textClass: 'text-emerald-700 dark:text-emerald-400',
    },
    at_risk: {
      icon: AlertTriangle,
      dotClass: 'text-amber-500',
      textClass: 'text-amber-700 dark:text-amber-400',
    },
    off_track: {
      icon: CircleX,
      dotClass: 'text-rose-500',
      textClass: 'text-rose-700 dark:text-rose-400',
    },
  }

  const threadStatusCfg = $derived(statusConfig[thread.status])
  const ThreadStatusIcon = $derived(threadStatusCfg.icon)

  const editStatusOptions: Array<{
    value: ProjectUpdateStatus
    label: string
    icon: typeof CircleCheck
    activeClass: string
  }> = [
    {
      value: 'on_track',
      label: 'On track',
      icon: CircleCheck,
      activeClass:
        'border-emerald-400 bg-emerald-50 text-emerald-700 dark:border-emerald-600 dark:bg-emerald-950/40 dark:text-emerald-300',
    },
    {
      value: 'at_risk',
      label: 'At risk',
      icon: AlertTriangle,
      activeClass:
        'border-amber-400 bg-amber-50 text-amber-700 dark:border-amber-600 dark:bg-amber-950/40 dark:text-amber-300',
    },
    {
      value: 'off_track',
      label: 'Off track',
      icon: CircleX,
      activeClass:
        'border-rose-400 bg-rose-50 text-rose-700 dark:border-rose-600 dark:bg-rose-950/40 dark:text-rose-300',
    },
  ]

  $effect(() => {
    editingStatus = thread.status
    editingTitle = thread.title
    if (thread.isDeleted) {
      editingThread = false
      commentDraft = ''
    }
  })

  function cancelThreadEdit() {
    editingThread = false
    editingStatus = thread.status
    editingTitle = thread.title
  }

  async function handleSaveThread() {
    const title = editingTitle.trim()
    if (!title || savingThread) return

    savingThread = true
    try {
      const success =
        (await onUpdateThread?.(thread.id, { status: editingStatus, title, body: title })) ?? false
      if (success) editingThread = false
    } finally {
      savingThread = false
    }
  }

  async function handleDeleteThread() {
    if (deletingThread || !window.confirm('Delete this update?')) return

    deletingThread = true
    try {
      const success = (await onDeleteThread?.(thread.id)) ?? false
      if (success) editingThread = false
    } finally {
      deletingThread = false
    }
  }

  async function handleCreateComment() {
    const body = commentDraft.trim()
    if (!body || creatingComment) return

    creatingComment = true
    try {
      const success = (await onCreateComment?.(thread.id, body)) ?? false
      if (success) {
        commentDraft = ''
        showComments = true
      }
    } finally {
      creatingComment = false
    }
  }

  function handleCommentKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey && commentDraft.trim() && !creatingComment) {
      e.preventDefault()
      void handleCreateComment()
    }
  }
</script>

{#if editingThread}
  <div class="border-border rounded-xl border px-3 py-2.5">
    <div class="flex items-center gap-1 pb-1.5">
      {#each editStatusOptions as opt (opt.value)}
        {@const Icon = opt.icon}
        <button
          type="button"
          class={cn(
            'flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium transition-colors',
            editingStatus === opt.value
              ? opt.activeClass
              : 'text-muted-foreground hover:bg-muted border-transparent',
          )}
          onclick={() => (editingStatus = opt.value)}
        >
          <Icon class="size-3" />
          {opt.label}
        </button>
      {/each}
    </div>
    <div class="flex items-center gap-2">
      <Input
        bind:value={editingTitle}
        class="h-8 flex-1 border-none bg-transparent px-0 text-sm shadow-none focus-visible:ring-0"
        aria-label={`Edit title for ${thread.title}`}
      />
      <Button
        size="sm"
        class="h-7 px-2.5 text-xs"
        onclick={handleSaveThread}
        disabled={!editingTitle.trim() || savingThread}
      >
        {savingThread ? 'Saving...' : 'Save'}
      </Button>
      <button
        type="button"
        class="text-muted-foreground hover:text-foreground transition-colors"
        onclick={cancelThreadEdit}
      >
        <X class="size-3.5" />
      </button>
    </div>
  </div>
{:else}
  <div
    class={cn(
      'group border-border hover:bg-muted/30 rounded-xl border px-3 py-2.5 transition-colors',
      thread.isDeleted && 'opacity-60',
    )}
  >
    <div class="flex items-start gap-2.5">
      <ThreadStatusIcon class={cn('mt-0.5 size-4 shrink-0', threadStatusCfg.dotClass)} />
      <div class="min-w-0 flex-1">
        <div class="flex items-baseline gap-2">
          <span class={cn('text-sm font-medium', thread.isDeleted && 'line-through')}>
            {thread.title}
          </span>
          <span class={cn('shrink-0 text-[10px] font-medium', threadStatusCfg.textClass)}>
            {projectUpdateStatusLabel(thread.status)}
          </span>
        </div>
        <div class="text-muted-foreground mt-0.5 flex flex-wrap items-center gap-x-1.5 text-[11px]">
          <span>{thread.createdBy}</span>
          <span>&middot;</span>
          <span>{formatRelativeTime(thread.createdAt)}</span>
          {#if isProjectUpdateEdited(thread.createdAt, thread.updatedAt, thread.editedAt)}
            <span>&middot;</span>
            <span
              >{projectUpdateEditedLabel(thread.createdAt, thread.updatedAt, thread.editedAt)}</span
            >
          {/if}
          {#if thread.commentCount > 0}
            <span>&middot;</span>
            <button
              type="button"
              class="hover:text-foreground transition-colors"
              onclick={() => (showComments = !showComments)}
            >
              {thread.commentCount}
              {thread.commentCount === 1 ? 'reply' : 'replies'}
            </button>
          {/if}
        </div>
      </div>

      <div
        class="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100"
      >
        {#if !thread.isDeleted}
          <button
            type="button"
            class="text-muted-foreground hover:text-foreground rounded p-1 transition-colors"
            aria-label={`Edit update ${thread.title}`}
            onclick={() => (editingThread = true)}
          >
            <Pencil class="size-3" />
          </button>
        {/if}
        <button
          type="button"
          class="text-muted-foreground hover:text-destructive rounded p-1 transition-colors"
          aria-label={`Delete update ${thread.title}`}
          onclick={handleDeleteThread}
          disabled={thread.isDeleted || deletingThread}
        >
          <Trash2 class="size-3" />
        </button>
      </div>
    </div>

    {#if thread.isDeleted}
      <p class="text-muted-foreground mt-1.5 ml-6.5 text-xs italic">Deleted</p>
    {/if}

    {#if showComments || thread.commentCount === 0}
      {#if thread.comments.length > 0 || !thread.isDeleted}
        <div class="mt-2 ml-6.5 space-y-2">
          {#each thread.comments as comment (comment.id)}
            <ProjectUpdateCommentItem
              threadId={thread.id}
              {comment}
              onUpdate={onUpdateComment}
              onDelete={onDeleteComment}
            />
          {/each}

          {#if !thread.isDeleted}
            <div class="flex items-center gap-2">
              <input
                type="text"
                bind:value={commentDraft}
                onkeydown={handleCommentKeydown}
                placeholder="Reply..."
                aria-label={`Reply to ${thread.title}`}
                class="text-foreground placeholder:text-muted-foreground min-w-0 flex-1 bg-transparent text-xs outline-none"
              />
              <button
                type="button"
                class={cn(
                  'shrink-0 rounded p-1 transition-colors',
                  commentDraft.trim() && !creatingComment
                    ? 'text-primary hover:bg-primary/10'
                    : 'text-muted-foreground/30 cursor-not-allowed',
                )}
                disabled={!commentDraft.trim() || creatingComment}
                onclick={handleCreateComment}
                aria-label="Send reply"
              >
                <Send class="size-3" />
              </button>
            </div>
          {/if}
        </div>
      {/if}
    {/if}
  </div>
{/if}
