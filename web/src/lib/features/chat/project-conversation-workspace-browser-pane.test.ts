import { fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { ProjectConversationWorkspaceBrowserState } from './project-conversation-workspace-browser-state.svelte'
import ProjectConversationWorkspaceBrowserPane from './project-conversation-workspace-browser-pane.svelte'

function buildPaneBrowserStub(): ProjectConversationWorkspaceBrowserState {
  const selectedRepo = {
    name: 'openase',
    path: 'services/openase',
    branch: 'agent/conv-123',
    headCommit: '123456789abc',
    headSummary: 'Add workspace browser scaffolding',
    dirty: true,
    filesChanged: 1,
    added: 2,
    removed: 0,
  }

  return {
    metadata: {
      workspacePath: '/tmp/conversation-1',
      repos: [selectedRepo],
    },
    selectedRepoPath: selectedRepo.path,
    selectedFilePath: 'src/main.ts',
    treeNodes: new Map([
      [
        '',
        [
          { path: 'src', name: 'src', kind: 'directory', sizeBytes: 0 },
          { path: 'README.md', name: 'README.md', kind: 'file', sizeBytes: 64 },
        ],
      ],
      ['src', [{ path: 'src/main.ts', name: 'main.ts', kind: 'file', sizeBytes: 42 }]],
    ]),
    expandedDirs: new Set(['src']),
    loadingDirs: new Set<string>(),
    recentFiles: [{ repoPath: selectedRepo.path, filePath: 'src/main.ts' }],
    openRepo: vi.fn(),
    toggleDir: vi.fn(),
    selectFile: vi.fn(),
    createFile: vi.fn(async () => undefined),
    renameFile: vi.fn(async () => undefined),
    deleteFile: vi.fn(async () => undefined),
    openTabs: [],
    activeTabKey: '',
    preview: null,
    patch: null,
    fileLoading: false,
    fileError: '',
    selectedEditorState: { dirty: true, savePhase: 'idle' },
    selectedDraftLineDiff: { added: [], modified: [], deletionAbove: [], deletionAtEnd: false },
    selectedChangedFiles: [],
    autosaveEnabled: true,
    getEditorState: () => null,
    discardDraft: vi.fn(),
    closeTab: vi.fn(),
    activateTab: vi.fn(),
    saveFile: vi.fn(async () => true),
    saveSelectedFile: vi.fn(async () => true),
    selectPreviousChangedFile: vi.fn(),
    selectNextChangedFile: vi.fn(),
    applySelectedPendingPatch: vi.fn(),
    discardSelectedPendingPatch: vi.fn(),
    revertSelectedDraft: vi.fn(),
    reloadSelectedSavedVersion: vi.fn(),
    keepSelectedDraft: vi.fn(),
    updateSelectedDraft: vi.fn(),
    updateSelectedSelection: vi.fn(),
    formatSelectedDocument: vi.fn(),
    formatSelectedSelection: vi.fn(),
    setAutosaveEnabled: vi.fn(),
  } as unknown as ProjectConversationWorkspaceBrowserState
}

function buildTerminalManagerStub() {
  return {
    panelOpen: false,
    refitAll: vi.fn(),
  }
}

describe('ProjectConversationWorkspaceBrowserPane', () => {
  beforeEach(() => {
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: {
        writeText: vi.fn(async () => undefined),
      },
    })
  })

  it('creates a new folder in the selected file parent and persists it with a .gitkeep sentinel', async () => {
    const browser = buildPaneBrowserStub()
    const selectedRepo = browser.metadata!.repos[0]
    const terminalManager = buildTerminalManagerStub()

    const view = render(ProjectConversationWorkspaceBrowserPane, {
      props: {
        browser,
        selectedRepo,
        selectedRepoDiff: null,
        terminalManager: terminalManager as never,
      },
    })

    await fireEvent.click(view.getByTestId('workspace-browser-new-folder'))

    const input = await view.findByTestId('workspace-browser-inline-input')
    expect(document.activeElement).toBe(input)
    ;(input as HTMLInputElement).value = 'utils'
    await fireEvent.keyDown(input, { key: 'Enter', code: 'Enter' })

    expect(browser.createFile).toHaveBeenCalledWith('src/utils/.gitkeep')
  })

  it('offers folder-specific context menu actions and creates files inline inside that folder', async () => {
    const browser = buildPaneBrowserStub()
    const selectedRepo = browser.metadata!.repos[0]
    const terminalManager = buildTerminalManagerStub()

    const view = render(ProjectConversationWorkspaceBrowserPane, {
      props: {
        browser,
        selectedRepo,
        selectedRepoDiff: null,
        terminalManager: terminalManager as never,
      },
    })

    await fireEvent.contextMenu(view.getByRole('button', { name: 'src' }), {
      clientX: 32,
      clientY: 48,
    })

    const menu = await view.findByTestId('workspace-browser-tree-menu')
    expect(within(menu).getByRole('menuitem', { name: 'New File' })).toBeTruthy()
    expect(within(menu).getByRole('menuitem', { name: 'New Folder' })).toBeTruthy()

    await fireEvent.click(within(menu).getByRole('menuitem', { name: 'New File' }))

    const input = await view.findByTestId('workspace-browser-inline-input')
    expect(document.activeElement).toBe(input)
    ;(input as HTMLInputElement).value = 'index.ts'
    await fireEvent.keyDown(input, { key: 'Enter', code: 'Enter' })

    expect(browser.createFile).toHaveBeenCalledWith('src/index.ts')
  })

  it('copies absolute and relative paths from the file menu without exposing folder-only items', async () => {
    const browser = buildPaneBrowserStub()
    const selectedRepo = browser.metadata!.repos[0]
    const terminalManager = buildTerminalManagerStub()
    const writeText = navigator.clipboard.writeText as ReturnType<typeof vi.fn>

    const view = render(ProjectConversationWorkspaceBrowserPane, {
      props: {
        browser,
        selectedRepo,
        selectedRepoDiff: null,
        terminalManager: terminalManager as never,
      },
    })

    const mainFileButton = view.getByRole('button', { name: 'main.ts' })

    await fireEvent.contextMenu(mainFileButton, { clientX: 40, clientY: 56 })

    const copyPathMenu = await view.findByTestId('workspace-browser-tree-menu')
    expect(within(copyPathMenu).queryByRole('menuitem', { name: 'New File' })).toBeNull()
    expect(within(copyPathMenu).queryByRole('menuitem', { name: 'New Folder' })).toBeNull()

    await fireEvent.click(within(copyPathMenu).getByRole('menuitem', { name: 'Copy Path' }))
    await waitFor(() =>
      expect(writeText).toHaveBeenCalledWith('/tmp/conversation-1/services/openase/src/main.ts'),
    )

    await fireEvent.contextMenu(mainFileButton, { clientX: 48, clientY: 64 })
    const relativeMenu = await view.findByTestId('workspace-browser-tree-menu')
    await fireEvent.click(
      within(relativeMenu).getByRole('menuitem', { name: 'Copy Relative Path' }),
    )
    await waitFor(() => expect(writeText).toHaveBeenCalledWith('services/openase/src/main.ts'))
  })

  it('renames and deletes files through the inline tree actions', async () => {
    const browser = buildPaneBrowserStub()
    const selectedRepo = browser.metadata!.repos[0]
    const terminalManager = buildTerminalManagerStub()
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)

    const view = render(ProjectConversationWorkspaceBrowserPane, {
      props: {
        browser,
        selectedRepo,
        selectedRepoDiff: null,
        terminalManager: terminalManager as never,
      },
    })

    const mainFileButton = view.getByRole('button', { name: 'main.ts' })

    await fireEvent.contextMenu(mainFileButton, { clientX: 56, clientY: 72 })
    const renameMenu = await view.findByTestId('workspace-browser-tree-menu')
    await fireEvent.click(within(renameMenu).getByRole('menuitem', { name: 'Rename…' }))

    const input = await view.findByTestId('workspace-browser-inline-input')
    ;(input as HTMLInputElement).value = 'app.ts'
    await fireEvent.keyDown(input, { key: 'Enter', code: 'Enter' })
    expect(browser.renameFile).toHaveBeenCalledWith('src/main.ts', 'src/app.ts')

    const refreshedMainFileButton = view.getByRole('button', { name: 'main.ts' })
    await fireEvent.contextMenu(refreshedMainFileButton, { clientX: 64, clientY: 80 })
    const deleteMenu = await view.findByTestId('workspace-browser-tree-menu')
    await fireEvent.click(within(deleteMenu).getByRole('menuitem', { name: 'Delete' }))

    expect(confirmSpy).toHaveBeenCalledWith(
      'Delete src/main.ts? Unsaved local draft changes will be discarded.',
    )
    expect(browser.deleteFile).toHaveBeenCalledWith('src/main.ts')
  })
})
