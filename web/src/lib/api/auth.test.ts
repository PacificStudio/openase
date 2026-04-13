import { describe, expect, it } from 'vitest'

import { normalizeReturnTo } from './auth'

describe('normalizeReturnTo', () => {
  it('preserves same-origin app paths', () => {
    expect(normalizeReturnTo('/orgs/org-1/projects/project-1/settings?tab=auth#sessions')).toBe(
      '/orgs/org-1/projects/project-1/settings?tab=auth#sessions',
    )
  })

  it('rejects external origins and protocol-relative targets', () => {
    expect(normalizeReturnTo('https://evil.example/phish')).toBe('/')
    expect(normalizeReturnTo('//evil.example/phish')).toBe('/')
  })

  it('falls back for malformed paths', () => {
    expect(normalizeReturnTo('http://%')).toBe('/')
    expect(normalizeReturnTo('   ')).toBe('/')
    expect(normalizeReturnTo(null)).toBe('/')
  })
})
