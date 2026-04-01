<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import type { ActivityItem } from '../types'
  import { Bot, GitPullRequest, CheckCircle, Play, MessageSquare, Circle } from '@lucide/svelte'
  import { activityEventTone } from '$lib/features/activity'
  import type { Component } from 'svelte'

  let {
    activities,
    class: className = '',
  }: {
    activities: ActivityItem[]
    class?: string
  } = $props()

  const typeIcons: Record<string, Component> = {
    'agent.claimed': Bot,
    'agent.launching': Play,
    'agent.ready': CheckCircle,
    'agent.failed': Bot,
    'agent.completed': CheckCircle,
    'pr.opened': GitPullRequest,
    'pr.merged': GitPullRequest,
    'pr.closed': GitPullRequest,
    comment: MessageSquare,
  }

  function getIcon(type: string): Component {
    return typeIcons[type] ?? Circle
  }

  function getColor(type: string): string {
    switch (activityEventTone(type)) {
      case 'success':
        return 'text-emerald-500'
      case 'warning':
        return 'text-amber-500'
      case 'danger':
        return 'text-red-500'
      case 'info':
        return 'text-sky-500'
      default:
        return 'text-muted-foreground'
    }
  }
</script>

<div class={cn('border-border bg-card rounded-md border', className)}>
  <div class="border-border flex items-center justify-between border-b px-4 py-3">
    <h3 class="text-foreground text-sm font-medium">Activity</h3>
    <span class="text-muted-foreground text-xs">Recent</span>
  </div>

  <div class="divide-border divide-y">
    {#each activities as item, idx (item.id)}
      {@const Icon = getIcon(item.type)}
      <div class="animate-stagger flex items-start gap-3 px-4 py-3" style="--stagger-index: {idx}">
        <Icon class={cn('mt-0.5 size-4 shrink-0', getColor(item.type))} />
        <div class="min-w-0 flex-1">
          <p class="text-foreground text-sm leading-snug">{item.message}</p>
          <div class="text-muted-foreground mt-1 flex items-center gap-2 text-xs">
            {#if item.agentName}
              <span class="font-mono">{item.agentName}</span>
              <span>&middot;</span>
            {/if}
            {#if item.ticketIdentifier}
              <span class="font-mono">{item.ticketIdentifier}</span>
              <span>&middot;</span>
            {/if}
            <span>{formatRelativeTime(item.timestamp)}</span>
          </div>
        </div>
      </div>
    {:else}
      <div class="px-4 py-8 text-center text-xs text-muted-foreground">
        No runtime activity events yet.
      </div>
    {/each}
  </div>
</div>
