<script lang="ts">
  import { formatRelativeTime } from '$lib/utils'
  import { Separator } from '$ui/separator'
  import type { WorkflowSummary } from '../types'

  let { workflow }: { workflow: WorkflowSummary } = $props()
</script>

{#if workflow.history.length > 0}
  <div class="bg-muted/20 space-y-2 px-4 py-3">
    <div class="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
      Control Plane
    </div>
    <div class="flex flex-wrap items-center gap-2 text-xs">
      <span class="text-foreground rounded-full border px-2 py-0.5 font-medium">
        Published v{workflow.version}
      </span>
      <span class="text-muted-foreground">{workflow.history.length} recorded version(s)</span>
    </div>
    <div class="flex flex-wrap gap-2">
      {#each workflow.history.slice(0, 4) as item (item.id)}
        <div class="bg-background rounded-lg border px-2.5 py-1.5 text-xs">
          <div class="text-foreground font-medium">
            v{item.version}
            {#if item.version === workflow.version}
              <span class="text-muted-foreground">· current</span>
            {/if}
          </div>
          <div class="text-muted-foreground mt-0.5">
            {formatRelativeTime(item.createdAt)} by {item.createdBy}
          </div>
        </div>
      {/each}
    </div>
  </div>

  <Separator />
{/if}
