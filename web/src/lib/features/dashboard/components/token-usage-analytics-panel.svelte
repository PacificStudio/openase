<script lang="ts">
  import { cn, formatCount } from '$lib/utils'
  import { Button } from '$ui/button'
  import { LineChart } from '$ui/chart'
  import { Skeleton } from '$ui/skeleton'
  import { CalendarRange, Flame, TrendingUp, Workflow } from '@lucide/svelte'
  import { formatTokenUsageTooltip, tokenUsageRangeOptions } from '../token-usage'
  import type { TokenUsageAnalytics, TokenUsageRange } from '../types'
  import TokenUsageCalendar from './token-usage-calendar.svelte'

  let {
    analytics,
    selectedRange,
    loading = false,
    onSelectRange,
    class: className = '',
  }: {
    analytics: TokenUsageAnalytics
    selectedRange: TokenUsageRange
    loading?: boolean
    onSelectRange?: (range: TokenUsageRange) => void
    class?: string
  } = $props()

  const chartLabels = $derived(analytics.days.map((d) => d.shortLabel))
  const chartDatasets = $derived([
    {
      data: analytics.days.map((d) => d.totalTokens),
      borderColor: 'oklch(0.56 0.155 152)',
      backgroundColor: 'oklch(0.56 0.155 152 / 0.10)',
      fill: true,
      tension: 0.35,
      borderWidth: 2,
      pointRadius: analytics.days.length > 60 ? 0 : 2.5,
      pointHoverRadius: 4,
      pointBackgroundColor: 'oklch(0.56 0.155 152)',
      pointBorderColor: 'oklch(0.56 0.155 152)',
      pointBorderWidth: 0,
    },
  ])
  const chartTooltip = $derived((index: number) => {
    const day = analytics.days[index]
    return day ? formatTokenUsageTooltip(day) : ''
  })
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
      {#each tokenUsageRangeOptions as range}
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
            <LineChart
              labels={chartLabels}
              datasets={chartDatasets}
              tooltipCallback={chartTooltip}
              yTickFormat={(v) => formatCount(v)}
              class="h-48 w-full"
            />
          </div>
        {/if}
      </div>

      <TokenUsageCalendar
        days={analytics.days}
        calendarCells={analytics.calendarCells}
        maxDailyTokens={analytics.maxDailyTokens}
        {loading}
      />
    </div>
  </div>
</section>
