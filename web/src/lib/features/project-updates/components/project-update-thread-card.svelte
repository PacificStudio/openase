<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import * as Select from '$ui/select'
  import { Pencil, Trash2, X } from '@lucide/svelte'
  import { isProjectUpdateEdited, projectUpdateEditedLabel } from '../metadata'
  import { projectUpdateStatusLabel } from '../status'
  import type { ProjectUpdateStatus, ProjectUpdateThread } from '../types'
  import {
    projectUpdateStatusConfig,
    projectUpdateStatusOptions,
  } from '../project-update-thread-status'
  import ProjectUpdateThreadReplies from './project-update-thread-replies.svelte'

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

  const threadStatusCfg = $derived(projectUpdateStatusConfig[thread.status])
  const ThreadStatusIcon = $derived(threadStatusCfg.icon)

  const currentEditStatusOption = $derived(
    projectUpdateStatusOptions.find((option) => option.value === editingStatus) ??
      projectUpdateStatusOptions[0],
  )
  const CurrentEditStatusIcon = $derived(currentEditStatusOption.icon)

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
  <div class="flex items-center gap-2 rounded-md px-2.5 py-1.5">
    <Select.Root
      type="single"
      value={editingStatus}
      onValueChange={(value) => {
        if (value) editingStatus = value as ProjectUpdateStatus
      }}
    >
      <Select.Trigger
        size="sm"
        class={cn(
          'w-auto shrink-0 gap-1 border-none px-1.5 text-xs font-medium shadow-none',
          currentEditStatusOption.textClass,
        )}
      >
        <CurrentEditStatusIcon class="size-3" />
        {currentEditStatusOption.label}
      </Select.Trigger>
      <Select.Content>
        {#each projectUpdateStatusOptions as opt (opt.value)}
          {@const Icon = opt.icon}
          <Select.Item value={opt.value}>
            <Icon class={cn('size-3', opt.textClass)} />
            {opt.label}
          </Select.Item>
        {/each}
      </Select.Content>
    </Select.Root>
    <Input
      bind:value={editingTitle}
      class="h-7 flex-1 border-none bg-transparent px-0 text-sm shadow-none focus-visible:ring-0"
      aria-label={`Edit title for ${thread.title}`}
    />
    <Button
      size="sm"
      class="h-6 px-2 text-xs"
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
{:else}
  <div
    class={cn(
      'group hover:bg-muted/30 rounded-lg px-2.5 py-2 transition-colors',
      thread.isDeleted && 'opacity-50',
    )}
  >
    <div class="flex items-center gap-2">
      <ThreadStatusIcon class={cn('size-3.5 shrink-0', threadStatusCfg.dotClass)} />
      <span
        class={cn(
          'min-w-0 flex-1 truncate text-sm font-medium',
          thread.isDeleted && 'line-through',
        )}
      >
        {thread.title}
      </span>

      {#if !thread.isDeleted}
        <Select.Root
          type="single"
          value={thread.status}
          onValueChange={(value) => {
            if (value && value !== thread.status) {
              void onUpdateThread?.(thread.id, {
                status: value as ProjectUpdateStatus,
                title: thread.title,
                body: thread.title,
              })
            }
          }}
        >
          <Select.Trigger
            size="sm"
            class={cn(
              'h-5 w-auto shrink-0 gap-0.5 border-none px-1.5 text-[10px] font-medium shadow-none',
              threadStatusCfg.textClass,
            )}
          >
            {projectUpdateStatusLabel(thread.status)}
          </Select.Trigger>
          <Select.Content>
            {#each projectUpdateStatusOptions as opt (opt.value)}
              {@const Icon = opt.icon}
              <Select.Item value={opt.value}>
                <Icon class={cn('size-3', opt.textClass)} />
                {opt.label}
              </Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
      {:else}
        <span class={cn('shrink-0 text-[10px] font-medium', threadStatusCfg.textClass)}>
          {projectUpdateStatusLabel(thread.status)}
        </span>
      {/if}

      <div
        class="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100"
      >
        {#if !thread.isDeleted}
          <button
            type="button"
            class="text-muted-foreground hover:text-foreground rounded p-0.5 transition-colors"
            aria-label={`Edit update ${thread.title}`}
            onclick={() => (editingThread = true)}
          >
            <Pencil class="size-3" />
          </button>
        {/if}
        <button
          type="button"
          class="text-muted-foreground hover:text-destructive rounded p-0.5 transition-colors"
          aria-label={`Delete update ${thread.title}`}
          onclick={handleDeleteThread}
          disabled={thread.isDeleted || deletingThread}
        >
          <Trash2 class="size-3" />
        </button>
      </div>
    </div>

    <div
      class="text-muted-foreground mt-0.5 ml-5.5 flex flex-wrap items-center gap-x-1.5 text-[11px]"
    >
      <span>{thread.createdBy}</span>
      <span class="opacity-40">&middot;</span>
      <span>{formatRelativeTime(thread.createdAt)}</span>
      {#if isProjectUpdateEdited(thread.createdAt, thread.updatedAt, thread.editedAt)}
        <span class="opacity-40">&middot;</span>
        <span>{projectUpdateEditedLabel(thread.createdAt, thread.updatedAt, thread.editedAt)}</span>
      {/if}
      {#if thread.commentCount > 0}
        <span class="opacity-40">&middot;</span>
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

    {#if thread.isDeleted}
      <p class="text-muted-foreground mt-1 ml-5.5 text-xs italic">Deleted</p>
    {/if}

    <div class="ml-5.5">
      <ProjectUpdateThreadReplies
        {thread}
        bind:commentDraft
        bind:showComments
        {creatingComment}
        onCommentKeydown={handleCommentKeydown}
        onCreateComment={handleCreateComment}
        {onUpdateComment}
        {onDeleteComment}
      />
    </div>
  </div>
{/if}
