<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import type { ActivityItem } from '../types'
  import { Bot, GitPullRequest, CheckCircle, Play, MessageSquare, Circle } from '@lucide/svelte'
  import type { Component } from 'svelte'

  let {
    activities,
    class: className = '',
  }: {
    activities: ActivityItem[]
    class?: string
  } = $props()

  const typeIcons: Record<string, Component> = {
    'agent.launching': Play,
    'agent.ready': CheckCircle,
    'agent.failed': Bot,
    'agent.heartbeat': Bot,
    agent_started: Play,
    agent_completed: CheckCircle,
    pr_opened: GitPullRequest,
    pr_merged: GitPullRequest,
    comment: MessageSquare,
    agent_assigned: Bot,
  }

  const typeColors: Record<string, string> = {
    'agent.launching': 'text-blue-500',
    'agent.ready': 'text-emerald-500',
    'agent.failed': 'text-red-500',
    'agent.heartbeat': 'text-blue-400',
    agent_started: 'text-blue-500',
    agent_completed: 'text-emerald-500',
    pr_opened: 'text-purple-500',
    pr_merged: 'text-emerald-500',
    comment: 'text-muted-foreground',
    agent_assigned: 'text-blue-400',
  }

  function getIcon(type: string): Component {
    return typeIcons[type] ?? Circle
  }

  function getColor(type: string): string {
    return typeColors[type] ?? 'text-muted-foreground'
  }
</script>

<div class={cn('border-border bg-card rounded-md border', className)}>
  <div class="border-border flex items-center justify-between border-b px-4 py-3">
    <h3 class="text-foreground text-sm font-medium">Activity</h3>
    <span class="text-muted-foreground text-xs">Recent</span>
  </div>

  <div class="divide-border divide-y">
    {#each activities as item (item.id)}
      {@const Icon = getIcon(item.type)}
      <div class="flex items-start gap-3 px-4 py-3">
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
