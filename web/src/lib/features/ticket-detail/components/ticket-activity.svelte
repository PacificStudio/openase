<script lang="ts">
  import Bot from '@lucide/svelte/icons/bot'
  import GitPullRequest from '@lucide/svelte/icons/git-pull-request'
  import Play from '@lucide/svelte/icons/play'
  import CircleCheck from '@lucide/svelte/icons/circle-check'
  import AlertTriangle from '@lucide/svelte/icons/alert-triangle'
  import MessageSquare from '@lucide/svelte/icons/message-square'
  import Settings from '@lucide/svelte/icons/settings'
  import { cn, formatRelativeTime } from '$lib/utils'
  import type { TicketActivity } from '../types'

  let { activities }: { activities: TicketActivity[] } = $props()

  const typeIcons: Record<string, { icon: typeof Bot; class: string }> = {
    agent_assigned: { icon: Bot, class: 'text-blue-400' },
    pr_opened: { icon: GitPullRequest, class: 'text-green-400' },
    started: { icon: Play, class: 'text-yellow-400' },
    completed: { icon: CircleCheck, class: 'text-green-400' },
    failed: { icon: AlertTriangle, class: 'text-red-400' },
    comment: { icon: MessageSquare, class: 'text-muted-foreground' },
    status_change: { icon: Settings, class: 'text-purple-400' },
  }

  const fallbackIcon = { icon: Settings, class: 'text-muted-foreground' }
</script>

<div class="flex flex-col gap-0 px-5 py-3">
  <span class="mb-2 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
    Activity
  </span>

  {#if activities.length === 0}
    <p class="py-4 text-center text-xs text-muted-foreground">No activity yet</p>
  {/if}

  <div class="relative">
    {#each activities as activity, i}
      {@const config = typeIcons[activity.type] ?? fallbackIcon}
      <div class="relative flex gap-3 pb-3">
        {#if i < activities.length - 1}
          <div class="absolute left-[7px] top-5 bottom-0 w-px bg-border"></div>
        {/if}

        <div class="relative z-10 mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-full bg-background">
          <svelte:component this={config.icon} class={cn('size-3.5', config.class)} />
        </div>

        <div class="flex flex-1 flex-col gap-0.5 min-w-0">
          <p class="text-xs leading-snug text-foreground">
            {activity.message}
          </p>
          <div class="flex items-center gap-2 text-[10px] text-muted-foreground">
            {#if activity.agentName}
              <span>{activity.agentName}</span>
              <span>·</span>
            {/if}
            <span>{formatRelativeTime(activity.timestamp)}</span>
          </div>
        </div>
      </div>
    {/each}
  </div>
</div>
