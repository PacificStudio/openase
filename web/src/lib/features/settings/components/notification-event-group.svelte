<script lang="ts">
  import type { NotificationRule } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Switch } from '$ui/switch'
  import type { EventGroup, EventSeverity } from '../notification-event-catalog'

  let {
    group,
    expanded,
    rules,
    togglingId,
    onToggleGroup,
    onToggleRule,
    onNewRule,
    onEditRule,
  }: {
    group: EventGroup
    expanded: boolean
    rules: NotificationRule[]
    togglingId: string | null
    onToggleGroup: (groupKey: string) => void
    onToggleRule: (rule: NotificationRule) => void
    onNewRule: (eventType: string) => void
    onEditRule: (rule: NotificationRule) => void
  } = $props()

  function ruleForEvent(eventType: string): NotificationRule | undefined {
    return rules.find((r) => r.event_type === eventType)
  }

  function groupStats(): { total: number; active: number } {
    let total = 0
    let active = 0
    for (const event of group.events) {
      const rule = ruleForEvent(event.eventType)
      if (rule) {
        total++
        if (rule.is_enabled) active++
      }
    }
    return { total, active }
  }

  function severityColor(severity: EventSeverity): string {
    switch (severity) {
      case 'critical':
        return 'text-red-500 bg-red-500/10 border-red-500/20'
      case 'warning':
        return 'text-amber-500 bg-amber-500/10 border-amber-500/20'
      case 'info':
        return 'text-blue-500 bg-blue-500/10 border-blue-500/20'
    }
  }

  function severityDot(severity: EventSeverity): string {
    switch (severity) {
      case 'critical':
        return 'bg-red-500'
      case 'warning':
        return 'bg-amber-500'
      case 'info':
        return 'bg-blue-500'
    }
  }

  const stats = $derived(groupStats())
</script>

<div class="border-border bg-card overflow-hidden rounded-lg border">
  <button
    type="button"
    class="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-muted/30"
    onclick={() => onToggleGroup(group.key)}
  >
    <svg
      class="text-muted-foreground size-4 shrink-0 transition-transform {expanded ? 'rotate-90' : ''}"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      stroke-width="2"
      stroke="currentColor"
    >
      <path stroke-linecap="round" stroke-linejoin="round" d="m8.25 4.5 7.5 7.5-7.5 7.5" />
    </svg>
    <span class="text-sm font-medium">{group.label}</span>
    <span class="text-muted-foreground text-xs">{group.events.length} events</span>
    {#if stats.total > 0}
      <Badge variant="outline" class="ml-auto text-[10px]">
        {stats.active}/{stats.total} active
      </Badge>
    {:else}
      <span class="text-muted-foreground ml-auto text-xs">No rules</span>
    {/if}
  </button>

  {#if expanded}
    <div class="border-t border-border/50">
      {#each group.events as event, idx (event.eventType)}
        {@const rule = ruleForEvent(event.eventType)}
        <div
          class="flex items-center gap-3 px-4 py-2.5 {idx > 0 ? 'border-t border-border/30' : ''}"
        >
          <span class="size-2 shrink-0 rounded-full {severityDot(event.severity)}"></span>
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="text-sm">{event.label}</span>
              <span
                class="rounded-sm border px-1.5 py-0.5 text-[10px] font-medium leading-none {severityColor(event.severity)}"
              >
                {event.severity}
              </span>
            </div>
            {#if rule}
              <p class="text-muted-foreground mt-0.5 text-xs">
                &rarr; {rule.channel.name}
                {#if !rule.channel.is_enabled}
                  <span class="text-amber-500">(channel disabled)</span>
                {/if}
              </p>
            {/if}
          </div>

          {#if rule}
            <Button
              variant="ghost"
              size="sm"
              class="h-7 px-2 text-xs"
              onclick={() => onEditRule(rule)}
            >
              Edit
            </Button>
            <Switch
              checked={rule.is_enabled}
              disabled={togglingId === rule.id}
              onCheckedChange={() => onToggleRule(rule)}
            />
          {:else}
            <Button
              variant="ghost"
              size="sm"
              class="text-muted-foreground h-7 px-2 text-xs"
              onclick={() => onNewRule(event.eventType)}
            >
              + Add rule
            </Button>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
