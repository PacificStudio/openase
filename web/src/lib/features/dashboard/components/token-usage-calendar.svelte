<script lang="ts">
  import { cn, formatCount } from '$lib/utils'
  import { Skeleton } from '$ui/skeleton'

  import { formatTokenUsageTooltip, tokenUsageIntensityClassName } from '../token-usage'
  import type { TokenUsageDayPoint } from '../types'

  let {
    days,
    calendarCells = [],
    maxDailyTokens,
    loading = false,
    class: className = '',
  }: {
    days: TokenUsageDayPoint[]
    calendarCells?: Array<TokenUsageDayPoint | null>
    maxDailyTokens: number
    loading?: boolean
    class?: string
  } = $props()

  const weekdayLabels = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
  const visibleCells = $derived(calendarCells.length > 0 ? calendarCells : days)
  const activeDays = $derived(days.filter((day) => day.totalTokens > 0).length)
  const rangeLabel = $derived(
    days.length > 0
      ? `${days[0]?.shortLabel ?? '—'} - ${days[days.length - 1]?.shortLabel ?? '—'}`
      : '—',
  )

  function dayNumber(date: string) {
    return Number(date.slice(-2))
  }

  function cellValue(day: TokenUsageDayPoint) {
    if (day.totalTokens <= 0) return ''
    return formatCount(day.totalTokens)
  }
</script>

<div class={cn('border-border rounded-lg border p-4', className)}>
  <div class="flex items-start justify-between gap-3">
    <div>
      <h3 class="text-foreground text-sm font-medium">Calendar</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Daily token usage laid out on a fixed weekly grid.
      </p>
    </div>
    <div class="text-right text-xs">
      <div class="text-muted-foreground">Active days</div>
      <div class="text-foreground font-medium">{activeDays}/{days.length}</div>
    </div>
  </div>

  {#if loading}
    <div class="mt-4">
      <Skeleton class="h-[220px] w-full rounded-lg" />
    </div>
  {:else if days.length === 0}
    <div class="text-muted-foreground flex h-[220px] items-center justify-center text-sm">
      No activity yet.
    </div>
  {:else}
    <div class="bg-muted/20 mt-4 rounded-xl border border-dashed p-3">
      <div class="text-muted-foreground mb-3 flex items-center justify-between text-[11px]">
        <span>{rangeLabel}</span>
        <span>Peak {formatCount(maxDailyTokens)}</span>
      </div>

      <div class="grid grid-cols-7 gap-1.5" data-testid="token-usage-calendar-grid">
        {#each weekdayLabels as weekday}
          <div
            class="text-muted-foreground text-center text-[10px] font-medium tracking-[0.12em] uppercase"
          >
            {weekday}
          </div>
        {/each}

        {#each visibleCells as day, index (day?.date ?? `blank-${index}`)}
          {#if day}
            <div
              class={cn(
                'border-border/50 relative aspect-square min-h-10 rounded-md border p-1.5 transition-transform hover:-translate-y-0.5',
                tokenUsageIntensityClassName(day.intensity),
              )}
              title={formatTokenUsageTooltip(day)}
              aria-label={formatTokenUsageTooltip(day)}
              data-testid="token-usage-calendar-cell"
            >
              <div class="flex h-full flex-col justify-between">
                <span class="text-foreground text-[10px] leading-none font-medium">
                  {dayNumber(day.date)}
                </span>
                <span class="text-foreground/80 text-[9px] leading-tight">
                  {cellValue(day)}
                </span>
              </div>
            </div>
          {:else}
            <div
              class="border-border/30 aspect-square min-h-10 rounded-md border border-dashed bg-transparent"
              aria-hidden="true"
            ></div>
          {/if}
        {/each}
      </div>
    </div>

    <div class="mt-3 flex items-center justify-between gap-3 text-[10px]">
      <span class="text-muted-foreground">{rangeLabel}</span>
      <div class="flex items-center gap-2">
        <span class="text-muted-foreground">Less</span>
        {#each [0, 1, 2, 3, 4] as intensity}
          <div
            class={cn(
              'border-border/40 size-3 rounded-[4px] border',
              tokenUsageIntensityClassName(intensity as TokenUsageDayPoint['intensity']),
            )}
          ></div>
        {/each}
        <span class="text-muted-foreground">More</span>
      </div>
    </div>
  {/if}
</div>
