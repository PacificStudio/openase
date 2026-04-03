import { describe, expect, it, vi } from 'vitest'

import { isAppContextFresh } from './project-shell-state'

describe('project shell state', () => {
  it('treats a recent successful workspace app-context fetch as fresh even when it returned zero organizations', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-03T12:00:00Z'))

    expect(isAppContextFresh('none::', 'none::', Date.now() - 5_000)).toBe(true)

    vi.useRealTimers()
  })

  it('treats app-context as stale after the freshness window expires', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-03T12:00:00Z'))

    expect(isAppContextFresh('none::', 'none::', Date.now() - 31_000)).toBe(false)
    expect(isAppContextFresh('none::', 'org:org-1:', Date.now() - 5_000)).toBe(false)

    vi.useRealTimers()
  })
})
