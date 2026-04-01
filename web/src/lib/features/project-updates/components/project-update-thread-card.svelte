<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Textarea } from '$ui/textarea'
  import { Pencil, Trash2 } from '@lucide/svelte'
  import { isProjectUpdateEdited, projectUpdateEditedLabel } from '../metadata'
  import {
    projectUpdateStatusBadgeClass,
    projectUpdateStatusLabel,
    projectUpdateStatusOptions,
  } from '../status'
  import type { ProjectUpdateStatus, ProjectUpdateThread } from '../types'
  import ProjectUpdateCommentItem from './project-update-comment-item.svelte'
  import ProjectUpdateMarkdownContent from './project-update-markdown-content.svelte'

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
  let editingBody = $state('')
  let commentDraft = $state('')
  let savingThread = $state(false)
  let deletingThread = $state(false)
  let creatingComment = $state(false)

  $effect(() => {
    editingStatus = thread.status
    editingTitle = thread.title
    editingBody = thread.bodyMarkdown
    if (thread.isDeleted) {
      editingThread = false
      commentDraft = ''
    }
  })

  function cancelThreadEdit() {
    editingThread = false
    editingStatus = thread.status
    editingTitle = thread.title
    editingBody = thread.bodyMarkdown
  }

  async function handleSaveThread() {
    const title = editingTitle.trim()
    const body = editingBody.trim()
    if (!title || !body || savingThread) return

    savingThread = true
    try {
      const success =
        (await onUpdateThread?.(thread.id, { status: editingStatus, title, body })) ?? false
      if (success) {
        editingThread = false
      }
    } finally {
      savingThread = false
    }
  }

  async function handleDeleteThread() {
    if (deletingThread || !window.confirm('Delete this update thread?')) return

    deletingThread = true
    try {
      const success = (await onDeleteThread?.(thread.id)) ?? false
      if (success) {
        editingThread = false
      }
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
      }
    } finally {
      creatingComment = false
    }
  }
</script>

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
      <h2 class={cn('text-lg font-semibold', thread.isDeleted && 'text-muted-foreground')}>
        {thread.title}
      </h2>
      <div class="text-muted-foreground mt-1 flex flex-wrap items-center gap-2 text-xs">
        <span>{thread.createdBy}</span>
        <span>{formatRelativeTime(thread.createdAt)}</span>
        {#if isProjectUpdateEdited(thread.createdAt, thread.updatedAt, thread.editedAt)}
          <span
            >{projectUpdateEditedLabel(thread.createdAt, thread.updatedAt, thread.editedAt)}</span
          >
        {/if}
        <span>last activity {formatRelativeTime(thread.lastActivityAt)}</span>
      </div>
    </div>
    <div class="flex items-center gap-2">
      {#if !editingThread}
        <Button
          size="icon-sm"
          variant="ghost"
          aria-label={`Edit update ${thread.title}`}
          onclick={() => (editingThread = true)}
          disabled={thread.isDeleted || deletingThread}
        >
          <Pencil class="size-4" />
        </Button>
      {/if}
      <Button
        size="icon-sm"
        variant="ghost"
        aria-label={`Delete update ${thread.title}`}
        onclick={handleDeleteThread}
        disabled={thread.isDeleted || deletingThread}
      >
        <Trash2 class="size-4" />
      </Button>
    </div>
  </div>

  <div class="space-y-4 px-5 py-4">
    {#if editingThread}
      <div class="space-y-3">
        <div class="grid gap-3 md:grid-cols-[180px_minmax(0,1fr)]">
          <label class="space-y-1.5 text-sm">
            <span class="text-muted-foreground">Delivery status</span>
            <select
              bind:value={editingStatus}
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
            <Input bind:value={editingTitle} aria-label={`Edit title for ${thread.title}`} />
          </label>
        </div>
        <label class="space-y-1.5 text-sm">
          <span class="text-muted-foreground">Body</span>
          <Textarea
            bind:value={editingBody}
            aria-label={`Edit body for ${thread.title}`}
            rows={6}
          />
        </label>
        <div class="flex justify-end gap-2">
          <Button size="sm" variant="outline" onclick={cancelThreadEdit} disabled={savingThread}>
            Cancel
          </Button>
          <Button
            size="sm"
            onclick={handleSaveThread}
            disabled={!editingTitle.trim() || !editingBody.trim() || savingThread}
          >
            {savingThread ? 'Saving…' : 'Save'}
          </Button>
        </div>
      </div>
    {:else if thread.isDeleted}
      <p class="text-muted-foreground text-sm italic">
        This update was deleted. Existing discussion remains visible for timeline continuity.
      </p>
    {:else}
      <ProjectUpdateMarkdownContent source={thread.bodyMarkdown} />
    {/if}
  </div>

  <div class="border-border border-t px-5 py-4">
    <div class="space-y-3">
      {#each thread.comments as comment (comment.id)}
        <ProjectUpdateCommentItem
          threadId={thread.id}
          {comment}
          onUpdate={onUpdateComment}
          onDelete={onDeleteComment}
        />
      {/each}

      {#if !thread.isDeleted}
        <div class="rounded-xl border border-dashed px-4 py-4">
          <label class="space-y-1.5 text-sm">
            <span class="text-muted-foreground">Reply</span>
            <Textarea
              bind:value={commentDraft}
              aria-label={`Reply to ${thread.title}`}
              rows={3}
              placeholder="Add a progress note, question, or follow-up."
            />
          </label>
          <div class="mt-3 flex justify-end">
            <Button
              size="sm"
              onclick={handleCreateComment}
              disabled={!commentDraft.trim() || creatingComment}
            >
              {creatingComment ? 'Posting…' : 'Add comment'}
            </Button>
          </div>
        </div>
      {/if}
    </div>
  </div>
</article>
