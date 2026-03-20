<script lang="ts">
  import { Badge } from '$ui/badge'
  import CircleCheck from '@lucide/svelte/icons/circle-check'
  import CircleX from '@lucide/svelte/icons/circle-x'
  import Loader from '@lucide/svelte/icons/loader'
  import Clock from '@lucide/svelte/icons/clock'
  import ChevronDown from '@lucide/svelte/icons/chevron-down'
  import { cn, formatRelativeTime } from '$lib/utils'
  import type { HookExecution } from '../types'

  let { hooks }: { hooks: HookExecution[] } = $props()

  let expandedId = $state<string | null>(null)

  const statusConfig: Record<string, { icon: typeof CircleCheck; class: string; label: string }> = {
    pass: { icon: CircleCheck, class: 'text-green-400', label: 'Pass' },
    fail: { icon: CircleX, class: 'text-red-400', label: 'Fail' },
    running: { icon: Loader, class: 'text-yellow-400 animate-spin', label: 'Running' },
    timeout: { icon: Clock, class: 'text-orange-400', label: 'Timeout' },
  }

  function toggle(id: string) {
    expandedId = expandedId === id ? null : id
  }
</script>

<div class="flex flex-col gap-2 px-5 py-3">
  <span class="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
    Hook Executions
  </span>

  {#if hooks.length === 0}
    <p class="py-4 text-center text-xs text-muted-foreground">No hook executions yet</p>
  {/if}

  {#each hooks as hook}
    {@const config = statusConfig[hook.status]}
    <button
      onclick={() => toggle(hook.id)}
      class="flex w-full flex-col rounded-md border border-border bg-muted/30 transition-colors hover:bg-muted/50"
    >
      <div class="flex items-center gap-2 px-3 py-2">
        {#if config}
          <svelte:component this={config.icon} class={cn('size-3.5 shrink-0', config.class)} />
        {/if}
        <span class="flex-1 truncate text-left text-xs font-medium text-foreground">
          {hook.hookName}
        </span>
        {#if hook.duration != null}
          <span class="text-[10px] text-muted-foreground">{hook.duration}ms</span>
        {/if}
        <span class="text-[10px] text-muted-foreground">
          {formatRelativeTime(hook.timestamp)}
        </span>
        <ChevronDown
          class={cn(
            'size-3 text-muted-foreground transition-transform',
            expandedId === hook.id && 'rotate-180',
          )}
        />
      </div>

      {#if expandedId === hook.id && hook.output}
        <div class="border-t border-border px-3 py-2">
          <pre class="max-h-32 overflow-auto whitespace-pre-wrap text-left font-mono text-[10px] leading-relaxed text-muted-foreground">{hook.output}</pre>
        </div>
      {/if}
    </button>
  {/each}
</div>
