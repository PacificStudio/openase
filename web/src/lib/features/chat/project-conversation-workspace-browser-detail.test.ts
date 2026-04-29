import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { EditorView } from '@codemirror/view'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { EDITOR_WRAP_MODE_STORAGE_KEY } from '$lib/components/code/wrap-mode'
import { appStore } from '$lib/stores/app.svelte'
import type { ProjectConversationWorkspaceBrowserState } from './project-conversation-workspace-browser-state.svelte'
import type { WorkspaceFileEditorState } from './project-conversation-workspace-browser-state-helpers'
import ProjectConversationWorkspaceBrowserDetail from './project-conversation-workspace-browser-detail.svelte'

function buildBrowserStub(
  overrides: Partial<ProjectConversationWorkspaceBrowserState> = {},
): ProjectConversationWorkspaceBrowserState {
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
    selection: null,
    pendingPatch: null,
  }

  const browser = {
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
    selectedChangedFiles: [],
    autosaveEnabled: true,
    getEditorState: () => editorState,
    discardDraft: () => {},
    closeTab: () => {},
    activateTab: () => {},
    saveFile: async () => true,
    saveSelectedFile: async () => true,
    selectPreviousChangedFile: () => {},
    selectNextChangedFile: () => {},
    applySelectedPendingPatch: () => {},
    discardSelectedPendingPatch: () => {},
    revertSelectedDraft: () => {},
    reloadSelectedSavedVersion: () => {},
    keepSelectedDraft: () => {},
    updateSelectedDraft: () => {},
    updateSelectedSelection: () => {},
    formatSelectedDocument: () => {},
    formatSelectedSelection: () => {},
    setAutosaveEnabled: () => {},
  } as unknown as ProjectConversationWorkspaceBrowserState

  return Object.assign(browser, overrides)
}

async function getEditorView(container: HTMLElement): Promise<EditorView> {
  await waitFor(() => expect(container.querySelector('.cm-editor')).not.toBeNull())

  const editorDom = container.querySelector('.cm-editor')
  expect(editorDom).not.toBeNull()

  const editorView = EditorView.findFromDOM(editorDom as HTMLElement)
  expect(editorView).not.toBeNull()
  return editorView as EditorView
}

async function pressEditorShortcut(editorView: EditorView, init: KeyboardEventInit) {
  editorView.focus()
  await fireEvent.keyDown(editorView.contentDOM, {
    bubbles: true,
    cancelable: true,
    ...init,
  })
}

describe('ProjectConversationWorkspaceBrowserDetail', () => {
  beforeEach(() => {
    window.localStorage.clear()
  })

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('shows a wrap toggle for text previews and persists the selected mode', async () => {
    const firstView = render(ProjectConversationWorkspaceBrowserDetail, {
      props: {
        browser: buildBrowserStub(),
        selectedRepo: {
          name: 'openase',
          path: 'services/openase',
          branch: 'feat/ase-168-wrap-toggle',
          currentRef: {
            kind: 'branch',
            displayName: 'feat/ase-168-wrap-toggle',
            cacheKey: 'branch:feat/ase-168-wrap-toggle',
            branchName: 'feat/ase-168-wrap-toggle',
            branchFullName: 'refs/heads/feat/ase-168-wrap-toggle',
            commitId: '123456789abc',
            shortCommitId: '1234567',
            subject: 'Support editor wrap toggle',
          },
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
    expect(wrapToggle.getAttribute('aria-label')).toBe('Chat Disable Line Wrap')
    expect(firstView.container.querySelector('.cm-lineWrapping')).not.toBeNull()

    await fireEvent.click(wrapToggle)

    await waitFor(() => expect(firstView.container.querySelector('.cm-lineWrapping')).toBeNull())
    expect(window.localStorage.getItem(EDITOR_WRAP_MODE_STORAGE_KEY)).toBe('nowrap')
    expect(wrapToggle.getAttribute('aria-pressed')).toBe('false')
    expect(wrapToggle.getAttribute('aria-label')).toBe('Enable Line Wrap')

    firstView.unmount()

    const secondView = render(ProjectConversationWorkspaceBrowserDetail, {
      props: {
        browser: buildBrowserStub(),
        selectedRepo: {
          name: 'openase',
          path: 'services/openase',
          branch: 'feat/ase-168-wrap-toggle',
          currentRef: {
            kind: 'branch',
            displayName: 'feat/ase-168-wrap-toggle',
            cacheKey: 'branch:feat/ase-168-wrap-toggle',
            branchName: 'feat/ase-168-wrap-toggle',
            branchFullName: 'refs/heads/feat/ase-168-wrap-toggle',
            commitId: '123456789abc',
            shortCommitId: '1234567',
            subject: 'Support editor wrap toggle',
          },
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
    expect(persistedToggle.getAttribute('aria-label')).toBe('Enable Line Wrap')
    expect(secondView.container.querySelector('.cm-lineWrapping')).toBeNull()
  })

  it('removes the old toolbar actions and routes editor actions through shortcuts and the context menu', async () => {
    const formatSelectedDocument = vi.fn()
    const formatSelectedSelection = vi.fn()
    const saveSelectedFile = vi.fn(async () => true)
    const revertSelectedDraft = vi.fn()
    const requestProjectAssistant = vi
      .spyOn(appStore, 'requestProjectAssistant')
      .mockImplementation(() => {})
    const browser = buildBrowserStub()
    const dirtyEditorState: WorkspaceFileEditorState = {
      ...(browser.selectedEditorState as WorkspaceFileEditorState),
      dirty: true,
    }

    const view = render(ProjectConversationWorkspaceBrowserDetail, {
      props: {
        browser: buildBrowserStub({
          selectedEditorState: dirtyEditorState,
          formatSelectedDocument,
          formatSelectedSelection,
          saveSelectedFile,
          revertSelectedDraft,
        }),
        selectedRepo: {
          name: 'openase',
          path: 'services/openase',
          branch: 'feat/ase-168-wrap-toggle',
          currentRef: {
            kind: 'branch',
            displayName: 'feat/ase-168-wrap-toggle',
            cacheKey: 'branch:feat/ase-168-wrap-toggle',
            branchName: 'feat/ase-168-wrap-toggle',
            branchFullName: 'refs/heads/feat/ase-168-wrap-toggle',
            commitId: '123456789abc',
            shortCommitId: '1234567',
            subject: 'Support editor wrap toggle',
          },
          headCommit: '123456789abc',
          headSummary: 'Support editor wrap toggle',
          dirty: true,
          filesChanged: 1,
          added: 1,
          removed: 0,
        },
      },
    })

    const editorView = await getEditorView(view.container)
    const editorShell = view.container.querySelector('.code-editor') as HTMLElement

    expect(view.queryByRole('button', { name: 'Format' })).toBeNull()
    expect(view.queryByRole('button', { name: 'Format selection' })).toBeNull()
    expect(view.queryByRole('button', { name: 'Explain selection' })).toBeNull()
    expect(view.queryByRole('button', { name: 'Rewrite selection' })).toBeNull()
    expect(view.queryByRole('button', { name: 'Revert' })).toBeNull()
    expect(view.queryByRole('button', { name: 'Save now' })).toBeNull()

    await pressEditorShortcut(editorView, { key: 's', code: 'KeyS', ctrlKey: true })
    await waitFor(() => expect(saveSelectedFile).toHaveBeenCalledTimes(1))

    await pressEditorShortcut(editorView, { key: 'f', code: 'KeyF', altKey: true, shiftKey: true })
    expect(formatSelectedDocument).toHaveBeenCalledTimes(1)
    expect(formatSelectedSelection).not.toHaveBeenCalled()

    editorView.dispatch({ selection: { anchor: 0, head: 5 } })

    await pressEditorShortcut(editorView, { key: 'f', code: 'KeyF', altKey: true, shiftKey: true })
    expect(formatSelectedSelection).toHaveBeenCalledTimes(1)

    await fireEvent.contextMenu(editorShell, { clientX: 72, clientY: 88 })

    const menu = await view.findByTestId('code-editor-context-menu')
    expect(within(menu).queryByRole('menuitem', { name: 'Format Document' })).toBeNull()
    expect(within(menu).getByRole('menuitem', { name: /^Format selection/ })).toBeTruthy()
    expect(within(menu).getByRole('menuitem', { name: 'Revert file' })).toBeTruthy()
    expect(within(menu).getByRole('menuitem', { name: 'Explain selection' })).toBeTruthy()
    expect(within(menu).getByRole('menuitem', { name: 'Rewrite selection' })).toBeTruthy()

    await fireEvent.click(within(menu).getByRole('menuitem', { name: 'Explain selection' }))
    expect(requestProjectAssistant).toHaveBeenCalledWith('Explain the selected code.')

    editorView.dispatch({ selection: { anchor: 0, head: 5 } })
    await fireEvent.contextMenu(editorShell, { clientX: 80, clientY: 96 })
    const rewriteMenu = await view.findByTestId('code-editor-context-menu')
    await fireEvent.click(within(rewriteMenu).getByRole('menuitem', { name: 'Rewrite selection' }))
    expect(requestProjectAssistant).toHaveBeenCalledWith('Rewrite the selected code.')

    editorView.dispatch({ selection: { anchor: 0, head: 0 } })
    await fireEvent.contextMenu(editorShell, { clientX: 88, clientY: 104 })
    const revertMenu = await view.findByTestId('code-editor-context-menu')
    await fireEvent.click(within(revertMenu).getByRole('menuitem', { name: 'Revert file' }))
    expect(revertSelectedDraft).toHaveBeenCalledTimes(1)
  })

  it('shows diff status markers and totals in the detail status bar for the selected changed file', async () => {
    const view = render(ProjectConversationWorkspaceBrowserDetail, {
      props: {
        browser: buildBrowserStub({
          selectedEditorState: {
            baseSavedContent: 'line one\nline two changed\nline three\n',
            baseSavedRevision: 'rev-2',
            latestSavedContent: 'line one\nline two changed\nline three\n',
            latestSavedRevision: 'rev-2',
            draftContent: 'line one\nline two changed\nline three\n',
            dirty: false,
            savePhase: 'idle',
            externalChange: false,
            errorMessage: '',
            encoding: 'utf-8',
            lineEnding: 'lf',
            lastSavedAt: '',
            selection: null,
            pendingPatch: null,
          },
          patch: {
            conversationId: 'conversation-1',
            repoPath: 'services/openase',
            path: 'README.md',
            status: 'renamed',
            diffKind: 'text',
            truncated: false,
            diff: '@@ -1 +1 @@\n-alpha\n+beta\n',
          },
          selectedChangedFiles: [
            {
              path: 'README.md',
              status: 'renamed',
              added: 3,
              removed: 1,
            },
          ],
        }),
        selectedRepo: {
          name: 'openase',
          path: 'services/openase',
          branch: 'feat/ase-168-wrap-toggle',
          currentRef: {
            kind: 'branch',
            displayName: 'feat/ase-168-wrap-toggle',
            cacheKey: 'branch:feat/ase-168-wrap-toggle',
            branchName: 'feat/ase-168-wrap-toggle',
            branchFullName: 'refs/heads/feat/ase-168-wrap-toggle',
            commitId: '123456789abc',
            shortCommitId: '1234567',
            subject: 'Support editor wrap toggle',
          },
          headCommit: '123456789abc',
          headSummary: 'Support editor wrap toggle',
          dirty: true,
          filesChanged: 1,
          added: 1,
          removed: 0,
        },
      },
    })

    expect(view.getByTestId('workspace-browser-status-badge').textContent).toBe('R')
    expect(view.getByTestId('workspace-browser-status-label').textContent).toBe('renamed')
    expect(view.getByTestId('workspace-browser-status-totals').textContent).toContain('+3 -1')
  })

  it('renders gutter diff markers from the saved workspace patch when the editor draft is clean', async () => {
    const savedContent = 'line one\nline two changed\nline three\n'

    const view = render(ProjectConversationWorkspaceBrowserDetail, {
      props: {
        browser: buildBrowserStub({
          selectedEditorState: {
            baseSavedContent: savedContent,
            baseSavedRevision: 'rev-2',
            latestSavedContent: savedContent,
            latestSavedRevision: 'rev-2',
            draftContent: savedContent,
            dirty: false,
            savePhase: 'idle',
            externalChange: false,
            errorMessage: '',
            encoding: 'utf-8',
            lineEnding: 'lf',
            lastSavedAt: '',
            selection: null,
            pendingPatch: null,
          },
          patch: {
            conversationId: 'conversation-1',
            repoPath: 'services/openase',
            path: 'README.md',
            status: 'modified',
            diffKind: 'text',
            truncated: false,
            diff: '@@ -1,3 +1,3 @@\n line one\n-line two\n+line two changed\n line three\n',
          },
          preview: {
            conversationId: 'conversation-1',
            repoPath: 'services/openase',
            path: 'README.md',
            sizeBytes: 32,
            mediaType: 'text/plain',
            previewKind: 'text',
            truncated: false,
            content: savedContent,
            revision: 'rev-2',
            writable: true,
            readOnlyReason: '',
            encoding: 'utf-8',
            lineEnding: 'lf',
          },
        }),
        selectedRepo: {
          name: 'openase',
          path: 'services/openase',
          branch: 'main',
          currentRef: {
            kind: 'branch',
            displayName: 'main',
            cacheKey: 'branch:main',
            branchName: 'main',
            branchFullName: 'refs/heads/main',
            commitId: '123456789abc',
            shortCommitId: '1234567',
            subject: 'Main branch',
          },
          headCommit: '123456789abc',
          headSummary: 'Main branch',
          dirty: true,
          filesChanged: 1,
          added: 1,
          removed: 0,
        },
      },
    })

    await waitFor(() =>
      expect(view.container.querySelector(".cm-diff-marker[data-kind='modified']")).not.toBeNull(),
    )
  })
})
