<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Textarea } from '$ui/textarea'
  import * as Select from '$ui/select'
  import { Pencil, Trash2, X } from '@lucide/svelte'
  import ProjectUpdateThreadBody from './project-update-thread-body.svelte'
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
      draft: { status: ProjectUpdateStatus; body: string },
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
  let editingBody = $state('')
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
    editingBody = thread.bodyMarkdown || thread.title
    if (thread.isDeleted) {
      editingThread = false
      commentDraft = ''
    }
  })

  function cancelThreadEdit() {
    editingThread = false
    editingStatus = thread.status
    editingBody = thread.bodyMarkdown || thread.title
  }

  async function handleSaveThread() {
    const body = editingBody.trim()
    if (!body || savingThread) return

    savingThread = true
    try {
      const success = (await onUpdateThread?.(thread.id, { status: editingStatus, body })) ?? false
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
  <div class="space-y-2 rounded-md px-2.5 py-1.5">
    <div class="flex items-center gap-2">
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
      <div class="ml-auto flex items-center gap-1">
        <Button
          size="sm"
          class="h-6 px-2 text-xs"
          onclick={handleSaveThread}
          disabled={!editingBody.trim() || savingThread}
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
    <Textarea
      bind:value={editingBody}
      class="min-h-7 w-full resize-none border-none bg-transparent px-1 py-1 text-sm shadow-none focus-visible:ring-0"
      aria-label={`Edit update body`}
      rows={2}
    />
  </div>
{:else}
  <div
    class={cn(
      'group hover:bg-muted/30 rounded-lg px-2.5 py-2 transition-colors',
      thread.isDeleted && 'opacity-50',
    )}
  >
    <div class="flex items-start gap-2">
      <ThreadStatusIcon class={cn('mt-0.5 size-3.5 shrink-0', threadStatusCfg.dotClass)} />
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
          <div class="text-muted-foreground flex flex-wrap items-center gap-x-1.5 text-[11px]">
            <span class="max-w-[8rem] truncate font-medium">{thread.createdBy}</span>
            <span class="opacity-40">&middot;</span>
            <span>{formatRelativeTime(thread.createdAt)}</span>
            {#if isProjectUpdateEdited(thread.createdAt, thread.updatedAt, thread.editedAt)}
              <span class="opacity-40">&middot;</span>
              <span
                >{projectUpdateEditedLabel(
                  thread.createdAt,
                  thread.updatedAt,
                  thread.editedAt,
                )}</span
              >
            {/if}
          </div>
          <div class="ml-auto flex shrink-0 items-center gap-1">
            {#if !thread.isDeleted}
              <Select.Root
                type="single"
                value={thread.status}
                onValueChange={(value) => {
                  if (value && value !== thread.status) {
                    void onUpdateThread?.(thread.id, {
                      status: value as ProjectUpdateStatus,
                      body: thread.bodyMarkdown || '',
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
                    <Select.Item value={opt.value}
                      ><Icon class={cn('size-3', opt.textClass)} />{opt.label}</Select.Item
                    >
                  {/each}
                </Select.Content>
              </Select.Root>
            {:else}
              <span class={cn('shrink-0 text-[10px] font-medium', threadStatusCfg.textClass)}
                >{projectUpdateStatusLabel(thread.status)}</span
              >
            {/if}
            <div
              class="flex items-center gap-0.5 sm:opacity-0 sm:transition-opacity sm:group-hover:opacity-100"
            >
              {#if !thread.isDeleted}
                <button
                  type="button"
                  class="text-muted-foreground hover:text-foreground rounded p-0.5 transition-colors"
                  aria-label="Edit update"
                  onclick={() => (editingThread = true)}
                >
                  <Pencil class="size-3" />
                </button>
              {/if}
              <button
                type="button"
                class="text-muted-foreground hover:text-destructive rounded p-0.5 transition-colors"
                aria-label="Delete update"
                onclick={handleDeleteThread}
                disabled={thread.isDeleted || deletingThread}
              >
                <Trash2 class="size-3" />
              </button>
            </div>
          </div>
        </div>
        <ProjectUpdateThreadBody
          bodyMarkdown={thread.bodyMarkdown}
          title={thread.title}
          isDeleted={thread.isDeleted}
        />
        {#if thread.isDeleted}
          <p class="text-muted-foreground mt-1 text-xs italic">Deleted</p>
        {/if}
        {#if thread.commentCount > 0}
          <button
            type="button"
            class="text-muted-foreground hover:text-foreground mt-1 text-[11px] transition-colors"
            onclick={() => (showComments = !showComments)}
          >
            {thread.commentCount}
            {thread.commentCount === 1 ? 'reply' : 'replies'}
          </button>
        {/if}
      </div>
    </div>

    <div class="ml-5">
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
