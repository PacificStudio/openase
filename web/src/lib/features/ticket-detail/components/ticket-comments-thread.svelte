<script lang="ts">
  import {
    ChevronDown,
    ChevronRight,
    ChevronUp,
    Bot,
    MessageSquare,
    Pencil,
    Trash2,
  } from '@lucide/svelte'
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Textarea } from '$ui/textarea'
  import TicketCommentHistoryControl from './ticket-comment-history-control.svelte'
  import TicketDescriptionTimelineItem from './ticket-description-timeline-item.svelte'
  import TicketMarkdownContent from './ticket-markdown-content.svelte'
  import TicketTimelineActivityItem from './ticket-timeline-activity-item.svelte'
  import { getEditedTimelineLabel, isEditedTimelineItem } from './ticket-comments-thread-helpers'
  import TicketTimelineComposer from './ticket-timeline-composer.svelte'
  import { groupDiscussionTimeline, type ActivityGroup } from '../discussion-grouping'
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
  let expandedActivityGroups = $state(new Set<string>())

  const displayItems = $derived(groupDiscussionTimeline(timeline))

  function toggleActivityGroup(groupId: string) {
    const next = new Set(expandedActivityGroups)
    if (next.has(groupId)) {
      next.delete(groupId)
    } else {
      next.add(groupId)
    }
    expandedActivityGroups = next
  }

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

<div>
  <div class="flex flex-col px-4 py-3">
    {#each displayItems as displayItem, index (displayItem.type === 'standalone' ? displayItem.item.id : displayItem.id)}
      {#if displayItem.type === 'activity_group'}
        <!-- Collapsed activity group -->
        {@const group = displayItem as ActivityGroup}
        {@const isExpanded = expandedActivityGroups.has(group.id)}
        <div class="relative pb-3">
          {#if index < displayItems.length - 1}
            <div class="bg-border absolute top-7 bottom-0 left-3 w-px"></div>
          {/if}
          <div class="border-border/50 bg-muted/10 rounded-md border">
            <button
              type="button"
              class="hover:bg-muted/30 flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-[11px] transition-colors"
              onclick={() => toggleActivityGroup(group.id)}
            >
              <ChevronRight
                class={cn(
                  'text-muted-foreground size-3 shrink-0 transition-transform duration-150',
                  isExpanded && 'rotate-90',
                )}
              />
              <Bot
                class={cn('size-3 shrink-0', group.hasFailed ? 'text-red-500' : 'text-blue-500')}
              />
              <span class="text-foreground min-w-0 flex-1 truncate font-medium">
                {group.summary}
              </span>
              {#if group.detail}
                <span class="text-muted-foreground/60 hidden shrink-0 text-[10px] sm:inline"
                  >{group.detail}</span
                >
              {/if}
              <span class="text-muted-foreground shrink-0 text-[10px]">{group.timeRange}</span>
            </button>
            {#if isExpanded}
              <div class="border-border/30 border-t px-1 py-1">
                {#each group.items as actItem (actItem.id)}
                  <div class="flex gap-3 px-2 py-1.5">
                    <TicketTimelineActivityItem item={actItem} />
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        </div>
      {:else}
        {@const item = displayItem.item}
        {#if item.kind === 'description'}
          <TicketDescriptionTimelineItem
            {ticket}
            {item}
            showConnector={index < displayItems.length - 1}
            {savingFields}
            {onSaveFields}
          />
        {:else if item.kind === 'comment'}
          <div class="relative flex gap-3 pb-3">
            {#if index < displayItems.length - 1}
              <div class="bg-border absolute top-7 bottom-0 left-3 w-px"></div>
            {/if}

            <div
              class="bg-background border-border relative z-10 mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full border"
            >
              <MessageSquare class={cn('size-3', 'text-foreground')} />
            </div>
            <div class="min-w-0 flex-1">
              <article class="border-border rounded-lg border">
                <div class="flex items-center justify-between gap-2 px-3 py-1.5">
                  <div class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5 text-xs">
                    <span class="font-medium">{item.actor.name}</span>
                    <span class="text-muted-foreground text-[11px]"
                      >{formatRelativeTime(item.createdAt)}</span
                    >
                    {#if isEditedTimelineItem(item)}
                      <span class="text-muted-foreground text-[10px] italic"
                        >{getEditedTimelineLabel(item)}</span
                      >
                    {/if}
                    {#if item.isDeleted}
                      <Badge variant="outline" class="h-4 px-1.5 py-0 text-[9px]">Deleted</Badge>
                    {/if}
                  </div>
                  <div class="flex shrink-0 items-center">
                    {#if editingCommentId !== item.commentId}
                      <Button
                        size="icon-xs"
                        variant="ghost"
                        class="size-5"
                        aria-label="Edit comment"
                        onclick={() => beginCommentEdit(item)}
                        disabled={Boolean(updatingCommentId || deletingCommentId || item.isDeleted)}
                      >
                        <Pencil class="size-3" />
                      </Button>
                    {/if}
                    <TicketCommentHistoryControl comment={item} onLoad={onLoadCommentHistory} />
                    <Button
                      size="icon-xs"
                      variant="ghost"
                      class="size-5"
                      aria-label="Delete comment"
                      onclick={() => handleDeleteComment(item.commentId)}
                      disabled={deletingCommentId === item.commentId ||
                        updatingCommentId === item.commentId ||
                        item.isDeleted}
                    >
                      <Trash2 class="size-3" />
                    </Button>
                    <Button
                      size="icon-xs"
                      variant="ghost"
                      class="size-5"
                      aria-label={isCommentCollapsed(item.commentId)
                        ? `Expand comment by ${item.actor.name}`
                        : `Collapse comment by ${item.actor.name}`}
                      onclick={() => toggleCommentCollapsed(item.commentId)}
                      disabled={editingCommentId === item.commentId}
                    >
                      {#if isCommentCollapsed(item.commentId)}
                        <ChevronDown class="size-3" />
                      {:else}
                        <ChevronUp class="size-3" />
                      {/if}
                    </Button>
                  </div>
                </div>
                <div class="px-3 py-2">
                  {#if editingCommentId === item.commentId}
                    <div class="space-y-2">
                      <Textarea
                        rows={4}
                        bind:value={editingBody}
                        disabled={updatingCommentId === item.commentId}
                      />
                      <div class="flex justify-end gap-2">
                        <Button
                          size="sm"
                          variant="outline"
                          class="h-6 px-2 text-[11px]"
                          onclick={cancelCommentEdit}
                          disabled={updatingCommentId === item.commentId}
                        >
                          Cancel
                        </Button>
                        <Button
                          size="sm"
                          class="h-6 px-2 text-[11px]"
                          onclick={() => handleSaveCommentEdit(item.commentId)}
                          disabled={!editingBody.trim() || updatingCommentId === item.commentId}
                        >
                          {updatingCommentId === item.commentId ? 'Saving…' : 'Save'}
                        </Button>
                      </div>
                    </div>
                  {:else if isCommentCollapsed(item.commentId)}
                    <p class="text-muted-foreground text-xs italic">
                      {item.isDeleted ? 'Deleted comment collapsed.' : 'Collapsed.'}
                    </p>
                  {:else if item.bodyMarkdown.trim()}
                    <div class={cn('text-sm', item.isDeleted && 'text-muted-foreground')}>
                      <TicketMarkdownContent source={item.bodyMarkdown} />
                    </div>
                  {:else}
                    <p class="text-muted-foreground text-xs italic">No content.</p>
                  {/if}
                </div>
              </article>
            </div>
          </div>
        {/if}
      {/if}
    {/each}

    <div class="relative flex gap-3 pt-1">
      <TicketTimelineComposer creating={creatingComment} onCreate={onCreateComment} />
    </div>
  </div>
</div>
