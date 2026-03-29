<script lang="ts">
  import FileText from '@lucide/svelte/icons/file-text'
  import Pencil from '@lucide/svelte/icons/pencil'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Textarea } from '$ui/textarea'
  import { cn, formatRelativeTime } from '$lib/utils'
  import TicketMarkdownContent from './ticket-markdown-content.svelte'
  import type { TicketDescriptionTimelineItem, TicketDetail } from '../types'

  let {
    ticket,
    item,
    showConnector = false,
    savingFields = false,
    onSaveFields,
  }: {
    ticket: TicketDetail
    item: TicketDescriptionTimelineItem
    showConnector?: boolean
    savingFields?: boolean
    onSaveFields?: (draft: { title: string; description: string; statusId: string }) => void
  } = $props()

  let editing = $state(false)
  let draft = $state('')

  function beginEdit() {
    draft = item.bodyMarkdown
    editing = true
  }

  function cancelEdit() {
    editing = false
    draft = ''
  }

  function handleSave() {
    const next = draft.trim()
    if (next === ticket.description.trim()) {
      editing = false
      return
    }

    onSaveFields?.({
      title: ticket.title,
      description: next,
      statusId: ticket.status.id,
    })
    editing = false
  }
</script>

<div class="relative flex gap-4 pb-6">
  {#if showConnector}
    <div class="bg-border absolute top-10 bottom-0 left-4 w-px"></div>
  {/if}
  <div
    class="bg-background border-border relative z-10 mt-1 flex size-8 shrink-0 items-center justify-center rounded-full border"
  >
    <FileText class={cn('size-4', 'text-sky-500')} />
  </div>
  <div class="min-w-0 flex-1">
    <article class="border-border bg-background rounded-xl border shadow-sm">
      <div class="border-border flex items-center justify-between gap-3 border-b px-4 py-3">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2 text-sm">
            <span class="font-medium">{item.actor.name}</span>
            <span class="text-muted-foreground">opened this ticket</span>
            <Badge variant="outline" class="h-5 px-2 text-[10px]">
              {item.identifier ?? ticket.identifier}
            </Badge>
          </div>
          <div class="text-muted-foreground mt-1 flex flex-wrap items-center gap-2 text-[11px]">
            <span>{formatRelativeTime(item.createdAt)}</span>
          </div>
        </div>
        <div class="flex items-center gap-1">
          <Badge variant="outline" class="text-[10px]">Description</Badge>
          {#if !editing}
            <Button
              size="icon-xs"
              variant="ghost"
              aria-label="Edit description"
              onclick={beginEdit}
              disabled={savingFields}
            >
              <Pencil class="size-3.5" />
            </Button>
          {/if}
        </div>
      </div>

      <div class="px-4 py-4">
        {#if editing}
          <div class="space-y-3">
            <Textarea rows={8} bind:value={draft} disabled={savingFields} />
            <div class="flex justify-end gap-2">
              <Button size="sm" variant="outline" onclick={cancelEdit} disabled={savingFields}>
                Cancel
              </Button>
              <Button size="sm" onclick={handleSave} disabled={savingFields}>
                {savingFields ? 'Saving…' : 'Save'}
              </Button>
            </div>
          </div>
        {:else if item.bodyMarkdown.trim()}
          <TicketMarkdownContent source={item.bodyMarkdown} />
        {:else}
          <p class="text-muted-foreground text-sm italic">No description provided.</p>
        {/if}
      </div>
    </article>
  </div>
</div>
