<script lang="ts">
  import { cn } from '$lib/utils'
  import { Label } from '$ui/label'
  import * as Tooltip from '$ui/tooltip'
  import type { WorkflowStatusOption } from '../types'

  let {
    label,
    statuses,
    selectedIds,
    disabled = false,
    disabledReasonById = {},
    onToggle,
  }: {
    label: string
    statuses: WorkflowStatusOption[]
    selectedIds: string[]
    disabled?: boolean
    disabledReasonById?: Record<string, string>
    onToggle: (statusId: string) => void
  } = $props()

  function chipClass(selected: boolean) {
    return cn(
      'rounded-full border px-3 py-1.5 text-xs transition-colors',
      selected
        ? 'border-primary/40 bg-primary/10 text-foreground'
        : 'border-border text-muted-foreground hover:bg-muted',
    )
  }
</script>

<div class="space-y-2" role="group" aria-label={label}>
  <Label>{label}</Label>
  <div class="flex flex-wrap gap-2">
    {#each statuses as status (status.id)}
      {@const disabledReason = disabledReasonById[status.id]}
      {@const isBlocked = Boolean(disabledReason) && !selectedIds.includes(status.id)}
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
          class={chipClass(selectedIds.includes(status.id))}
          {disabled}
          onclick={() => onToggle(status.id)}
        >
          {status.name}
        </button>
      {/if}
    {/each}
  </div>
</div>
