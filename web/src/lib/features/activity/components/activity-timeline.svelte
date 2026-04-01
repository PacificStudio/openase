<script lang="ts">
  import {
    AlertTriangle,
    ArrowRightLeft,
    Bot,
    CheckCircle2,
    Circle,
    GitMerge,
    GitPullRequest,
    Heart,
    LogOut,
    MessageSquare,
    Play,
    Rocket,
    TicketPlus,
    XCircle,
  } from '@lucide/svelte'
  import type { Component } from 'svelte'
  import { cn, formatRelativeTime } from '$lib/utils'
  import type { ActivityEntry } from '../types'
  import { activityEventLabel, activityEventTone } from '../event-catalog'

  let {
    entries,
    class: className = '',
  }: {
    entries: ActivityEntry[]
    class?: string
  } = $props()

  type EventStyle = {
    icon: Component
    iconColor: string
    bgColor: string
    badgeLabel: string
    badgeClass: string
    messageClass: string
  }

  const eventIcons: Record<string, Component> = {
    'agent.claimed': Bot,
    'agent.launching': Rocket,
    'agent.ready': CheckCircle2,
    'agent.paused': AlertTriangle,
    'agent.failed': XCircle,
    'agent.completed': CheckCircle2,
    'agent.terminated': LogOut,
    'agent.heartbeat': Heart,
    'ticket.created': TicketPlus,
    ticket_created: TicketPlus,
    'ticket.updated': ArrowRightLeft,
    'ticket.status_changed': ArrowRightLeft,
    status_changed: ArrowRightLeft,
    'ticket.completed': CheckCircle2,
    'ticket.cancelled': AlertTriangle,
    'ticket.retry_scheduled': AlertTriangle,
    'ticket.retry_paused': AlertTriangle,
    'ticket.budget_exhausted': AlertTriangle,
    agent_started: Play,
    'hook.started': Play,
    'hook.passed': CheckCircle2,
    'hook.failed': XCircle,
    hook_failed: XCircle,
    'pr.opened': GitPullRequest,
    pr_opened: GitPullRequest,
    'pr.merged': GitMerge,
    pr_merged: GitMerge,
    'pr.closed': GitPullRequest,
    comment: MessageSquare,
    comment_added: MessageSquare,
  }

  function getToneClasses(eventType: string) {
    switch (activityEventTone(eventType)) {
      case 'success':
        return {
          iconColor: 'text-emerald-500',
          bgColor: 'bg-emerald-500/10',
          badgeClass: 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400',
          messageClass: 'text-emerald-700 dark:text-emerald-300',
        }
      case 'warning':
        return {
          iconColor: 'text-amber-500',
          bgColor: 'bg-amber-500/10',
          badgeClass: 'bg-amber-500/15 text-amber-600 dark:text-amber-400',
          messageClass: 'text-amber-700 dark:text-amber-300',
        }
      case 'danger':
        return {
          iconColor: 'text-red-500',
          bgColor: 'bg-red-500/10',
          badgeClass: 'bg-red-500/15 text-red-600 dark:text-red-400',
          messageClass: 'text-red-700 dark:text-red-300',
        }
      case 'info':
        return {
          iconColor: 'text-sky-500',
          bgColor: 'bg-sky-500/10',
          badgeClass: 'bg-sky-500/15 text-sky-600 dark:text-sky-400',
          messageClass: 'text-foreground',
        }
      default:
        return {
          iconColor: 'text-muted-foreground',
          bgColor: 'bg-muted',
          badgeClass: 'bg-muted text-muted-foreground',
          messageClass: 'text-foreground',
        }
    }
  }

  function getStyle(eventType: string): EventStyle {
    return {
      icon: eventIcons[eventType] ?? Circle,
      badgeLabel: activityEventLabel(eventType),
      ...getToneClasses(eventType),
    }
  }

  function isHighlight(eventType: string): boolean {
    const tone = activityEventTone(eventType)
    return tone === 'success' || tone === 'danger'
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
    return Array.from(groups.entries()).map(([label, groupedEntries]) => ({
      label,
      entries: groupedEntries,
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
      <div class="space-y-1">
        {#each group.entries as entry, idx (entry.id)}
          {@const style = getStyle(entry.eventType)}
          {@const Icon = style.icon}
          {@const highlight = isHighlight(entry.eventType)}
          <div
            class={cn(
              'animate-stagger flex items-start gap-3 rounded-md px-3 py-2.5 transition-colors',
              highlight && style.bgColor,
            )}
            style="--stagger-index: {idx}"
          >
            <span
              class={cn(
                'mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full',
                style.bgColor,
              )}
            >
              <Icon class={cn('size-3.5', style.iconColor)} />
            </span>

            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span
                  class={cn(
                    'inline-flex shrink-0 items-center rounded px-1.5 py-0.5 text-[10px] leading-none font-medium',
                    style.badgeClass,
                  )}
                >
                  {style.badgeLabel}
                </span>
                <p class={cn('text-sm leading-snug', style.messageClass)}>
                  {entry.message}
                </p>
              </div>
              <div class="text-muted-foreground mt-1 flex flex-wrap items-center gap-1.5 text-xs">
                {#if entry.agentName}
                  <span class="font-mono">{entry.agentName}</span>
                {/if}
                {#if entry.ticketIdentifier}
                  <span
                    class="bg-muted inline-flex items-center rounded px-1.5 py-0.5 font-mono text-[11px] leading-none"
                  >
                    {entry.ticketIdentifier}
                  </span>
                {/if}
                <span class="text-muted-foreground/60">{formatRelativeTime(entry.timestamp)}</span>
              </div>
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/each}
</div>
