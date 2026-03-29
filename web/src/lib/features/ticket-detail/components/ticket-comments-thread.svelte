<script lang="ts">
  import ChevronDown from '@lucide/svelte/icons/chevron-down'
  import ChevronUp from '@lucide/svelte/icons/chevron-up'
  import MessageSquare from '@lucide/svelte/icons/message-square'
  import Pencil from '@lucide/svelte/icons/pencil'
  import Trash2 from '@lucide/svelte/icons/trash-2'
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Textarea } from '$ui/textarea'
  import TicketCommentHistoryControl from './ticket-comment-history-control.svelte'
  import TicketDescriptionTimelineItem from './ticket-description-timeline-item.svelte'
  import TicketMarkdownContent from './ticket-markdown-content.svelte'
  import TicketTimelineActivityItem from './ticket-timeline-activity-item.svelte'
  import TicketTimelineComposer from './ticket-timeline-composer.svelte'
  import type {
    TicketCommentRevision,
    TicketCommentTimelineItem,
    TicketDetail,
    TicketTimelineItem,
  } from '../types'

  let {
    ticket,
    timeline,
    creatingComment = false,
    updatingCommentId = null,
    deletingCommentId = null,
    savingFields = false,
    onSaveFields,
    onCreateComment,
    onUpdateComment,
    onDeleteComment,
    onLoadCommentHistory,
  }: {
    ticket: TicketDetail
    timeline: TicketTimelineItem[]
    creatingComment?: boolean
    updatingCommentId?: string | null
    deletingCommentId?: string | null
    savingFields?: boolean
    onSaveFields?: (draft: { title: string; description: string; statusId: string }) => void
    onCreateComment?: (body: string) => Promise<boolean> | boolean
    onUpdateComment?: (commentId: string, body: string) => Promise<boolean> | boolean
    onDeleteComment?: (commentId: string) => Promise<boolean> | boolean
    onLoadCommentHistory?: (
      commentId: string,
    ) => Promise<TicketCommentRevision[]> | TicketCommentRevision[]
  } = $props()

  let editingCommentId = $state<string | null>(null)
  let editingBody = $state('')
  let collapsedCommentIds = $state<Record<string, boolean>>({})

  function beginCommentEdit(comment: TicketCommentTimelineItem) {
    editingCommentId = comment.commentId
    editingBody = comment.bodyMarkdown
  }

  function cancelCommentEdit() {
    editingCommentId = null
    editingBody = ''
  }

  async function handleSaveCommentEdit(commentId: string) {
    const body = editingBody.trim()
    if (!body || updatingCommentId === commentId) return

    const success = (await onUpdateComment?.(commentId, body)) ?? false
    if (success) cancelCommentEdit()
  }

  async function handleDeleteComment(commentId: string) {
    if (deletingCommentId === commentId || !window.confirm('Delete this comment?')) return

    const success = (await onDeleteComment?.(commentId)) ?? false
    if (success && editingCommentId === commentId) cancelCommentEdit()
  }

  function isEdited(item: TicketTimelineItem) {
    return Boolean(item.editedAt) || item.updatedAt !== item.createdAt
  }

  function editedLabel(item: TicketTimelineItem) {
    const editedAt = item.editedAt ?? item.updatedAt
    return editedAt && isEdited(item) ? `edited ${formatRelativeTime(editedAt)}` : null
  }

  function isCommentCollapsed(commentId: string) {
    return Boolean(collapsedCommentIds[commentId])
  }

  function toggleCommentCollapsed(commentId: string) {
    collapsedCommentIds = { ...collapsedCommentIds, [commentId]: !collapsedCommentIds[commentId] }
  }

  $effect(() => {
    if (!editingCommentId) return
    const editingComment = timeline.find(
      (item): item is TicketCommentTimelineItem =>
        item.kind === 'comment' && item.commentId === editingCommentId,
    )
    if (!editingComment || editingComment.isDeleted) cancelCommentEdit()
  })
</script>

<div class="border-border flex flex-1 flex-col overflow-y-auto border-r">
  <div class="flex flex-col px-6 py-5">
    {#each timeline as item, index (item.id)}
      {#if item.kind === 'activity'}
        <div class="relative flex gap-4 pb-6">
          {#if index < timeline.length - 1}
            <div class="bg-border absolute top-10 bottom-0 left-4 w-px"></div>
          {/if}
          <TicketTimelineActivityItem {item} />
        </div>
      {:else if item.kind === 'description'}
        <TicketDescriptionTimelineItem
          {ticket}
          {item}
          showConnector={index < timeline.length - 1}
          {savingFields}
          {onSaveFields}
        />
      {:else}
        <div class="relative flex gap-4 pb-6">
          {#if index < timeline.length - 1}
            <div class="bg-border absolute top-10 bottom-0 left-4 w-px"></div>
          {/if}

          <div
            class="bg-background border-border relative z-10 mt-1 flex size-8 shrink-0 items-center justify-center rounded-full border"
          >
            <MessageSquare class={cn('size-4', 'text-foreground')} />
          </div>
          <div class="min-w-0 flex-1">
            <article class="border-border bg-background rounded-xl border shadow-sm">
              <div class="border-border flex items-center justify-between gap-3 border-b px-4 py-3">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2 text-sm">
                    <span class="font-medium">{item.actor.name}</span>
                    <span class="text-muted-foreground">commented</span>
                  </div>
                  <div
                    class="text-muted-foreground mt-1 flex flex-wrap items-center gap-2 text-[11px]"
                  >
                    <span>{formatRelativeTime(item.createdAt)}</span>
                    {#if isEdited(item)}
                      <span class="italic">{editedLabel(item)}</span>
                    {/if}
                    <span>rev {item.revisionCount}</span>
                    {#if item.isDeleted}
                      <Badge variant="outline" class="h-5 px-2 text-[10px]">Deleted</Badge>
                    {/if}
                  </div>
                </div>

                <div class="flex items-center gap-1">
                  {#if editingCommentId !== item.commentId}
                    <Button
                      size="icon-xs"
                      variant="ghost"
                      aria-label="Edit comment"
                      onclick={() => beginCommentEdit(item)}
                      disabled={Boolean(updatingCommentId || deletingCommentId || item.isDeleted)}
                    >
                      <Pencil class="size-3.5" />
                    </Button>
                  {/if}
                  <TicketCommentHistoryControl comment={item} onLoad={onLoadCommentHistory} />
                  <Button
                    size="icon-xs"
                    variant="ghost"
                    aria-label="Delete comment"
                    onclick={() => handleDeleteComment(item.commentId)}
                    disabled={deletingCommentId === item.commentId ||
                      updatingCommentId === item.commentId ||
                      item.isDeleted}
                  >
                    <Trash2 class="size-3.5" />
                  </Button>
                  <Button
                    size="icon-xs"
                    variant="ghost"
                    aria-label={isCommentCollapsed(item.commentId)
                      ? `Expand comment by ${item.actor.name}`
                      : `Collapse comment by ${item.actor.name}`}
                    onclick={() => toggleCommentCollapsed(item.commentId)}
                    disabled={editingCommentId === item.commentId}
                  >
                    {#if isCommentCollapsed(item.commentId)}
                      <ChevronDown class="size-3.5" />
                    {:else}
                      <ChevronUp class="size-3.5" />
                    {/if}
                  </Button>
                </div>
              </div>

              <div class="px-4 py-4">
                {#if editingCommentId === item.commentId}
                  <div class="space-y-3">
                    <Textarea
                      rows={6}
                      bind:value={editingBody}
                      disabled={updatingCommentId === item.commentId}
                    />
                    <div class="flex justify-end gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onclick={cancelCommentEdit}
                        disabled={updatingCommentId === item.commentId}
                      >
                        Cancel
                      </Button>
                      <Button
                        size="sm"
                        onclick={() => handleSaveCommentEdit(item.commentId)}
                        disabled={!editingBody.trim() || updatingCommentId === item.commentId}
                      >
                        {updatingCommentId === item.commentId ? 'Saving…' : 'Save'}
                      </Button>
                    </div>
                  </div>
                {:else if isCommentCollapsed(item.commentId)}
                  <p class="text-muted-foreground text-sm italic">
                    {item.isDeleted ? 'Deleted comment collapsed.' : 'Comment collapsed.'}
                  </p>
                {:else if item.bodyMarkdown.trim()}
                  <div class={cn(item.isDeleted && 'text-muted-foreground')}>
                    <TicketMarkdownContent source={item.bodyMarkdown} />
                  </div>
                {:else}
                  <p class="text-muted-foreground text-sm italic">No comment body.</p>
                {/if}
              </div>
            </article>
          </div>
        </div>
      {/if}
    {/each}

    <div class="relative flex gap-4 pt-1">
      <TicketTimelineComposer creating={creatingComment} onCreate={onCreateComment} />
    </div>
  </div>
</div>
