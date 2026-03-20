<script lang="ts">
  import { cn } from '$lib/utils'
  import { Bot, Ticket, ShieldCheck } from '@lucide/svelte'

  let {
    runningAgents,
    activeTickets,
    pendingApprovals,
    class: className = '',
  }: {
    runningAgents: number
    activeTickets: number
    pendingApprovals: number
    class?: string
  } = $props()
</script>

<div class={cn('border-border bg-card rounded-md border p-4', className)}>
  <h3 class="text-foreground mb-4 text-sm font-medium">Running Now</h3>

  <div class="space-y-4">
    <div class="flex items-center gap-3">
      <div class="flex size-8 items-center justify-center rounded-md bg-emerald-500/10">
        <Bot class="size-4 text-emerald-500" />
      </div>
      <div class="min-w-0 flex-1">
        <span class="text-foreground text-sm">{runningAgents} agents</span>
        <p class="text-muted-foreground text-xs">Currently executing</p>
      </div>
    </div>

    <div class="flex items-center gap-3">
      <div class="flex size-8 items-center justify-center rounded-md bg-blue-500/10">
        <Ticket class="size-4 text-blue-500" />
      </div>
      <div class="min-w-0 flex-1">
        <span class="text-foreground text-sm">{activeTickets} tickets</span>
        <p class="text-muted-foreground text-xs">In progress</p>
      </div>
    </div>

    <div class="flex items-center gap-3">
      <div class="flex size-8 items-center justify-center rounded-md bg-amber-500/10">
        <ShieldCheck class="size-4 text-amber-500" />
      </div>
      <div class="min-w-0 flex-1">
        <span class="text-foreground text-sm">{pendingApprovals} approvals</span>
        <p class="text-muted-foreground text-xs">Awaiting review</p>
      </div>
    </div>
  </div>
</div>
