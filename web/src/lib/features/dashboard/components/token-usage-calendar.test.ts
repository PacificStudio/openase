import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it } from 'vitest'

import type { TokenUsageDayPoint } from '../types'
import TokenUsageCalendar from './token-usage-calendar.svelte'

describe('TokenUsageCalendar', () => {
  afterEach(() => {
    cleanup()
  })

  it('renders a stable 7-column calendar grid with padded empty cells', () => {
    const days: TokenUsageDayPoint[] = [
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
    ]

    const { getByText, container } = render(TokenUsageCalendar, {
      props: {
        days,
        calendarCells: [null, ...days, null],
        maxDailyTokens: 200,
      },
    })

    expect(getByText('Calendar')).toBeTruthy()
    expect(getByText('Active days')).toBeTruthy()
    expect(getByText('Sun')).toBeTruthy()
    expect(getByText('Sat')).toBeTruthy()
    expect(container.querySelector('[data-testid="token-usage-calendar-grid"]')).toBeTruthy()
    expect(container.querySelectorAll('[data-testid="token-usage-calendar-cell"]')).toHaveLength(2)
  })
})
