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

<div class="grid min-w-0 gap-3 md:grid-cols-2 xl:grid-cols-4">
  {#each bars as bar (bar.key)}
    <div class="space-y-2 rounded-xl border border-transparent px-2 py-1">
      <Tooltip.Root>
        <Tooltip.Trigger>
          {#snippet child({ props })}
            <button
              {...props}
              type="button"
              class="text-muted-foreground inline-flex cursor-help items-center rounded-full border border-dashed px-2 py-0.5 text-[11px] font-medium tracking-[0.08em] uppercase"
            >
              {bar.label}
            </button>
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

      <div class="space-y-1">
        <div class="text-foreground text-sm font-medium">{bar.summary}</div>
        {#if bar.segments && bar.segments.length > 0}
          {#each bar.segments as seg (seg.label ?? seg.barClass)}
            <div class="flex items-center gap-2">
              {#if seg.label}
                <span class="text-muted-foreground w-10 shrink-0 text-[10px]">{seg.label}</span>
              {/if}
              <div class="bg-muted h-1.5 flex-1 overflow-hidden rounded-full">
                <div
                  class={cn('h-full rounded-full transition-all', seg.barClass)}
                  style={`width: ${seg.percent}%`}
                ></div>
              </div>
            </div>
          {/each}
        {:else}
          <div class="bg-muted h-2 overflow-hidden rounded-full">
            <div
              class={cn('h-full rounded-full transition-all', bar.barClass)}
              style={`width: ${bar.percent}%`}
            ></div>
          </div>
        {/if}
      </div>
    </div>
  {/each}
</div>
