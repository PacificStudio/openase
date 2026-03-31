<script lang="ts">
  import Bot from '@lucide/svelte/icons/bot'
  import GitPullRequest from '@lucide/svelte/icons/git-pull-request'
  import Play from '@lucide/svelte/icons/play'
  import CircleCheck from '@lucide/svelte/icons/circle-check'
  import AlertTriangle from '@lucide/svelte/icons/alert-triangle'
  import MessageSquare from '@lucide/svelte/icons/message-square'
  import Settings from '@lucide/svelte/icons/settings'
  import { cn, formatRelativeTime } from '$lib/utils'
  import { activityEventTone } from '$lib/features/activity'
  import type { TicketActivity } from '../types'

  let {
    activities,
    label = 'Activity',
    emptyText = 'No activity yet',
  }: {
    activities: TicketActivity[]
    label?: string
    emptyText?: string
  } = $props()

  function typeConfig(type: string) {
    if (type.startsWith('pr.')) return { icon: GitPullRequest, class: 'text-green-400' }
    if (type.startsWith('agent.')) return { icon: Bot, class: 'text-blue-400' }
    if (type === 'hook.started') return { icon: Play, class: 'text-yellow-400' }
    if (type === 'comment') return { icon: MessageSquare, class: 'text-muted-foreground' }

    switch (activityEventTone(type)) {
      case 'success':
        return { icon: CircleCheck, class: 'text-green-400' }
      case 'danger':
        return { icon: AlertTriangle, class: 'text-red-400' }
      default:
        return { icon: Settings, class: 'text-muted-foreground' }
    }
  }
</script>

<div class="flex flex-col gap-0 px-6 py-4">
  <span class="text-muted-foreground mb-2 text-[10px] font-medium tracking-wider uppercase">
    {label}
  </span>

  {#if activities.length === 0}
    <p class="text-muted-foreground py-4 text-center text-xs">{emptyText}</p>
  {/if}

  <div class="relative">
    {#each activities as activity, i}
      {@const config = typeConfig(activity.type)}
      <div class="relative flex gap-3 pb-3">
        {#if i < activities.length - 1}
          <div class="bg-border absolute top-5 bottom-0 left-[7px] w-px"></div>
        {/if}

        <div
          class="bg-background relative z-10 mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-full"
        >
          <config.icon class={cn('size-3.5', config.class)} />
        </div>

        <div class="flex min-w-0 flex-1 flex-col gap-0.5">
          <p class="text-foreground text-xs leading-snug">
            {activity.message}
          </p>
          <div class="text-muted-foreground flex items-center gap-2 text-[10px]">
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
