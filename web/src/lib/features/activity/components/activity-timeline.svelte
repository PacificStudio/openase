<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import type { ActivityEntry } from '../types'

  let {
    entries,
    class: className = '',
  }: {
    entries: ActivityEntry[]
    class?: string
  } = $props()

  const dotColors: Record<string, string> = {
    ticket_created: 'bg-blue-500',
    agent_started: 'bg-emerald-500',
    agent_completed: 'bg-emerald-400',
    hook_failed: 'bg-red-500',
    pr_opened: 'bg-purple-500',
    pr_merged: 'bg-violet-500',
    comment_added: 'bg-amber-500',
    status_changed: 'bg-sky-500',
    agent_stalled: 'bg-yellow-500',
    budget_alert: 'bg-orange-500',
  }

  function getDotColor(eventType: string): string {
    return dotColors[eventType] ?? 'bg-muted-foreground'
  }

  function getDateLabel(timestamp: string): string {
    const date = new Date(timestamp)
    const now = new Date()
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
    const target = new Date(date.getFullYear(), date.getMonth(), date.getDate())
    const diff = today.getTime() - target.getTime()
    const days = Math.floor(diff / 86_400_000)

    if (days === 0) return 'Today'
    if (days === 1) return 'Yesterday'
    if (days < 7) return `${days} days ago`
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
    })
  }

  type GroupedEntries = { label: string; entries: ActivityEntry[] }[]

  function groupByDate(items: ActivityEntry[]): GroupedEntries {
    const groups = new Map<string, ActivityEntry[]>()
    for (const item of items) {
      const label = getDateLabel(item.timestamp)
      if (!groups.has(label)) groups.set(label, [])
      groups.get(label)!.push(item)
    }
    return Array.from(groups.entries()).map(([label, entries]) => ({
      label,
      entries,
    }))
  }

  const grouped = $derived(groupByDate(entries))
</script>

<div class={cn('space-y-6', className)}>
  {#each grouped as group (group.label)}
    <div>
      <h3 class="text-muted-foreground mb-3 text-xs font-medium tracking-wider uppercase">
        {group.label}
      </h3>
      <div class="relative space-y-0">
        {#each group.entries as entry, i (entry.id)}
          <div class="group relative flex gap-3 pb-4">
            <div class="flex flex-col items-center">
              <span
                class={cn(
                  'ring-background mt-1.5 size-2.5 shrink-0 rounded-full ring-4',
                  getDotColor(entry.eventType),
                )}
              ></span>
              {#if i < group.entries.length - 1}
                <span class="bg-border mt-1 h-full w-px"></span>
              {/if}
            </div>
            <div class="min-w-0 flex-1 pb-1">
              <p class="text-foreground text-sm leading-snug">
                {entry.message}
              </p>
              <div class="text-muted-foreground mt-1 flex flex-wrap items-center gap-2 text-xs">
                {#if entry.agentName}
                  <span class="text-muted-foreground/80 font-mono">{entry.agentName}</span>
                  <span>&middot;</span>
                {/if}
                {#if entry.ticketIdentifier}
                  <a
                    href="/tickets/{entry.ticketIdentifier}"
                    class="text-primary font-mono hover:underline"
                  >
                    {entry.ticketIdentifier}
                  </a>
                  <span>&middot;</span>
                {/if}
                <span>{formatRelativeTime(entry.timestamp)}</span>
              </div>
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/each}
</div>
