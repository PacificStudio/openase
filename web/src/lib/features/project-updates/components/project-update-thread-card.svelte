<script lang="ts">
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import * as Select from '$ui/select'
  import { AlertTriangle, CircleCheck, CircleX, Pencil, Trash2, X } from '@lucide/svelte'
  import { projectUpdateStatusLabel } from '../status'
  import type { ProjectUpdateStatus, ProjectUpdateThread } from '../types'
  import ProjectUpdateThreadFooter from './project-update-thread-footer.svelte'

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
    textClass: string
  }> = [
    { value: 'on_track', label: 'On track', icon: CircleCheck, textClass: 'text-emerald-600' },
    { value: 'at_risk', label: 'At risk', icon: AlertTriangle, textClass: 'text-amber-600' },
    { value: 'off_track', label: 'Off track', icon: CircleX, textClass: 'text-rose-600' },
  ]

  const currentEditStatusOption = $derived(
    editStatusOptions.find((o) => o.value === editingStatus) ?? editStatusOptions[0],
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
  <div class="border-border rounded-xl border px-3 py-2.5">
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
            'w-auto shrink-0 gap-1 border-none px-2 text-xs font-medium shadow-none',
            currentEditStatusOption.textClass,
          )}
        >
          <CurrentEditStatusIcon class="size-3" />
          {currentEditStatusOption.label}
        </Select.Trigger>
        <Select.Content>
          {#each editStatusOptions as opt (opt.value)}
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
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <span class={cn('text-sm font-medium', thread.isDeleted && 'line-through')}>
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
                <ThreadStatusIcon class={cn('size-3', threadStatusCfg.dotClass)} />
                {projectUpdateStatusLabel(thread.status)}
              </Select.Trigger>
              <Select.Content>
                {#each editStatusOptions as opt (opt.value)}
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
        </div>
        <ProjectUpdateThreadFooter
          {thread}
          {showComments}
          {commentDraft}
          {creatingComment}
          onToggleComments={() => (showComments = !showComments)}
          onCommentDraftChange={(value) => {
            commentDraft = value
          }}
          onCreateComment={handleCreateComment}
          onCommentKeydown={handleCommentKeydown}
          {onUpdateComment}
          {onDeleteComment}
        />
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
  </div>
{/if}
