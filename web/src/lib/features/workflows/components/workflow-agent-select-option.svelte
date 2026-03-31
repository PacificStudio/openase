<script lang="ts">
  import { adapterIconPath, availabilityDotColor } from '$lib/features/providers'
  import { cn } from '$lib/utils'
  import { Wrench } from '@lucide/svelte'
  import type { WorkflowAgentOption } from '../types'

  let { option }: { option: WorkflowAgentOption } = $props()

  const iconPath = $derived(adapterIconPath(option.adapterType))
</script>

<div class="flex items-center gap-2.5 py-0.5">
  {#if iconPath}
    <img src={iconPath} alt="" class="size-4 shrink-0" />
  {:else}
    <Wrench class="text-muted-foreground size-4 shrink-0" />
  {/if}
  <span class="truncate">{option.agentName}</span>
  <span class={cn('size-1.5 shrink-0 rounded-full', availabilityDotColor(option.available))}></span>
  <span class="text-muted-foreground ml-auto truncate text-xs">{option.machineName}</span>
</div>
