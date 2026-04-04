<script lang="ts">
  import MessageSquare from '@lucide/svelte/icons/message-square'
  import Pencil from '@lucide/svelte/icons/pencil'
  import Trash2 from '@lucide/svelte/icons/trash-2'
  import { Button } from '$ui/button'
  import { Textarea } from '$ui/textarea'
  import { formatRelativeTime } from '$lib/utils'
  import type { TicketActivity, TicketComment } from '../types'
  import TicketActivityList from './ticket-activity.svelte'
  import TicketMarkdownContent from './ticket-markdown-content.svelte'

  let {
    comments,
    activities,
    creatingComment = false,
    updatingCommentId = null,
    deletingCommentId = null,
    onCreateComment,
    onUpdateComment,
    onDeleteComment,
  }: {
    comments: TicketComment[]
    activities: TicketActivity[]
    creatingComment?: boolean
    updatingCommentId?: string | null
    deletingCommentId?: string | null
    onCreateComment?: (body: string) => Promise<boolean> | boolean
    onUpdateComment?: (commentId: string, body: string) => Promise<boolean> | boolean
    onDeleteComment?: (commentId: string) => Promise<boolean> | boolean
  } = $props()

  let composerBody = $state('')
  let editingCommentId = $state<string | null>(null)
  let editingBody = $state('')

  function beginEdit(comment: TicketComment) {
    editingCommentId = comment.id
    editingBody = comment.body
  }

  function cancelEdit() {
    editingCommentId = null
    editingBody = ''
  }

  async function handleCreateComment() {
    const body = composerBody.trim()
    if (!body || creatingComment) return

    const success = (await onCreateComment?.(body)) ?? false
    if (success) {
      composerBody = ''
    }
  }

  async function handleSaveEdit(commentId: string) {
    const body = editingBody.trim()
    if (!body || updatingCommentId === commentId) return

    const success = (await onUpdateComment?.(commentId, body)) ?? false
    if (success) {
      cancelEdit()
    }
  }

  async function handleDeleteComment(commentId: string) {
    if (deletingCommentId === commentId) return
    if (!window.confirm('Delete this comment?')) return

    const success = (await onDeleteComment?.(commentId)) ?? false
    if (success && editingCommentId === commentId) {
      cancelEdit()
    }
  }

  function formatAuthor(value: string) {
    const normalized = value.trim()
    if (!normalized) return 'Unknown'
    return normalized.includes(':') ? (normalized.split(':').at(-1) ?? normalized) : normalized
  }

  $effect(() => {
    if (!editingCommentId) return
    if (comments.some((comment) => comment.id === editingCommentId)) return
    cancelEdit()
  })
</script>

<div class="flex flex-col gap-6 px-6 py-5">
  <section class="border-border bg-muted/20 rounded-lg border p-4">
    <div class="flex items-start justify-between gap-3">
      <div>
        <h3 class="text-sm font-medium">Discussion</h3>
        <p class="text-muted-foreground mt-1 text-xs">
          Add context, decisions, and handoff notes. Markdown is supported.
        </p>
      </div>
      <MessageSquare class="text-muted-foreground size-4 shrink-0" />
    </div>

    <div class="mt-4 space-y-3">
      <Textarea
        rows={5}
        bind:value={composerBody}
        placeholder="Add a comment with Markdown..."
        disabled={creatingComment}
      />
      <div class="flex justify-end">
        <Button
          size="sm"
          onclick={handleCreateComment}
          disabled={!composerBody.trim() || creatingComment}
        >
          {creatingComment ? 'Posting…' : 'Comment'}
        </Button>
      </div>
    </div>
  </section>

  <section class="space-y-3">
    <div class="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
      Comments
    </div>

    {#if comments.length === 0}
      <div
        class="border-border bg-muted/10 text-muted-foreground rounded-lg border px-4 py-6 text-center text-xs"
      >
        No comments yet
      </div>
    {/if}

    {#each comments as comment}
      <article class="border-border bg-background rounded-lg border">
        <div class="border-border flex items-center justify-between gap-3 border-b px-4 py-3">
          <div class="min-w-0">
            <div class="truncate text-sm font-medium">{formatAuthor(comment.createdBy)}</div>
            <div class="text-muted-foreground mt-1 flex items-center gap-2 text-[11px]">
              <span>{formatRelativeTime(comment.createdAt)}</span>
              {#if comment.updatedAt}
                <span>edited</span>
              {/if}
            </div>
          </div>

          <div class="flex items-center gap-1">
            {#if editingCommentId !== comment.id}
              <Button
                size="icon-xs"
                variant="ghost"
                aria-label="Edit comment"
                onclick={() => beginEdit(comment)}
                disabled={Boolean(updatingCommentId || deletingCommentId)}
              >
                <Pencil class="size-3.5" />
              </Button>
            {/if}
            <Button
              size="icon-xs"
              variant="ghost"
              aria-label="Delete comment"
              onclick={() => handleDeleteComment(comment.id)}
              disabled={deletingCommentId === comment.id || updatingCommentId === comment.id}
            >
              <Trash2 class="size-3.5" />
            </Button>
          </div>
        </div>

        <div class="px-4 py-4">
          {#if editingCommentId === comment.id}
            <div class="space-y-3">
              <Textarea
                rows={6}
                bind:value={editingBody}
                disabled={updatingCommentId === comment.id}
              />
              <div class="flex justify-end gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  onclick={cancelEdit}
                  disabled={updatingCommentId === comment.id}
                >
                  Cancel
                </Button>
                <Button
                  size="sm"
                  onclick={() => handleSaveEdit(comment.id)}
                  disabled={!editingBody.trim() || updatingCommentId === comment.id}
                >
                  {updatingCommentId === comment.id ? 'Saving…' : 'Save'}
                </Button>
              </div>
            </div>
          {:else}
            <TicketMarkdownContent source={comment.body} />
          {/if}
        </div>
      </article>
    {/each}
  </section>

  <TicketActivityList {activities} label="System Activity" emptyText="No system activity yet" />
</div>
