import { cleanup, render } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import {
  buildSkillEditorData,
  resetSkillEditorAppStore,
  seedSkillEditorAppStore,
} from './skill-editor-page.test-support'

const { loadSkillEditorData } = vi.hoisted(() => ({
  loadSkillEditorData: vi.fn(),
}))

const { goto } = vi.hoisted(() => ({
  goto: vi.fn(),
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
  },
}))

vi.mock('$app/navigation', () => ({
  goto,
  beforeNavigate: vi.fn(),
}))

vi.mock('./skill-editor-page.helpers', async () => {
  const actual = await vi.importActual<typeof import('./skill-editor-page.helpers')>(
    './skill-editor-page.helpers',
  )
  return {
    ...actual,
    loadSkillEditorData,
  }
})

vi.mock('$lib/api/openase', async () => {
  const actual = await vi.importActual<typeof import('$lib/api/openase')>('$lib/api/openase')
  return actual
})

vi.mock('$lib/stores/toast.svelte', () => ({ toastStore }))

import SkillEditorPage from './skill-editor-page.svelte'

describe('SkillEditorPage', () => {
  beforeAll(() => {
    HTMLElement.prototype.scrollIntoView ??= vi.fn()
    HTMLElement.prototype.hasPointerCapture ??= vi.fn(() => false)
    HTMLElement.prototype.releasePointerCapture ??= vi.fn()
    globalThis.ResizeObserver ??= class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  })

  afterEach(() => {
    cleanup()
    resetSkillEditorAppStore()
    vi.clearAllMocks()
  })

  it('keeps the skill editor focused on direct editing without a refinement drawer', async () => {
    seedSkillEditorAppStore()
    loadSkillEditorData.mockResolvedValue(buildSkillEditorData())

    const { findByRole, queryByRole, queryByPlaceholderText } = render(SkillEditorPage, {
      props: { skillId: 'skill-1' },
    })

    expect(await findByRole('button', { name: 'Save' })).toBeTruthy()
    expect(queryByRole('button', { name: 'Fix & verify' })).toBeNull()
    expect(
      queryByPlaceholderText(
        'Describe what Codex should improve and verify for this draft bundle…',
      ),
    ).toBeNull()
  })
})
