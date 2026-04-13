import { formatCount } from '$lib/utils'
import { getOrganizationTokenUsage, getProjectTokenUsage } from '$lib/api/openase'
import type { TokenUsageDay, TokenUsageResponse } from '$lib/api/contracts'
import type {
  TokenUsageAnalytics,
  TokenUsageDayPoint,
  TokenUsagePeak,
  TokenUsageRange,
} from './types'

export const tokenUsageRangeOptions = [7, 30, 90, 365] as const satisfies readonly TokenUsageRange[]

const shortDateFormatter = new Intl.DateTimeFormat('en-US', {
  timeZone: 'UTC',
  month: 'short',
  day: 'numeric',
})

const longDateFormatter = new Intl.DateTimeFormat('en-US', {
  timeZone: 'UTC',
  weekday: 'short',
  month: 'short',
  day: 'numeric',
})

export function emptyTokenUsageAnalytics(rangeDays: TokenUsageRange): TokenUsageAnalytics {
  return {
    rangeDays,
    days: [],
    calendarCells: [],
    totalTokens: 0,
    avgDailyTokens: 0,
    totalRuns: 0,
    peakDay: null,
    maxDailyTokens: 0,
  }
}

export function buildTokenUsageQuery(rangeDays: TokenUsageRange, now = new Date()) {
  const toDate = startOfUTCDay(now)
  const fromDate = addUTCDays(toDate, -(rangeDays - 1))

  return {
    from: toUTCDateString(fromDate),
    to: toUTCDateString(toDate),
  }
}

export function mapTokenUsage(
  payload: TokenUsageResponse,
  rangeDays: TokenUsageRange,
  now = new Date(),
): TokenUsageAnalytics {
  const query = buildTokenUsageQuery(rangeDays, now)
  const rawByDate = new Map(
    (payload.days ?? [])
      .filter((day): day is TokenUsageDay & { date: string } => Boolean(day.date))
      .map((day) => [day.date, day] as const),
  )

  const days: TokenUsageDayPoint[] = []
  const fromDate = parseUTCDate(query.from)
  const toDate = parseUTCDate(query.to)

  for (let cursor = fromDate; cursor <= toDate; cursor = addUTCDays(cursor, 1)) {
    const usageDate = toUTCDateString(cursor)
    const raw = rawByDate.get(usageDate)
    days.push({
      date: usageDate,
      dayLabel: longDateFormatter.format(cursor),
      shortLabel: shortDateFormatter.format(cursor),
      inputTokens: raw?.input_tokens ?? 0,
      outputTokens: raw?.output_tokens ?? 0,
      cachedInputTokens: raw?.cached_input_tokens ?? 0,
      reasoningTokens: raw?.reasoning_tokens ?? 0,
      totalTokens: raw?.total_tokens ?? 0,
      finalizedRunCount: raw?.finalized_run_count ?? 0,
      intensity: 0,
    })
  }

  const maxDailyTokens = days.reduce((max, day) => Math.max(max, day.totalTokens), 0)
  const normalizedDays = days.map((day) => ({
    ...day,
    intensity: classifyIntensity(day.totalTokens, maxDailyTokens),
  }))
  const calendarCells = buildCalendarCells(normalizedDays)
  const totalTokens =
    payload.summary?.total_tokens ?? normalizedDays.reduce((sum, day) => sum + day.totalTokens, 0)
  const avgDailyTokens =
    payload.summary?.avg_daily_tokens ??
    (normalizedDays.length > 0 ? Math.round(totalTokens / normalizedDays.length) : 0)
  const totalRuns = normalizedDays.reduce((sum, day) => sum + day.finalizedRunCount, 0)
  const peakDay = mapPeakDay(payload, normalizedDays)

  return {
    rangeDays,
    days: normalizedDays,
    calendarCells,
    totalTokens,
    avgDailyTokens,
    totalRuns,
    peakDay,
    maxDailyTokens,
  }
}

export async function loadOrganizationTokenUsage(
  orgId: string,
  rangeDays: TokenUsageRange,
  opts?: { signal?: AbortSignal; now?: Date },
) {
  return loadTokenUsage(
    (query, requestOpts) => getOrganizationTokenUsage(orgId, query, requestOpts),
    rangeDays,
    opts,
  )
}

export async function loadProjectTokenUsage(
  projectId: string,
  rangeDays: TokenUsageRange,
  opts?: { signal?: AbortSignal; now?: Date },
) {
  return loadTokenUsage(
    (query, requestOpts) => getProjectTokenUsage(projectId, query, requestOpts),
    rangeDays,
    opts,
  )
}

export function buildTokenUsageTrendPoints(days: TokenUsageDayPoint[]) {
  if (days.length === 0) return []

  const usableWidth = 96
  const usableHeight = 34
  const startX = 2
  const startY = 8
  const maxTokens = days.reduce((max, day) => Math.max(max, day.totalTokens), 0)
  const stepX = days.length === 1 ? 0 : usableWidth / (days.length - 1)

  return days.map((day, index) => {
    const ratio = maxTokens <= 0 ? 0 : day.totalTokens / maxTokens
    return {
      day,
      x: Number((startX + stepX * index).toFixed(2)),
      y: Number((startY + usableHeight * (1 - ratio)).toFixed(2)),
    }
  })
}

export function tokenUsageIntensityClassName(intensity: TokenUsageDayPoint['intensity']) {
  switch (intensity) {
    case 4:
      return 'bg-emerald-500/95'
    case 3:
      return 'bg-emerald-500/70'
    case 2:
      return 'bg-emerald-500/45'
    case 1:
      return 'bg-emerald-500/20'
    default:
      return 'bg-muted/30'
  }
}

export function formatTokenUsageTooltip(day: TokenUsageDayPoint) {
  return `${day.dayLabel}: ${formatCount(day.totalTokens)} tokens, ${formatCount(day.inputTokens)} in, ${formatCount(day.outputTokens)} out, ${day.finalizedRunCount} runs`
}

async function loadTokenUsage(
  load: (
    query: ReturnType<typeof buildTokenUsageQuery>,
    opts?: { signal?: AbortSignal },
  ) => Promise<TokenUsageResponse>,
  rangeDays: TokenUsageRange,
  opts?: { signal?: AbortSignal; now?: Date },
) {
  const query = buildTokenUsageQuery(rangeDays, opts?.now)
  const payload = await load(query, opts?.signal ? { signal: opts.signal } : undefined)
  return mapTokenUsage(payload, rangeDays, opts?.now)
}

function mapPeakDay(
  payload: TokenUsageResponse,
  days: TokenUsageDayPoint[],
): TokenUsagePeak | null {
  const payloadPeakDay = payload.summary?.peak_day
  if (payloadPeakDay?.date) {
    const date = parseUTCDate(payloadPeakDay.date)
    return {
      date: payloadPeakDay.date,
      dayLabel: longDateFormatter.format(date),
      totalTokens: payloadPeakDay.total_tokens ?? 0,
    }
  }

  const computedPeakDay = days.reduce<TokenUsageDayPoint | null>((current, day) => {
    if (day.totalTokens <= 0) return current
    if (!current || day.totalTokens > current.totalTokens) return day
    return current
  }, null)

  return computedPeakDay
    ? {
        date: computedPeakDay.date,
        dayLabel: computedPeakDay.dayLabel,
        totalTokens: computedPeakDay.totalTokens,
      }
    : null
}

function buildCalendarCells(days: TokenUsageDayPoint[]) {
  if (days.length === 0) return []

  const firstDay = parseUTCDate(days[0].date)
  const lastDay = parseUTCDate(days[days.length - 1].date)
  const leadingPadding = firstDay.getUTCDay()
  const trailingPadding = 6 - lastDay.getUTCDay()

  return [
    ...Array.from({ length: leadingPadding }, () => null),
    ...days,
    ...Array.from({ length: trailingPadding }, () => null),
  ]
}

function classifyIntensity(
  totalTokens: number,
  maxDailyTokens: number,
): TokenUsageDayPoint['intensity'] {
  if (totalTokens <= 0 || maxDailyTokens <= 0) return 0

  const ratio = totalTokens / maxDailyTokens
  if (ratio >= 0.75) return 4
  if (ratio >= 0.5) return 3
  if (ratio >= 0.25) return 2
  return 1
}

function parseUTCDate(value: string) {
  return new Date(`${value}T00:00:00Z`)
}

function startOfUTCDay(value: Date) {
  return new Date(Date.UTC(value.getUTCFullYear(), value.getUTCMonth(), value.getUTCDate()))
}

function addUTCDays(value: Date, deltaDays: number) {
  return new Date(
    Date.UTC(value.getUTCFullYear(), value.getUTCMonth(), value.getUTCDate() + deltaDays),
  )
}

function toUTCDateString(value: Date) {
  const month = `${value.getUTCMonth() + 1}`.padStart(2, '0')
  const day = `${value.getUTCDate()}`.padStart(2, '0')
  return `${value.getUTCFullYear()}-${month}-${day}`
}
