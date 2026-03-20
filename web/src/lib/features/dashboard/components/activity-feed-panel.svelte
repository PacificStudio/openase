<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import type { ActivityItem } from '../types'
  import {
    Bot,
    GitPullRequest,
    CheckCircle,
    Play,
    MessageSquare,
    Circle,
  } from '@lucide/svelte'
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

<div class={cn('rounded-md border border-border bg-card', className)}>
  <div class="flex items-center justify-between border-b border-border px-4 py-3">
    <h3 class="text-sm font-medium text-foreground">Activity</h3>
    <span class="text-xs text-muted-foreground">Recent</span>
  </div>

  <div class="divide-y divide-border">
    {#each activities as item (item.id)}
      {@const Icon = getIcon(item.type)}
      <div class="flex items-start gap-3 px-4 py-3">
        <Icon class={cn('size-4 mt-0.5 shrink-0', getColor(item.type))} />
        <div class="flex-1 min-w-0">
          <p class="text-sm text-foreground leading-snug">{item.message}</p>
          <div class="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
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
