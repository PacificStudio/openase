<script lang="ts">
  import { cn } from '$lib/utils'
  import * as Tooltip from '$ui/tooltip'
  import type { WorkflowStatusOption } from '../types'

  let {
    label,
    statuses = [],
    selectedStatusIds = [],
    disabledReasonById = {},
    disabled = false,
    onToggle,
  }: {
    label: string
    statuses?: WorkflowStatusOption[]
    selectedStatusIds?: string[]
    disabledReasonById?: Record<string, string>
    disabled?: boolean
    onToggle?: (statusId: string) => void
  } = $props()
</script>

<div class="space-y-1.5">
  <div class="text-muted-foreground text-xs font-medium tracking-wide uppercase">{label}</div>
  <div class="flex flex-wrap gap-2">
    {#each statuses as status (status.id)}
      {@const isSelected = selectedStatusIds.includes(status.id)}
      {@const disabledReason = disabledReasonById[status.id]}
      {@const isBlocked = Boolean(disabledReason) && !isSelected}
      {#if isBlocked}
        <Tooltip.Root delayDuration={200}>
          <Tooltip.Trigger>
            {#snippet child({ props })}
              <button
                {...props}
                type="button"
                class="border-border text-muted-foreground/50 cursor-not-allowed rounded-full border border-dashed px-3 py-1.5 text-xs line-through"
                disabled
              >
                {status.name}
              </button>
            {/snippet}
          </Tooltip.Trigger>
          <Tooltip.Content side="top" class="text-xs">
            {disabledReason}
          </Tooltip.Content>
        </Tooltip.Root>
      {:else}
        <button
          type="button"
          class={cn(
            'rounded-full border px-3 py-1.5 text-xs transition-colors',
            isSelected
              ? 'border-primary/40 bg-primary/10 text-foreground font-medium'
              : 'border-border text-muted-foreground hover:bg-muted',
          )}
          {disabled}
          onclick={() => onToggle?.(status.id)}
        >
          {status.name}
        </button>
      {/if}
    {/each}
  </div>
</div>
