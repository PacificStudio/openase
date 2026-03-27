<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Textarea } from '$ui/textarea'
  import MessageSquare from '@lucide/svelte/icons/message-square'
  import Pencil from '@lucide/svelte/icons/pencil'
  import Trash2 from '@lucide/svelte/icons/trash-2'
  import { formatRelativeTime } from '$lib/utils'
  import TicketActivityList from './ticket-activity.svelte'
  import TicketMarkdownContent from './ticket-markdown-content.svelte'
  import type { TicketActivity, TicketComment, TicketDetail } from '../types'

  let {
    ticket,
    comments,
    activities,
    savingFields = false,
    creatingComment = false,
    updatingCommentId = null,
    deletingCommentId = null,
    onSaveFields,
    onCreateComment,
    onUpdateComment,
    onDeleteComment,
  }: {
    ticket: TicketDetail
    comments: TicketComment[]
    activities: TicketActivity[]
    savingFields?: boolean
    creatingComment?: boolean
    updatingCommentId?: string | null
    deletingCommentId?: string | null
    onSaveFields?: (draft: { title: string; description: string; statusId: string }) => void
    onCreateComment?: (body: string) => Promise<boolean> | boolean
    onUpdateComment?: (commentId: string, body: string) => Promise<boolean> | boolean
    onDeleteComment?: (commentId: string) => Promise<boolean> | boolean
  } = $props()

  let editingDescription = $state(false)
  let descriptionDraft = $state('')
  let composerOpen = $state(false)
  let composerBody = $state('')
  let editingCommentId = $state<string | null>(null)
  let editingBody = $state('')

  function beginDescriptionEdit() {
    descriptionDraft = ticket.description
    editingDescription = true
  }

  function cancelDescriptionEdit() {
    editingDescription = false
    descriptionDraft = ''
  }

  function handleDescriptionSave() {
    const next = descriptionDraft.trim()
    if (next === ticket.description) {
      editingDescription = false
      return
    }
    onSaveFields?.({
      title: ticket.title,
      description: next,
      statusId: ticket.status.id,
    })
    editingDescription = false
  }

  function beginCommentEdit(comment: TicketComment) {
    editingCommentId = comment.id
    editingBody = comment.body
  }

  function cancelCommentEdit() {
    editingCommentId = null
    editingBody = ''
  }

  async function handleCreateComment() {
    const body = composerBody.trim()
    if (!body || creatingComment) return
    const success = (await onCreateComment?.(body)) ?? false
    if (success) {
      composerBody = ''
      composerOpen = false
    }
  }

  async function handleSaveCommentEdit(commentId: string) {
    const body = editingBody.trim()
    if (!body || updatingCommentId === commentId) return
    const success = (await onUpdateComment?.(commentId, body)) ?? false
    if (success) cancelCommentEdit()
  }

  async function handleDeleteComment(commentId: string) {
    if (deletingCommentId === commentId) return
    if (!window.confirm('Delete this comment?')) return
    const success = (await onDeleteComment?.(commentId)) ?? false
    if (success && editingCommentId === commentId) cancelCommentEdit()
  }

  function formatAuthor(value: string) {
    const normalized = value.trim()
    if (!normalized) return 'Unknown'
    return normalized.includes(':') ? (normalized.split(':').at(-1) ?? normalized) : normalized
  }

  $effect(() => {
    if (!editingCommentId) return
    if (comments.some((comment) => comment.id === editingCommentId)) return
    cancelCommentEdit()
  })
</script>

<div class="border-border flex flex-1 flex-col overflow-y-auto border-r">
  <div class="flex flex-col gap-4 px-6 py-5">
    <article class="border-border bg-background rounded-lg border">
      <div class="border-border flex items-center justify-between gap-3 border-b px-4 py-3">
        <div class="min-w-0">
          <div class="truncate text-sm font-medium">{formatAuthor(ticket.createdBy)}</div>
          <div class="text-muted-foreground mt-0.5 text-[11px]">
            opened {formatRelativeTime(ticket.createdAt)}
          </div>
        </div>
        <div class="flex items-center gap-1">
          <Badge variant="outline" class="text-[10px]">Description</Badge>
          {#if !editingDescription}
            <Button
              size="icon-xs"
              variant="ghost"
              aria-label="Edit description"
              onclick={beginDescriptionEdit}
              disabled={savingFields}
            >
              <Pencil class="size-3.5" />
            </Button>
          {/if}
        </div>
      </div>
      <div class="px-4 py-4">
        {#if editingDescription}
          <div class="space-y-3">
            <Textarea rows={8} bind:value={descriptionDraft} disabled={savingFields} />
            <div class="flex justify-end gap-2">
              <Button
                size="sm"
                variant="outline"
                onclick={cancelDescriptionEdit}
                disabled={savingFields}
              >
                Cancel
              </Button>
              <Button size="sm" onclick={handleDescriptionSave} disabled={savingFields}>
                {savingFields ? 'Saving…' : 'Save'}
              </Button>
            </div>
          </div>
        {:else if ticket.description.trim()}
          <TicketMarkdownContent source={ticket.description} />
        {:else}
          <p class="text-muted-foreground text-sm italic">No description provided.</p>
        {/if}
      </div>
    </article>

    {#each comments as comment (comment.id)}
      <article class="border-border bg-background rounded-lg border">
        <div class="border-border flex items-center justify-between gap-3 border-b px-4 py-3">
          <div class="min-w-0">
            <div class="truncate text-sm font-medium">{formatAuthor(comment.createdBy)}</div>
            <div class="text-muted-foreground mt-0.5 flex items-center gap-2 text-[11px]">
              <span>{formatRelativeTime(comment.createdAt)}</span>
              {#if comment.updatedAt}
                <span class="italic">edited</span>
              {/if}
            </div>
          </div>
          <div class="flex items-center gap-1">
            {#if editingCommentId !== comment.id}
              <Button
                size="icon-xs"
                variant="ghost"
                aria-label="Edit comment"
                onclick={() => beginCommentEdit(comment)}
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
                  onclick={cancelCommentEdit}
                  disabled={updatingCommentId === comment.id}
                >
                  Cancel
                </Button>
                <Button
                  size="sm"
                  onclick={() => handleSaveCommentEdit(comment.id)}
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

    <div class="border-border bg-muted/10 rounded-lg border p-4">
      {#if composerOpen}
        <div class="mb-3 flex items-center gap-2">
          <MessageSquare class="text-muted-foreground size-4" />
          <span class="text-sm font-medium">Add a comment</span>
        </div>
        <Textarea
          rows={4}
          bind:value={composerBody}
          placeholder="Leave a comment (Markdown supported)…"
          disabled={creatingComment}
        />
        <div class="mt-3 flex justify-end gap-2">
          <Button
            size="sm"
            variant="outline"
            onclick={() => {
              composerOpen = false
              composerBody = ''
            }}
            disabled={creatingComment}
          >
            Cancel
          </Button>
          <Button
            size="sm"
            onclick={handleCreateComment}
            disabled={!composerBody.trim() || creatingComment}
          >
            {creatingComment ? 'Posting…' : 'Comment'}
          </Button>
        </div>
      {:else}
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-2">
            <MessageSquare class="text-muted-foreground size-4" />
            <span class="text-sm font-medium">Comment on this ticket</span>
          </div>
          <Button
            size="sm"
            variant="outline"
            onclick={() => {
              composerOpen = true
            }}
          >
            Add comment
          </Button>
        </div>
      {/if}
    </div>

    {#if activities.length > 0}
      <TicketActivityList {activities} label="Activity" emptyText="" />
    {/if}
  </div>
</div>
