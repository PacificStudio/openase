<script lang="ts">
  import MessageSquare from '@lucide/svelte/icons/message-square'
  import Pencil from '@lucide/svelte/icons/pencil'
  import Trash2 from '@lucide/svelte/icons/trash-2'
  import { Button } from '$ui/button'
  import { Textarea } from '$ui/textarea'
  import { formatRelativeTime } from '$lib/utils'
  import TicketActivityList from './ticket-activity.svelte'
  import TicketMarkdown from './ticket-markdown.svelte'
  import type { TicketActivity, TicketComment } from '../types'

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
    onCreateComment?: (draft: { body: string }) => void | Promise<void>
    onUpdateComment?: (commentId: string, draft: { body: string }) => void | Promise<void>
    onDeleteComment?: (commentId: string) => void | Promise<void>
  } = $props()

  let newCommentBody = $state('')
  let editingCommentId = $state<string | null>(null)
  let editingBody = $state('')

  async function submitNewComment() {
    if (newCommentBody.trim() === '') return
    await onCreateComment?.({ body: newCommentBody })
    newCommentBody = ''
  }

  function beginEdit(comment: TicketComment) {
    editingCommentId = comment.id
    editingBody = comment.body
  }

  function cancelEdit() {
    editingCommentId = null
    editingBody = ''
  }

  async function saveEdit(commentId: string) {
    if (editingBody.trim() === '') return
    await onUpdateComment?.(commentId, { body: editingBody })
    cancelEdit()
  }

  function wasEdited(comment: TicketComment) {
    return comment.updatedAt !== comment.createdAt
  }
</script>

<section class="space-y-6 px-5 py-4">
  <div class="space-y-3">
    <div class="flex items-center gap-2">
      <MessageSquare class="text-muted-foreground size-4" />
      <h3 class="text-sm font-medium">Discussion</h3>
    </div>

    {#if comments.length === 0}
      <div class="border-border bg-muted/20 rounded-lg border border-dashed px-4 py-6 text-center">
        <p class="text-sm font-medium">No comments yet</p>
        <p class="text-muted-foreground mt-1 text-xs">
          Leave review notes, handoff context, or human decisions here.
        </p>
      </div>
    {:else}
      <div class="space-y-3">
        {#each comments as comment (comment.id)}
          <article class="border-border bg-background rounded-xl border p-4">
            <div class="mb-3 flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="text-sm font-medium">{comment.createdBy}</div>
                <div class="text-muted-foreground flex items-center gap-2 text-xs">
                  <span>{formatRelativeTime(comment.createdAt)}</span>
                  {#if wasEdited(comment)}
                    <span>· edited</span>
                  {/if}
                </div>
              </div>
              <div class="flex items-center gap-1">
                <Button
                  variant="ghost"
                  size="icon-xs"
                  disabled={updatingCommentId === comment.id || deletingCommentId === comment.id}
                  onclick={() => beginEdit(comment)}
                >
                  <Pencil class="size-3.5" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  disabled={updatingCommentId === comment.id || deletingCommentId === comment.id}
                  onclick={() => onDeleteComment?.(comment.id)}
                >
                  <Trash2 class="size-3.5" />
                </Button>
              </div>
            </div>

            {#if editingCommentId === comment.id}
              <div class="space-y-3">
                <Textarea
                  rows={6}
                  bind:value={editingBody}
                  disabled={updatingCommentId === comment.id}
                />
                <div class="flex items-center justify-end gap-2">
                  <Button variant="ghost" size="sm" onclick={cancelEdit}>Cancel</Button>
                  <Button
                    size="sm"
                    disabled={editingBody.trim() === '' || updatingCommentId === comment.id}
                    onclick={() => saveEdit(comment.id)}
                  >
                    {updatingCommentId === comment.id ? 'Saving…' : 'Save'}
                  </Button>
                </div>
              </div>
            {:else}
              <TicketMarkdown body={comment.body} />
            {/if}
          </article>
        {/each}
      </div>
    {/if}

    <div class="border-border bg-muted/20 rounded-xl border p-4">
      <div class="mb-2 text-sm font-medium">Add comment</div>
      <Textarea
        rows={6}
        bind:value={newCommentBody}
        disabled={creatingComment}
        placeholder="Share review notes, decisions, or handoff context using Markdown."
      />
      <div class="mt-3 flex justify-end">
        <Button
          disabled={newCommentBody.trim() === '' || creatingComment}
          onclick={submitNewComment}
        >
          {creatingComment ? 'Posting…' : 'Comment'}
        </Button>
      </div>
    </div>
  </div>

  <TicketActivityList {activities} title="System activity" emptyLabel="No system activity yet" />
</section>
