import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { EDITOR_WRAP_MODE_STORAGE_KEY } from '$lib/components/code/wrap-mode'
import type { ProjectConversationWorkspaceBrowserState } from './project-conversation-workspace-browser-state.svelte'
import type { WorkspaceFileEditorState } from './project-conversation-workspace-browser-state-helpers'
import ProjectConversationWorkspaceBrowserDetail from './project-conversation-workspace-browser-detail.svelte'

function buildBrowserStub(): ProjectConversationWorkspaceBrowserState {
  const editorState: WorkspaceFileEditorState = {
    baseSavedContent: 'alpha '.repeat(60),
    baseSavedRevision: 'rev-1',
    latestSavedContent: 'alpha '.repeat(60),
    latestSavedRevision: 'rev-1',
    draftContent: 'alpha '.repeat(60),
    dirty: false,
    savePhase: 'idle',
    externalChange: false,
    errorMessage: '',
    encoding: 'utf-8',
    lineEnding: 'lf',
    lastSavedAt: '',
  }

  return {
    openTabs: [{ repoPath: 'services/openase', filePath: 'README.md' }],
    activeTabKey: 'services/openase::README.md',
    preview: {
      conversationId: 'conversation-1',
      repoPath: 'services/openase',
      path: 'README.md',
      sizeBytes: 64,
      mediaType: 'text/plain',
      previewKind: 'text' as const,
      truncated: false,
      content: 'alpha '.repeat(60),
      revision: 'rev-1',
      writable: true,
      readOnlyReason: '',
      encoding: 'utf-8' as const,
      lineEnding: 'lf' as const,
    },
    patch: null,
    fileLoading: false,
    fileError: '',
    selectedEditorState: editorState,
    selectedDraftLineDiff: { added: [], modified: [], deletionAbove: [], deletionAtEnd: false },
    getEditorState: () => editorState,
    discardDraft: () => {},
    closeTab: () => {},
    activateTab: () => {},
    saveFile: async () => true,
    saveSelectedFile: async () => true,
    revertSelectedDraft: () => {},
    reloadSelectedSavedVersion: () => {},
    keepSelectedDraft: () => {},
    updateSelectedDraft: () => {},
  } as unknown as ProjectConversationWorkspaceBrowserState
}

describe('ProjectConversationWorkspaceBrowserDetail', () => {
  beforeEach(() => {
    window.localStorage.clear()
  })

  afterEach(() => {
    cleanup()
  })

  it('shows a wrap toggle for text previews and persists the selected mode', async () => {
    const firstView = render(ProjectConversationWorkspaceBrowserDetail, {
      props: {
        browser: buildBrowserStub(),
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
      },
    })

    await waitFor(() => expect(firstView.container.querySelector('.cm-editor')).not.toBeNull())

    const wrapToggle = firstView.getByTestId('workspace-browser-wrap-toggle')
    expect(wrapToggle.getAttribute('aria-pressed')).toBe('true')
    expect(wrapToggle.getAttribute('aria-label')).toBe('Disable line wrap')
    expect(firstView.container.querySelector('.cm-lineWrapping')).not.toBeNull()

    await fireEvent.click(wrapToggle)

    await waitFor(() => expect(firstView.container.querySelector('.cm-lineWrapping')).toBeNull())
    expect(window.localStorage.getItem(EDITOR_WRAP_MODE_STORAGE_KEY)).toBe('nowrap')
    expect(wrapToggle.getAttribute('aria-pressed')).toBe('false')
    expect(wrapToggle.getAttribute('aria-label')).toBe('Enable line wrap')

    firstView.unmount()

    const secondView = render(ProjectConversationWorkspaceBrowserDetail, {
      props: {
        browser: buildBrowserStub(),
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
      },
    })

    await waitFor(() => expect(secondView.container.querySelector('.cm-editor')).not.toBeNull())
    const persistedToggle = secondView.getByTestId('workspace-browser-wrap-toggle')
    expect(persistedToggle.getAttribute('aria-pressed')).toBe('false')
    expect(persistedToggle.getAttribute('aria-label')).toBe('Enable line wrap')
    expect(secondView.container.querySelector('.cm-lineWrapping')).toBeNull()
  })
})
