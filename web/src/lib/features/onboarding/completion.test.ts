import { beforeEach, describe, expect, it } from 'vitest'

import { markProjectOnboardingCompleted, readProjectOnboardingCompletion } from './completion'

describe('project onboarding completion storage', () => {
  beforeEach(() => {
    window.localStorage.clear()
  })

  it('stores completion per project', () => {
    expect(readProjectOnboardingCompletion('project-1')).toBe(false)

    markProjectOnboardingCompleted('project-1')

    expect(readProjectOnboardingCompletion('project-1')).toBe(true)
    expect(readProjectOnboardingCompletion('project-2')).toBe(false)
  })

  it('ignores empty project ids', () => {
    markProjectOnboardingCompleted('   ')

    expect(readProjectOnboardingCompletion('   ')).toBe(false)
  })
})
