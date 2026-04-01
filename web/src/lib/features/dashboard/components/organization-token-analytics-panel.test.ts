import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it } from 'vitest'

import type { OrganizationTokenUsageAnalytics } from '../types'
import OrganizationTokenAnalyticsPanel from './organization-token-analytics-panel.svelte'

describe('OrganizationTokenAnalyticsPanel', () => {
  afterEach(() => {
    cleanup()
  })

  it('renders trend and calendar summaries for organization token snapshots', () => {
    const analytics: OrganizationTokenUsageAnalytics = {
      rangeDays: 30,
      days: [
        {
          date: '2026-03-30',
          dayLabel: 'Mon, Mar 30',
          shortLabel: 'Mar 30',
          inputTokens: 90,
          outputTokens: 30,
          cachedInputTokens: 12,
          reasoningTokens: 4,
          totalTokens: 120,
          finalizedRunCount: 2,
          intensity: 3,
        },
        {
          date: '2026-03-31',
          dayLabel: 'Tue, Mar 31',
          shortLabel: 'Mar 31',
          inputTokens: 150,
          outputTokens: 50,
          cachedInputTokens: 24,
          reasoningTokens: 8,
          totalTokens: 200,
          finalizedRunCount: 3,
          intensity: 4,
        },
      ],
      calendarCells: [
        null,
        {
          date: '2026-03-30',
          dayLabel: 'Mon, Mar 30',
          shortLabel: 'Mar 30',
          inputTokens: 90,
          outputTokens: 30,
          cachedInputTokens: 12,
          reasoningTokens: 4,
          totalTokens: 120,
          finalizedRunCount: 2,
          intensity: 3,
        },
        {
          date: '2026-03-31',
          dayLabel: 'Tue, Mar 31',
          shortLabel: 'Mar 31',
          inputTokens: 150,
          outputTokens: 50,
          cachedInputTokens: 24,
          reasoningTokens: 8,
          totalTokens: 200,
          finalizedRunCount: 3,
          intensity: 4,
        },
      ],
      totalTokens: 320,
      avgDailyTokens: 160,
      totalRuns: 5,
      peakDay: {
        date: '2026-03-31',
        dayLabel: 'Tue, Mar 31',
        totalTokens: 200,
      },
      maxDailyTokens: 200,
    }

    const { getByText, container } = render(OrganizationTokenAnalyticsPanel, {
      props: {
        analytics,
        selectedRange: 30,
      },
    })

    expect(getByText('Token Usage')).toBeTruthy()
    expect(getByText('Trend')).toBeTruthy()
    expect(getByText('Calendar')).toBeTruthy()
    expect(getByText('320')).toBeTruthy()
    expect(getByText('Tue, Mar 31')).toBeTruthy()
    expect(container.querySelector('svg')).toBeTruthy()
  })
})
