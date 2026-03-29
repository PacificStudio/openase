<script lang="ts">
  import FileText from '@lucide/svelte/icons/file-text'
  import MessageSquare from '@lucide/svelte/icons/message-square'
  import Pencil from '@lucide/svelte/icons/pencil'
  import Trash2 from '@lucide/svelte/icons/trash-2'
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Textarea } from '$ui/textarea'
  import TicketMarkdownContent from './ticket-markdown-content.svelte'
  import TicketTimelineActivityItem from './ticket-timeline-activity-item.svelte'
  import TicketTimelineComposer from './ticket-timeline-composer.svelte'
  import type { TicketCommentTimelineItem, TicketDetail, TicketTimelineItem } from '../types'

  let {
    ticket,
    timeline,
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
    timeline: TicketTimelineItem[]
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
  let editingCommentId = $state<string | null>(null)
  let editingBody = $state('')

  const itemIcons = {
    description: { icon: FileText, className: 'text-sky-500' },
    comment: { icon: MessageSquare, className: 'text-foreground' },
  } as const

  function beginDescriptionEdit(bodyMarkdown: string) {
    descriptionDraft = bodyMarkdown
    editingDescription = true
  }

  function cancelDescriptionEdit() {
    editingDescription = false
    descriptionDraft = ''
  }

  function handleDescriptionSave() {
    const next = descriptionDraft.trim()
    if (next === ticket.description.trim()) {
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
    if (success) {
      cancelCommentEdit()
    }
  }

  async function handleDeleteComment(commentId: string) {
    if (deletingCommentId === commentId) return
    if (!window.confirm('Delete this comment?')) return

    const success = (await onDeleteComment?.(commentId)) ?? false
    if (success && editingCommentId === commentId) {
      cancelCommentEdit()
    }
  }

  function isEdited(item: TicketTimelineItem) {
    return Boolean(item.editedAt) || item.updatedAt !== item.createdAt
  }

  $effect(() => {
    if (!editingCommentId) return
    if (timeline.some((item) => item.kind === 'comment' && item.commentId === editingCommentId))
      return
    cancelCommentEdit()
  })
</script>

<div class="border-border flex flex-1 flex-col overflow-y-auto border-r">
  <div class="flex flex-col px-6 py-5">
    {#each timeline as item, index (item.id)}
      <div class="relative flex gap-4 pb-6">
        {#if index < timeline.length - 1}
          <div class="bg-border absolute top-10 bottom-0 left-4 w-px"></div>
        {/if}

        {#if item.kind === 'activity'}
          <TicketTimelineActivityItem {item} />
        {:else}
          {@const itemStyle = itemIcons[item.kind]}
          <div
            class="bg-background border-border relative z-10 mt-1 flex size-8 shrink-0 items-center justify-center rounded-full border"
          >
            <itemStyle.icon class={cn('size-4', itemStyle.className)} />
          </div>
          <div class="min-w-0 flex-1">
            <article class="border-border bg-background rounded-xl border shadow-sm">
              <div class="border-border flex items-center justify-between gap-3 border-b px-4 py-3">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2 text-sm">
                    <span class="font-medium">{item.actor.name}</span>
                    <span class="text-muted-foreground">
                      {item.kind === 'description' ? 'opened this ticket' : 'commented'}
                    </span>
                    {#if item.kind === 'description'}
                      <Badge variant="outline" class="h-5 px-2 text-[10px]">
                        {item.identifier ?? ticket.identifier}
                      </Badge>
                    {/if}
                  </div>
                  <div
                    class="text-muted-foreground mt-1 flex flex-wrap items-center gap-2 text-[11px]"
                  >
                    <span>{formatRelativeTime(item.createdAt)}</span>
                    {#if isEdited(item)}
                      <span class="italic">edited</span>
                    {/if}
                    {#if item.kind === 'comment'}
                      <span>rev {item.revisionCount}</span>
                    {/if}
                  </div>
                </div>
                <div class="flex items-center gap-1">
                  {#if item.kind === 'description'}
                    <Badge variant="outline" class="text-[10px]">Description</Badge>
                    {#if !editingDescription}
                      <Button
                        size="icon-xs"
                        variant="ghost"
                        aria-label="Edit description"
                        onclick={() => beginDescriptionEdit(item.bodyMarkdown)}
                        disabled={savingFields}
                      >
                        <Pencil class="size-3.5" />
                      </Button>
                    {/if}
                  {:else}
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
                  {/if}
                </div>
              </div>

              <div class="px-4 py-4">
                {#if item.kind === 'description' && editingDescription}
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
                {:else if item.kind === 'comment' && editingCommentId === item.commentId}
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
                {:else if item.bodyMarkdown.trim()}
                  <div
                    class={cn(item.kind === 'comment' && item.isDeleted && 'text-muted-foreground')}
                  >
                    <TicketMarkdownContent source={item.bodyMarkdown} />
                  </div>
                {:else if item.kind === 'description'}
                  <p class="text-muted-foreground text-sm italic">No description provided.</p>
                {:else}
                  <p class="text-muted-foreground text-sm italic">No comment body.</p>
                {/if}
              </div>
            </article>
          </div>
        {/if}
      </div>
    {/each}

    <div class="relative flex gap-4 pt-1">
      <TicketTimelineComposer creating={creatingComment} onCreate={onCreateComment} />
    </div>
  </div>
</div>
