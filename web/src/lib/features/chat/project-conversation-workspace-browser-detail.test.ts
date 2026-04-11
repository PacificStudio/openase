import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { EDITOR_WRAP_MODE_STORAGE_KEY } from '$lib/components/code/wrap-mode'

import ProjectConversationWorkspaceBrowserDetail from './project-conversation-workspace-browser-detail.svelte'

describe('ProjectConversationWorkspaceBrowserDetail', () => {
  beforeEach(() => {
    window.localStorage.clear()
  })

  afterEach(() => {
    cleanup()
  })

  it('shows a wrap toggle for text previews and persists the selected mode', async () => {
    const preview = {
      conversationId: 'conversation-1',
      repoPath: 'services/openase',
      path: 'README.md',
      sizeBytes: 64,
      mediaType: 'text/plain',
      previewKind: 'text' as const,
      truncated: false,
      content: 'alpha '.repeat(60),
    }

    const firstView = render(ProjectConversationWorkspaceBrowserDetail, {
      props: {
        selectedRepo: {
          name: 'openase',
          path: 'services/openase',
          branch: 'feat/ase-168-wrap-toggle',
          headCommit: '123456789abc',
          headSummary: 'Support editor wrap toggle',
          dirty: true,
          filesChanged: 1,
          added: 1,
          removed: 0,
        },
        selectedFilePath: 'README.md',
        preview,
        patch: null,
      },
    })

    await waitFor(() => expect(firstView.container.querySelector('.cm-editor')).not.toBeNull())

    const wrapToggle = firstView.getByTestId('workspace-browser-wrap-toggle')
    expect(wrapToggle.textContent).toContain('Wrap on')
    expect(firstView.container.querySelector('.cm-lineWrapping')).not.toBeNull()

    await fireEvent.click(wrapToggle)

    await waitFor(() => expect(firstView.container.querySelector('.cm-lineWrapping')).toBeNull())
    expect(window.localStorage.getItem(EDITOR_WRAP_MODE_STORAGE_KEY)).toBe('nowrap')
    expect(wrapToggle.textContent).toContain('Wrap off')

    firstView.unmount()

    const secondView = render(ProjectConversationWorkspaceBrowserDetail, {
      props: {
        selectedRepo: {
          name: 'openase',
          path: 'services/openase',
          branch: 'feat/ase-168-wrap-toggle',
          headCommit: '123456789abc',
          headSummary: 'Support editor wrap toggle',
          dirty: true,
          filesChanged: 1,
          added: 1,
          removed: 0,
        },
        selectedFilePath: 'README.md',
        preview,
        patch: null,
      },
    })

    await waitFor(() => expect(secondView.container.querySelector('.cm-editor')).not.toBeNull())
    expect(secondView.getByTestId('workspace-browser-wrap-toggle').textContent).toContain(
      'Wrap off',
    )
    expect(secondView.container.querySelector('.cm-lineWrapping')).toBeNull()
  })
})
