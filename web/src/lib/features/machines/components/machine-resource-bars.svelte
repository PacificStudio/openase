<script lang="ts">
  import * as Tooltip from '$ui/tooltip'
  import { cn } from '$lib/utils'

  type BarSegment = {
    percent: number
    barClass: string
    label?: string
  }

  type ResourceBar = {
    key: string
    label: string
    percent: number
    summary: string
    detail: string
    barClass: string
    segments?: BarSegment[]
  }

  let { bars }: { bars: ResourceBar[] } = $props()
</script>

<div class="grid min-w-0 grid-cols-2 gap-x-4 gap-y-2 sm:grid-cols-4">
  {#each bars as bar (bar.key)}
    <Tooltip.Root>
      <Tooltip.Trigger>
        {#snippet child({ props })}
          <div {...props} class="min-w-0 cursor-help space-y-1">
            <div class="flex items-baseline justify-between gap-1">
              <span class="text-muted-foreground truncate text-[10px] font-medium uppercase">
                {bar.label}
              </span>
              <span class="text-foreground shrink-0 text-xs font-medium">{bar.summary}</span>
            </div>
            {#if bar.segments && bar.segments.length > 0}
              <div class="space-y-0.5">
                {#each bar.segments as seg (seg.label ?? seg.barClass)}
                  <div class="bg-muted h-1 overflow-hidden rounded-full">
                    <div
                      class={cn('h-full rounded-full transition-all', seg.barClass)}
                      style={`width: ${seg.percent}%`}
                    ></div>
                  </div>
                {/each}
              </div>
            {:else}
              <div class="bg-muted h-1.5 overflow-hidden rounded-full">
                <div
                  class={cn('h-full rounded-full transition-all', bar.barClass)}
                  style={`width: ${bar.percent}%`}
                ></div>
              </div>
            {/if}
          </div>
        {/snippet}
      </Tooltip.Trigger>
      <Tooltip.Content
        side="top"
        sideOffset={6}
        class="bg-popover text-popover-foreground border-border max-w-[20rem] rounded-lg border p-3 shadow-xl"
        arrowClasses="bg-popover fill-popover"
      >
        <div class="space-y-1">
          <div class="text-sm font-medium">{bar.label}</div>
          <p class="text-muted-foreground text-xs leading-5">{bar.detail}</p>
        </div>
      </Tooltip.Content>
    </Tooltip.Root>
  {/each}
</div>
