import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import {
  buildSkillEditorData,
  initialContent,
  resetSkillEditorAppStore,
  runbookContent,
  seedSkillEditorAppStore,
} from './skill-editor-page.test-support'

const { loadSkillEditorData } = vi.hoisted(() => ({
  loadSkillEditorData: vi.fn(),
}))

const { closeSkillRefinementSession, streamSkillRefinement } = vi.hoisted(() => ({
  closeSkillRefinementSession: vi.fn(),
  streamSkillRefinement: vi.fn(),
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

vi.mock('$lib/api/skill-refinement', () => ({
  closeSkillRefinementSession,
  streamSkillRefinement,
}))

vi.mock('$lib/stores/toast.svelte', () => ({ toastStore }))

import SkillEditorPage from './skill-editor-page.svelte'

describe('SkillEditorPage refinement flow', () => {
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

  it('applies an AI multi-file suggestion back into the skill bundle editor', async () => {
    seedSkillEditorAppStore()
    loadSkillEditorData.mockResolvedValue(buildSkillEditorData())

    streamSkillRefinement.mockImplementation(async (_request, handlers) => {
      handlers.onEvent({
        kind: 'session',
        payload: {
          sessionId: 'session-skill-1',
          workspacePath: '/tmp/skill-tests/openase/deploy/session-skill-1/workspace',
        },
      })
      handlers.onEvent({
        kind: 'status',
        payload: {
          sessionId: 'session-skill-1',
          phase: 'testing',
          attempt: 1,
          message: 'Codex is running verification commands.',
        },
      })
      handlers.onEvent({
        kind: 'result',
        payload: {
          sessionId: 'session-skill-1',
          status: 'verified',
          workspacePath: '/tmp/skill-tests/openase/deploy/session-skill-1/workspace',
          providerId: 'provider-1',
          providerName: 'Codex',
          attempts: 1,
          transcriptSummary: 'Bundle verified after tightening the deploy instructions.',
          commandOutputSummary: 'bash -n scripts/redeploy.sh\n./scripts/redeploy.sh',
          candidateBundleHash: 'bundle-hash-1',
          candidateFiles: [
            {
              path: 'SKILL.md',
              file_kind: 'entrypoint',
              media_type: 'text/markdown; charset=utf-8',
              encoding: 'utf8',
              is_executable: false,
              size_bytes: initialContent.length + 49,
              sha256: 'sha-entry-verified',
              content: [
                '---',
                'name: "deploy"',
                'description: "Deploy safely"',
                '---',
                '',
                'Use safe steps.',
                '',
                'Verify rollback steps before production deploys.',
              ].join('\n'),
              content_base64: 'ignored',
            },
            {
              path: 'references/runbook.md',
              file_kind: 'reference',
              media_type: 'text/markdown; charset=utf-8',
              encoding: 'utf8',
              is_executable: false,
              size_bytes: runbookContent.length,
              sha256: 'sha-runbook',
              content: runbookContent,
              content_base64: 'ignored',
            },
          ],
        },
      })
    })

    const { container, findByRole, findByPlaceholderText, getByRole } = render(SkillEditorPage, {
      props: { skillId: 'skill-1' },
    })

    let editor: HTMLTextAreaElement | undefined
    await waitFor(() => {
      const candidates = [...container.querySelectorAll('textarea')]
      editor =
        candidates.find(
          (item): item is HTMLTextAreaElement =>
            item instanceof HTMLTextAreaElement && item.value === initialContent,
        ) ?? undefined
      expect(editor).toBeDefined()
    })

    if (!editor) {
      throw new Error('expected skill editor textarea to render')
    }
    const resolvedEditor = editor
    expect(resolvedEditor.value).toBe(initialContent)

    await fireEvent.click(await findByRole('button', { name: 'Fix & verify' }))

    const prompt = await findByPlaceholderText(
      'Describe what Codex should improve and verify for this draft bundle…',
    )
    await fireEvent.input(prompt, { target: { value: 'Make the deploy skill safer.' } })
    await fireEvent.click(await findByRole('button', { name: 'Fix and verify' }))

    await fireEvent.click(await findByRole('button', { name: 'references/runbook.md' }))
    await fireEvent.click(await findByRole('button', { name: 'Apply All' }))

    const saveButton = getByRole('button', { name: 'Save' }) as HTMLButtonElement
    await waitFor(() => {
      expect(resolvedEditor.value).toBe(runbookContent)
      expect(saveButton.disabled).toBe(false)
    })

    expect(streamSkillRefinement).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: 'project-1',
        skillId: 'skill-1',
        providerId: 'provider-1',
        message: 'Make the deploy skill safer.',
      }),
      expect.any(Object),
    )
    expect(closeSkillRefinementSession).not.toHaveBeenCalled()
  })
})
