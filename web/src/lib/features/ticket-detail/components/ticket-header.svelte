<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Separator } from '$ui/separator'
  import Copy from '@lucide/svelte/icons/copy'
  import Check from '@lucide/svelte/icons/check'
  import X from '@lucide/svelte/icons/x'
  import { cn } from '$lib/utils'
  import type { TicketDetail } from '../types'

  let {
    ticket,
    onClose,
  }: {
    ticket: TicketDetail
    onClose?: () => void
  } = $props()

  let copied = $state(false)

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
</script>

<div class="flex flex-col gap-3 px-5 pt-5 pb-3">
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <button
        onclick={copyIdentifier}
        class="flex items-center gap-1.5 rounded px-1.5 py-0.5 font-mono text-xs text-muted-foreground hover:bg-muted transition-colors"
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

  <h2 class="text-sm font-medium leading-snug">{ticket.title}</h2>

  <div class="flex items-center gap-2">
    <Badge
      class="text-[10px]"
      style="background-color: {ticket.status.color}20; color: {ticket.status.color}; border-color: {ticket.status.color}30"
    >
      {ticket.status.name}
    </Badge>
  </div>
</div>
<Separator />
