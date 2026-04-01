import { describe, expect, it } from 'vitest'

import type { OrganizationTokenUsageResponse } from '$lib/api/contracts'
import {
  buildOrganizationTokenUsageQuery,
  mapOrganizationTokenUsage,
} from './organization-token-usage'

describe('organization token usage mapping', () => {
  it('builds an inclusive UTC date window for the selected range', () => {
    expect(buildOrganizationTokenUsageQuery(30, new Date('2026-04-01T18:45:00Z'))).toEqual({
      from: '2026-03-03',
      to: '2026-04-01',
    })
  })

  it('pads missing days and builds calendar intensity from the API payload', () => {
    const payload: OrganizationTokenUsageResponse = {
      days: [
        {
          date: '2026-03-29',
          input_tokens: 60,
          output_tokens: 20,
          cached_input_tokens: 10,
          reasoning_tokens: 5,
          total_tokens: 80,
          finalized_run_count: 2,
        },
        {
          date: '2026-03-31',
          input_tokens: 180,
          output_tokens: 40,
          cached_input_tokens: 30,
          reasoning_tokens: 10,
          total_tokens: 220,
          finalized_run_count: 4,
        },
        {
          date: '2026-04-01',
          input_tokens: 50,
          output_tokens: 10,
          cached_input_tokens: 0,
          reasoning_tokens: 0,
          total_tokens: 60,
          finalized_run_count: 1,
        },
      ],
      summary: {
        total_tokens: 360,
        avg_daily_tokens: 90,
        peak_day: {
          date: '2026-03-31',
          total_tokens: 220,
        },
      },
    }

    const mapped = mapOrganizationTokenUsage(payload, 7, new Date('2026-04-01T09:00:00Z'))

    expect(mapped.days.map((day) => `${day.date}:${day.totalTokens}:${day.intensity}`)).toEqual([
      '2026-03-26:0:0',
      '2026-03-27:0:0',
      '2026-03-28:0:0',
      '2026-03-29:80:2',
      '2026-03-30:0:0',
      '2026-03-31:220:4',
      '2026-04-01:60:2',
    ])
    expect(mapped.totalTokens).toBe(360)
    expect(mapped.avgDailyTokens).toBe(90)
    expect(mapped.totalRuns).toBe(7)
    expect(mapped.peakDay).toEqual({
      date: '2026-03-31',
      dayLabel: 'Tue, Mar 31',
      totalTokens: 220,
    })
    expect(mapped.calendarCells).toHaveLength(14)
    expect(mapped.calendarCells[0]).toBeNull()
    expect(mapped.calendarCells[4]?.date).toBe('2026-03-26')
    expect(mapped.calendarCells[11]).toBeNull()
  })
})
