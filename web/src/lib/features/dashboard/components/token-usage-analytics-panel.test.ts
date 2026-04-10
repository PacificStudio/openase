import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { TokenUsageAnalytics } from '../types'
import TokenUsageAnalyticsPanel from './token-usage-analytics-panel.svelte'

vi.mock('$ui/chart', async () => {
  const { default: LineChart } = await import('./test-line-chart.stub.svelte')
  return { LineChart }
})

describe('TokenUsageAnalyticsPanel', () => {
  afterEach(() => {
    cleanup()
  })

  it('renders trend and calendar summaries for shared token snapshots', () => {
    const analytics: TokenUsageAnalytics = {
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

    const { getByText, container } = render(TokenUsageAnalyticsPanel, {
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
    expect(container.querySelector('[data-testid="mock-line-chart"]')).toBeTruthy()
    expect(container.querySelectorAll('[data-testid="token-usage-calendar-cell"]')).toHaveLength(2)
  })
})
