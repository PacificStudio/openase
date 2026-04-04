<script lang="ts">
  import type { ProviderRateLimitSummary } from './rate-limit'

  let { rateLimit }: { rateLimit: ProviderRateLimitSummary } = $props()

  function barColor(usedPercent: number): string {
    if (usedPercent >= 90) return 'bg-red-500'
    if (usedPercent >= 70) return 'bg-amber-500'
    return 'bg-emerald-500'
  }

  function barTrackColor(usedPercent: number): string {
    if (usedPercent >= 90) return 'bg-red-500/15'
    if (usedPercent >= 70) return 'bg-amber-500/15'
    return 'bg-emerald-500/15'
  }
</script>

<div class="bg-muted/30 rounded-lg border px-3 py-2 text-[11px]">
  {#if rateLimit.windows.length > 0}
    <div class="space-y-2">
      {#each rateLimit.windows as window}
        <div>
          <div class="flex items-center justify-between gap-2">
            <span class="text-muted-foreground">{window.label}</span>
            <div class="flex items-center gap-2">
              <span class="text-foreground font-medium tabular-nums">
                {window.usedPercent.toFixed(1)}% used
              </span>
              {#if window.windowMinutes != null}
                <span class="text-muted-foreground">· {window.windowMinutes}m</span>
              {/if}
              {#if rateLimit.planType}
                <span class="text-muted-foreground">· {rateLimit.planType}</span>
              {/if}
            </div>
          </div>
          <div
            class="mt-1 h-1.5 w-full overflow-hidden rounded-full {barTrackColor(
              window.usedPercent,
            )}"
          >
            <div
              class="h-full rounded-full transition-all duration-300 {barColor(window.usedPercent)}"
              style="width: {Math.min(window.usedPercent, 100)}%"
            ></div>
          </div>
          {#if window.resetsAt}
            <div class="text-muted-foreground mt-1">
              Resets {new Date(window.resetsAt).toLocaleString()}
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {:else}
    <div class="flex items-center justify-between gap-3">
      <span class="text-muted-foreground">Rate limit</span>
      <span class="text-foreground font-medium">{rateLimit.headline}</span>
    </div>
    <div class="text-muted-foreground mt-1">{rateLimit.detail}</div>
  {/if}
  <div class="text-muted-foreground mt-1">{rateLimit.updatedLabel}</div>
</div>
