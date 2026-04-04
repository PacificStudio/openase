<script lang="ts">
  import { ChevronRight, Play, CheckCircle, AlertCircle, Info, Loader } from '@lucide/svelte'
  import { cn } from '$lib/utils'
  import type { TicketRunTranscriptBlock } from '../types'

  let { block }: { block: Extract<TicketRunTranscriptBlock, { kind: 'task_status' }> } = $props()

  let expanded = $state(false)

  function statusIcon(statusType: string) {
    switch (statusType) {
      case 'task_started':
        return Play
      case 'task_progress':
        return Loader
      case 'reasoning_updated':
        return Info
      case 'thread_status':
      case 'session_state':
        return Info
      case 'turn_done':
        return CheckCircle
      case 'error':
        return AlertCircle
      default:
        return Info
    }
  }

  function statusColor(statusType: string) {
    switch (statusType) {
      case 'task_started':
      case 'thread_status':
        return 'text-sky-500'
      case 'task_progress':
        return 'text-sky-400'
      case 'reasoning_updated':
        return 'text-amber-500'
      case 'session_state':
        return 'text-amber-600'
      case 'turn_done':
        return 'text-emerald-500'
      case 'error':
        return 'text-red-500'
      default:
        return 'text-muted-foreground'
    }
  }

  const Icon = $derived(statusIcon(block.statusType))
  const iconColor = $derived(statusColor(block.statusType))
  const hasDetails = $derived(
    Boolean(block.detail) || Boolean(block.raw && Object.keys(block.raw).length),
  )
</script>

<div class="group">
  <button
    type="button"
    class="hover:bg-muted/40 flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors"
    onclick={() => {
      if (hasDetails) expanded = !expanded
    }}
    disabled={!hasDetails}
  >
    {#if hasDetails}
      <ChevronRight
        class={cn(
          'text-muted-foreground size-3 shrink-0 transition-transform duration-150',
          expanded && 'rotate-90',
        )}
      />
    {:else}
      <span class="size-3 shrink-0"></span>
    {/if}
    <Icon class={cn('size-3.5 shrink-0', iconColor)} />
    <span class="text-foreground min-w-0 flex-1 truncate">{block.title}</span>
    <span class="text-muted-foreground/60 shrink-0 text-[10px]"
      >{block.statusType.replace(/_/g, ' ')}</span
    >
  </button>

  {#if expanded}
    <div class="border-border/40 ml-5 border-l pt-1 pb-2 pl-3 text-xs">
      {#if block.detail}
        <p class="text-muted-foreground whitespace-pre-wrap">{block.detail}</p>
      {/if}
      {#if block.raw && Object.keys(block.raw).length > 0}
        <details class="mt-1">
          <summary class="text-muted-foreground hover:text-foreground cursor-pointer"
            >Raw payload</summary
          >
          <pre
            class="bg-muted/60 mt-1 max-h-48 overflow-auto rounded-md px-2.5 py-1.5 font-mono text-[11px] leading-5 whitespace-pre-wrap">{JSON.stringify(
              block.raw,
              null,
              2,
            )}</pre>
        </details>
      {/if}
    </div>
  {/if}
</div>
