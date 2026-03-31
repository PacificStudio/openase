<script lang="ts">
  import { adapterIconPath, availabilityDotColor } from '$lib/features/providers'
  import { cn } from '$lib/utils'
  import { Wrench } from '@lucide/svelte'
  import type { WorkflowAgentOption } from '../types'

  let { selectedAgent }: { selectedAgent: WorkflowAgentOption | null } = $props()

  const selectedAgentIconPath = $derived(
    selectedAgent ? adapterIconPath(selectedAgent.adapterType) : '',
  )
</script>

{#if selectedAgent}
  <div class="flex items-center gap-2.5">
    {#if selectedAgentIconPath}
      <img src={selectedAgentIconPath} alt="" class="size-5 shrink-0" />
    {:else}
      <Wrench class="text-muted-foreground size-5 shrink-0" />
    {/if}
    <div class="min-w-0 text-left">
      <div class="text-foreground truncate text-sm font-medium">{selectedAgent.agentName}</div>
      <div class="text-muted-foreground truncate text-xs">
        {selectedAgent.providerName} &middot; {selectedAgent.modelName}
      </div>
    </div>
    <span
      class={cn(
        'ml-auto size-2 shrink-0 rounded-full',
        availabilityDotColor(selectedAgent.available),
      )}
    ></span>
  </div>
{:else}
  <span class="text-muted-foreground">Select bound agent</span>
{/if}
