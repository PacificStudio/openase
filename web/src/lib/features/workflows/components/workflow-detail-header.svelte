<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import { workflowFamilyDescriptions } from '../model'
  import { Button } from '$ui/button'
  import { Power } from '@lucide/svelte'
  import type { WorkflowSummary } from '../types'

  let {
    workflow,
    isActive,
    disabled = false,
    onToggle,
  }: {
    workflow: WorkflowSummary
    isActive: boolean
    disabled?: boolean
    onToggle?: () => void
  } = $props()
</script>

<div class="px-4 py-3">
  <div class="flex items-start justify-between gap-3">
    <div>
      <h3 class="text-foreground text-sm font-medium">{workflow.name}</h3>
      <div class="text-muted-foreground mt-1 flex items-center gap-2 text-xs">
        <span>{workflow.type}</span>
        <span>{workflowFamilyDescriptions[workflow.workflowFamily]}</span>
        {#if workflow.roleName}
          <span>{workflow.roleName}</span>
        {/if}
        <span>v{workflow.version}</span>
        <span class={cn('size-1.5 rounded-full', isActive ? 'bg-emerald-500' : 'bg-neutral-500')}
        ></span>
        <span>{isActive ? 'Active' : 'Inactive'}</span>
      </div>
    </div>
    <Button
      type="button"
      variant={isActive ? 'outline' : 'default'}
      size="sm"
      {disabled}
      onclick={() => onToggle?.()}
    >
      <Power class="size-4" />
      {isActive ? 'Deactivate' : 'Activate'}
    </Button>
  </div>
  <div class="text-muted-foreground mt-2 text-xs">
    Last modified {formatRelativeTime(workflow.lastModified)}
  </div>
</div>
