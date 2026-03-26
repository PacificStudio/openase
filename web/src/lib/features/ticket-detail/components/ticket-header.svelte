<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Separator } from '$ui/separator'
  import * as Select from '$ui/select'
  import Copy from '@lucide/svelte/icons/copy'
  import Check from '@lucide/svelte/icons/check'
  import Pencil from '@lucide/svelte/icons/pencil'
  import Save from '@lucide/svelte/icons/save'
  import X from '@lucide/svelte/icons/x'
  import { cn } from '$lib/utils'
  import type { TicketDetail, TicketStatusOption } from '../types'

  let {
    ticket,
    statuses,
    savingFields = false,
    onClose,
    onSaveFields,
  }: {
    ticket: TicketDetail
    statuses: TicketStatusOption[]
    savingFields?: boolean
    onClose?: () => void
    onSaveFields?: (draft: { title: string; description: string; statusId: string }) => void
  } = $props()

  let copied = $state(false)
  let titleEditOpen = $state(false)
  let titleDraft = $state('')

  const priorityColors: Record<string, string> = {
    urgent: 'bg-red-500/15 text-red-400 border-red-500/20',
    high: 'bg-orange-500/15 text-orange-400 border-orange-500/20',
    medium: 'bg-yellow-500/15 text-yellow-400 border-yellow-500/20',
    low: 'bg-blue-500/15 text-blue-400 border-blue-500/20',
  }

  const typeLabels: Record<string, string> = {
    feature: 'Feature',
    bugfix: 'Bug Fix',
    refactor: 'Refactor',
    chore: 'Chore',
  }

  function copyIdentifier() {
    navigator.clipboard.writeText(ticket.identifier)
    copied = true
    setTimeout(() => (copied = false), 1500)
  }

  const titleDirty = $derived(titleDraft.trim() !== ticket.title)

  $effect(() => {
    if (!titleEditOpen) {
      titleDraft = ticket.title
    }
  })

  function toggleTitleEdit() {
    if (titleEditOpen) {
      handleTitleSave()
      return
    }
    titleDraft = ticket.title
    titleEditOpen = true
  }

  function handleTitleSave() {
    const nextTitle = titleDraft.trim()
    if (!nextTitle) {
      titleDraft = ticket.title
      titleEditOpen = false
      return
    }
    if (nextTitle === ticket.title) {
      titleEditOpen = false
      return
    }
    onSaveFields?.({
      title: nextTitle,
      description: ticket.description,
      statusId: ticket.status.id,
    })
    titleEditOpen = false
  }

  function handleStatusChange(nextStatusId: string) {
    if (!nextStatusId || nextStatusId === ticket.status.id) {
      return
    }
    onSaveFields?.({
      title: ticket.title,
      description: ticket.description,
      statusId: nextStatusId,
    })
  }

  function cancelTitleEdit() {
    titleDraft = ticket.title
    titleEditOpen = false
  }

  function handleTitleKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter') {
      event.preventDefault()
      handleTitleSave()
    }
    if (event.key === 'Escape') {
      cancelTitleEdit()
    }
  }
</script>

<div class="flex flex-col gap-4 px-6 pt-6 pb-4">
  <div class="flex items-center justify-between">
    <div class="flex flex-wrap items-center gap-2">
      <button
        onclick={copyIdentifier}
        class="text-muted-foreground hover:bg-muted flex items-center gap-1.5 rounded px-1.5 py-0.5 font-mono text-xs transition-colors"
      >
        {ticket.identifier}
        {#if copied}
          <Check class="size-3 text-green-400" />
        {:else}
          <Copy class="size-3" />
        {/if}
      </button>
      <Badge class={cn('text-[10px] uppercase', priorityColors[ticket.priority])}>
        {ticket.priority}
      </Badge>
      <Badge variant="outline" class="text-[10px]">
        {typeLabels[ticket.type] ?? ticket.type}
      </Badge>
    </div>
    <div class="flex items-center gap-1">
      <Button variant="ghost" size="icon-sm" onclick={onClose}>
        <X class="size-3.5" />
      </Button>
    </div>
  </div>

  <div class="flex items-start justify-between gap-3">
    <div class="min-w-0 flex-1">
      {#if titleEditOpen}
        <div class="flex items-center gap-2">
          <Input
            bind:value={titleDraft}
            class="h-10 text-sm font-medium"
            disabled={savingFields}
            onkeydown={handleTitleKeydown}
          />
          <Button variant="outline" size="sm" onclick={cancelTitleEdit} disabled={savingFields}>
            Cancel
          </Button>
          <Button size="sm" onclick={handleTitleSave} disabled={savingFields || !titleDirty}>
            <Save class="size-3.5" />
            {savingFields ? 'Saving…' : 'Save'}
          </Button>
        </div>
      {:else}
        <div class="flex items-center gap-2">
          <h2 class="min-w-0 flex-1 text-sm leading-snug font-medium">{ticket.title}</h2>
          <Button variant="ghost" size="icon-sm" onclick={toggleTitleEdit} aria-label="Edit title">
            <Pencil class="size-3.5" />
          </Button>
        </div>
      {/if}
    </div>
  </div>

  <div class="flex flex-wrap items-center gap-2">
    <Select.Root type="single" value={ticket.status.id} onValueChange={handleStatusChange}>
      <Select.Trigger
        class="h-7 rounded-full border px-3 py-0 text-xs font-medium shadow-none"
        disabled={savingFields}
        style="background-color: {ticket.status.color}20; color: {ticket.status
          .color}; border-color: {ticket.status.color}30"
      >
        {savingFields ? 'Saving…' : ticket.status.name}
      </Select.Trigger>
      <Select.Content>
        {#each statuses as status (status.id)}
          <Select.Item value={status.id}>{status.name}</Select.Item>
        {/each}
      </Select.Content>
    </Select.Root>
  </div>
</div>
<Separator />
