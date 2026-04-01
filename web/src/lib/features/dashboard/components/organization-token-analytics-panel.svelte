<script lang="ts">
  import { cn, formatCount } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Skeleton } from '$ui/skeleton'
  import { CalendarRange, Flame, TrendingUp, Workflow } from '@lucide/svelte'
  import {
    buildOrganizationTokenUsageTrendPoints,
    formatOrganizationTokenUsageTooltip,
    organizationTokenUsageIntensityClassName,
    organizationTokenUsageRangeOptions,
  } from '../organization-token-usage'
  import type {
    OrganizationTokenUsageAnalytics,
    OrganizationTokenUsageDayPoint,
    OrganizationTokenUsageRange,
  } from '../types'

  let {
    analytics,
    selectedRange,
    loading = false,
    onSelectRange,
    class: className = '',
  }: {
    analytics: OrganizationTokenUsageAnalytics
    selectedRange: OrganizationTokenUsageRange
    loading?: boolean
    onSelectRange?: (range: OrganizationTokenUsageRange) => void
    class?: string
  } = $props()

  const weekdayLabels = ['S', 'M', 'T', 'W', 'T', 'F', 'S']
  const trendPoints = $derived(buildOrganizationTokenUsageTrendPoints(analytics.days))
  const trendPolyline = $derived(trendPoints.map((point) => `${point.x},${point.y}`).join(' '))
  const trendArea = $derived(
    trendPoints.length > 0
      ? `2,44 ${trendPoints.map((point) => `${point.x},${point.y}`).join(' ')} 98,44`
      : '',
  )
</script>

<section class={cn('border-border bg-card rounded-xl border', className)}>
  <div
    class="border-border flex flex-col gap-3 border-b px-4 py-4 lg:flex-row lg:items-start lg:justify-between"
  >
    <div>
      <div class="flex items-center gap-2">
        <h2 class="text-foreground text-base font-semibold">Token Usage</h2>
        <CalendarRange class="text-muted-foreground size-4" />
      </div>
      <p class="text-muted-foreground mt-1 text-sm">
        UTC-day snapshots materialized from terminalized agent runs.
      </p>
    </div>

    <div class="flex flex-wrap gap-2">
      {#each organizationTokenUsageRangeOptions as range}
        <Button
          size="sm"
          variant={range === selectedRange ? 'default' : 'outline'}
          onclick={() => onSelectRange?.(range)}
        >
          {range}d
        </Button>
      {/each}
    </div>
  </div>

  <div class="space-y-5 p-4">
    <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
      {#if loading}
        {#each { length: 4 } as _}
          <div class="bg-muted/40 rounded-lg px-3 py-3">
            <Skeleton class="h-3 w-16" />
            <Skeleton class="mt-2 h-6 w-20" />
          </div>
        {/each}
      {:else}
        <div class="bg-muted/40 rounded-lg px-3 py-3">
          <div class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">Window</div>
          <div class="text-foreground mt-2 text-lg font-semibold">{selectedRange} days</div>
        </div>
        <div class="bg-muted/40 rounded-lg px-3 py-3">
          <div
            class="text-muted-foreground flex items-center gap-1.5 text-[11px] tracking-[0.12em] uppercase"
          >
            <TrendingUp class="size-3" />
            <span>Total tokens</span>
          </div>
          <div class="text-foreground mt-2 text-lg font-semibold">
            {formatCount(analytics.totalTokens)}
          </div>
        </div>
        <div class="bg-muted/40 rounded-lg px-3 py-3">
          <div
            class="text-muted-foreground flex items-center gap-1.5 text-[11px] tracking-[0.12em] uppercase"
          >
            <Flame class="size-3" />
            <span>Peak day</span>
          </div>
          <div class="text-foreground mt-2 text-lg font-semibold">
            {analytics.peakDay ? formatCount(analytics.peakDay.totalTokens) : '—'}
          </div>
          <div class="text-muted-foreground mt-1 text-xs">
            {analytics.peakDay?.dayLabel ?? 'No finalized runs yet'}
          </div>
        </div>
        <div class="bg-muted/40 rounded-lg px-3 py-3">
          <div
            class="text-muted-foreground flex items-center gap-1.5 text-[11px] tracking-[0.12em] uppercase"
          >
            <Workflow class="size-3" />
            <span>Avg daily / runs</span>
          </div>
          <div class="text-foreground mt-2 text-lg font-semibold">
            {formatCount(analytics.avgDailyTokens)}
          </div>
          <div class="text-muted-foreground mt-1 text-xs">
            {formatCount(analytics.totalRuns)} finalized runs
          </div>
        </div>
      {/if}
    </div>

    <div class="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1.4fr)_minmax(18rem,0.9fr)]">
      <div class="border-border rounded-lg border p-4">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-foreground text-sm font-medium">Trend</h3>
            <p class="text-muted-foreground mt-1 text-xs">
              Finalized token usage attributed to run terminal day.
            </p>
          </div>
          <div class="text-right text-xs">
            <div class="text-muted-foreground">High</div>
            <div class="text-foreground font-medium">{formatCount(analytics.maxDailyTokens)}</div>
          </div>
        </div>

        {#if loading}
          <div class="mt-4 space-y-3">
            <Skeleton class="h-44 w-full rounded-lg" />
            <div class="flex justify-between">
              <Skeleton class="h-3 w-16" />
              <Skeleton class="h-3 w-16" />
            </div>
          </div>
        {:else if analytics.days.length === 0}
          <div class="text-muted-foreground flex h-48 items-center justify-center text-sm">
            No token snapshots yet.
          </div>
        {:else if analytics.maxDailyTokens === 0}
          <div class="flex h-48 flex-col items-center justify-center gap-2 text-center">
            <p class="text-foreground text-sm font-medium">No finalized token usage in range.</p>
            <p class="text-muted-foreground max-w-sm text-xs">
              Daily snapshots appear here once runs reach a terminal state.
            </p>
          </div>
        {:else}
          <div class="mt-4">
            <svg
              viewBox="0 0 100 48"
              class="h-48 w-full overflow-visible"
              preserveAspectRatio="none"
            >
              <defs>
                <linearGradient id="token-usage-area" x1="0" x2="0" y1="0" y2="1">
                  <stop offset="0%" stop-color="currentColor" stop-opacity="0.22" />
                  <stop offset="100%" stop-color="currentColor" stop-opacity="0.02" />
                </linearGradient>
              </defs>
              {#each [10, 22, 34, 44] as y}
                <line x1="2" y1={y} x2="98" y2={y} class="stroke-border/80" stroke-width="0.45" />
              {/each}
              <polygon points={trendArea} class="fill-current text-emerald-500/90" />
              <polyline
                points={trendPolyline}
                class="text-emerald-600 dark:text-emerald-400"
                fill="none"
                stroke="currentColor"
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="1.7"
              />
              {#each trendPoints as point (point.day.date)}
                <circle
                  cx={point.x}
                  cy={point.y}
                  r="1.8"
                  class="fill-card stroke-emerald-600 dark:stroke-emerald-400"
                  stroke-width="1.4"
                >
                  <title>{formatOrganizationTokenUsageTooltip(point.day)}</title>
                </circle>
              {/each}
            </svg>

            <div class="text-muted-foreground mt-3 flex items-center justify-between text-xs">
              <span>{analytics.days[0]?.shortLabel ?? '—'}</span>
              <span>{analytics.days[analytics.days.length - 1]?.shortLabel ?? '—'}</span>
            </div>
          </div>
        {/if}
      </div>

      <div class="border-border rounded-lg border p-4">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-foreground text-sm font-medium">Calendar</h3>
            <p class="text-muted-foreground mt-1 text-xs">
              Heatmap intensity tracks each UTC day&apos;s finalized token total.
            </p>
          </div>
          <div class="text-muted-foreground text-xs">Low to high</div>
        </div>

        {#if loading}
          <div class="mt-4 grid grid-cols-7 gap-1.5">
            {#each { length: 35 } as _}
              <Skeleton class="aspect-square h-auto w-full rounded-sm" />
            {/each}
          </div>
        {:else}
          <div class="mt-4">
            <div class="mb-2 grid grid-cols-7 gap-1.5">
              {#each weekdayLabels as label}
                <div class="text-muted-foreground text-center text-[10px] font-medium uppercase">
                  {label}
                </div>
              {/each}
            </div>
            <div class="grid grid-cols-7 gap-1.5">
              {#each analytics.calendarCells as day, index (`${day?.date ?? 'empty'}-${index}`)}
                {#if day}
                  <div
                    class={cn(
                      'border-border/80 aspect-square rounded-sm border',
                      organizationTokenUsageIntensityClassName(day.intensity),
                    )}
                    title={formatOrganizationTokenUsageTooltip(day)}
                  ></div>
                {:else}
                  <div
                    class="border-border/40 bg-muted/15 aspect-square rounded-sm border border-dashed"
                  ></div>
                {/if}
              {/each}
            </div>
            <div class="mt-3 flex items-center justify-between text-xs">
              <span class="text-muted-foreground">0</span>
              <div class="flex items-center gap-1">
                {#each [0, 1, 2, 3, 4] as intensity}
                  <div
                    class={cn(
                      'border-border/80 size-2.5 rounded-[4px] border',
                      organizationTokenUsageIntensityClassName(
                        intensity as OrganizationTokenUsageDayPoint['intensity'],
                      ),
                    )}
                  ></div>
                {/each}
              </div>
              <span class="text-muted-foreground">{formatCount(analytics.maxDailyTokens)}</span>
            </div>
          </div>
        {/if}
      </div>
    </div>

    <p class="text-muted-foreground text-xs">
      Missing days are lazily backfilled from finalized runs when this view is queried.
    </p>
  </div>
</section>
