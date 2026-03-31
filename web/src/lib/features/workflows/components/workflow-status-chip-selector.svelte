<script lang="ts">
  import { cn } from '$lib/utils'
  import type { WorkflowStatusOption } from '../types'

  let {
    label,
    statuses = [],
    selectedStatusIds = [],
    disabled = false,
    onToggle,
  }: {
    label: string
    statuses?: WorkflowStatusOption[]
    selectedStatusIds?: string[]
    disabled?: boolean
    onToggle?: (statusId: string) => void
  } = $props()
</script>

<div class="space-y-1.5">
  <div class="text-muted-foreground text-xs font-medium tracking-wide uppercase">{label}</div>
  <div class="flex flex-wrap gap-2">
    {#each statuses as status (status.id)}
      <button
        type="button"
        class={cn(
          'rounded-full border px-3 py-1.5 text-xs transition-colors',
          selectedStatusIds.includes(status.id)
            ? 'border-primary/40 bg-primary/10 text-foreground font-medium'
            : 'border-border text-muted-foreground hover:bg-muted',
        )}
        {disabled}
        onclick={() => onToggle?.(status.id)}
      >
        {status.name}
      </button>
    {/each}
  </div>
</div>
