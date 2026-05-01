<script lang="ts">
  import { cn } from '$lib/utils'

  let {
    currentStepIndex,
    totalSteps,
  }: {
    currentStepIndex: number
    totalSteps: number
  } = $props()
</script>

<div class="flex items-center justify-between gap-1 px-1 pt-1">
  <div class="flex flex-1 gap-1">
    {#each Array(totalSteps) as _, idx (idx)}
      <span
        class={cn(
          'h-1 flex-1 rounded-full transition-colors',
          idx <= currentStepIndex ? 'bg-primary' : 'bg-muted',
        )}
      ></span>
    {/each}
  </div>
  <span class="text-muted-foreground shrink-0 text-[10px] font-medium">
    {currentStepIndex + 1} / {totalSteps}
  </span>
</div>
