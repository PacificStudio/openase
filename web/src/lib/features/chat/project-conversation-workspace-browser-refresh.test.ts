import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

const {
  checkoutProjectConversationWorkspaceBranch,
  commitProjectConversationWorkspace,
  createProjectConversationWorkspaceFile,
  discardProjectConversationWorkspaceFile,
  deleteProjectConversationWorkspaceFile,
  getProjectConversationWorkspaceGitGraph,
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceRepoRefs,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
  renameProjectConversationWorkspaceFile,
  searchProjectConversationWorkspacePaths,
  stageProjectConversationWorkspaceFile,
  syncProjectConversationWorkspace,
} = vi.hoisted(() => ({
  checkoutProjectConversationWorkspaceBranch: vi.fn(),
  commitProjectConversationWorkspace: vi.fn(),
  createProjectConversationWorkspaceFile: vi.fn(),
  discardProjectConversationWorkspaceFile: vi.fn(),
  deleteProjectConversationWorkspaceFile: vi.fn(),
  getProjectConversationWorkspaceGitGraph: vi.fn(),
  getProjectConversationWorkspace: vi.fn(),
  getProjectConversationWorkspaceRepoRefs: vi.fn(),
  getProjectConversationWorkspaceFilePatch: vi.fn(),
  getProjectConversationWorkspaceFilePreview: vi.fn(),
  listProjectConversationWorkspaceTree: vi.fn(),
  renameProjectConversationWorkspaceFile: vi.fn(),
  searchProjectConversationWorkspacePaths: vi.fn(),
  stageProjectConversationWorkspaceFile: vi.fn(),
  syncProjectConversationWorkspace: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  checkoutProjectConversationWorkspaceBranch,
  commitProjectConversationWorkspace,
  createProjectConversationWorkspaceFile,
  discardProjectConversationWorkspaceFile,
  deleteProjectConversationWorkspaceFile,
  getProjectConversationWorkspaceGitGraph,
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceRepoRefs,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
  renameProjectConversationWorkspaceFile,
  searchProjectConversationWorkspacePaths,
  stageProjectConversationWorkspaceFile,
  syncProjectConversationWorkspace,
}))

vi.mock('@xterm/xterm', () => ({
  Terminal: class {
    cols = 96
    rows = 28
    loadAddon() {}
    open() {}
    focus() {}
    clear() {}
    reset() {}
    dispose() {}
    write() {}
    onData() {
      return { dispose() {} }
    }
    onResize() {
      return { dispose() {} }
    }
  },
}))

vi.mock('@xterm/addon-fit', () => ({
  FitAddon: class {
    fit() {}
  },
}))

vi.mock('@xterm/xterm/css/xterm.css', () => ({}))
import ProjectConversationWorkspaceBrowser from './project-conversation-workspace-browser.svelte'
import {
  deferredPromise,
  ensureResizeObserver,
  mockWorkspaceMetadata,
  workspaceMetadata,
  workspaceDiff,
} from './project-conversation-workspace-browser.test-helpers'

describe('ProjectConversationWorkspaceBrowser', () => {
  beforeAll(() => {
    ensureResizeObserver()
  })

  beforeEach(() => {
    mockGitContext()
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  function mockGitContext() {
    getProjectConversationWorkspaceRepoRefs.mockResolvedValue({
      repoRefs: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        currentRef: workspaceMetadata.repos[0].currentRef,
        localBranches: [],
        remoteBranches: [],
      },
    })
    getProjectConversationWorkspaceGitGraph.mockResolvedValue({
      gitGraph: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        limit: 40,
        commits: [],
      },
    })
    checkoutProjectConversationWorkspaceBranch.mockResolvedValue({
      checkout: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        currentRef: workspaceMetadata.repos[0].currentRef,
        createdLocalBranch: '',
      },
    })
  }

  it('shows a sync prompt when repo bindings changed and refreshes the browser after syncing', async () => {
    mockWorkspaceMetadata(getProjectConversationWorkspace)
    listProjectConversationWorkspaceTree.mockResolvedValue({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: '',
        entries: [],
      },
    })
    syncProjectConversationWorkspace.mockResolvedValue({
      workspace: structuredClone(workspaceMetadata),
    })

    const syncedMetadata = structuredClone(workspaceMetadata)
    getProjectConversationWorkspace
      .mockResolvedValueOnce({ workspace: structuredClone(workspaceMetadata) })
      .mockResolvedValueOnce({ workspace: syncedMetadata })

    const view = render(ProjectConversationWorkspaceBrowser, {
      props: {
        conversationId: 'conversation-1',
        workspaceDiff: {
          ...workspaceDiff,
          syncPrompt: {
            reason: 'repo_binding_changed',
            missingRepos: [{ name: 'docs', path: 'docs' }],
          },
        },
        workspaceDiffLoading: false,
      },
    })
    await fireEvent.click(await view.findByRole('button', { name: /Sync Repos/i }))
    await waitFor(() => {
      expect(syncProjectConversationWorkspace).toHaveBeenCalledWith('conversation-1')
      expect(getProjectConversationWorkspace).toHaveBeenCalledTimes(2)
    })
  })

  it('preserves the expanded directory and selected file when the toolbar refresh is clicked', async () => {
    mockWorkspaceMetadata(getProjectConversationWorkspace)
    listProjectConversationWorkspaceTree.mockImplementation(async (_conversationId, input) => ({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: input.path,
        entries:
          input.path === ''
            ? [{ path: 'src', name: 'src', kind: 'directory', sizeBytes: 0 }]
            : [{ path: 'src/main.ts', name: 'main.ts', kind: 'file', sizeBytes: 42 }],
      },
    }))
    getProjectConversationWorkspaceFilePreview.mockResolvedValue({
      filePreview: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'src/main.ts',
        sizeBytes: 42,
        mediaType: 'text/plain',
        previewKind: 'text',
        truncated: false,
        content: 'export const refreshed = true;\n',
      },
    })
    getProjectConversationWorkspaceFilePatch.mockResolvedValue({
      filePatch: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'src/main.ts',
        status: 'modified',
        diffKind: 'text',
        truncated: false,
        diff: '@@ -1 +1 @@\n+export const refreshed = true;\n',
      },
    })

    const view = render(ProjectConversationWorkspaceBrowser, {
      props: {
        conversationId: 'conversation-1',
        workspaceDiff,
        workspaceDiffLoading: false,
      },
    })

    await fireEvent.click(await view.findByRole('button', { name: 'src' }, { timeout: 3000 }))
    await fireEvent.click(await view.findByRole('button', { name: 'main.ts' }, { timeout: 3000 }))
    await waitFor(() => {
      expect(getProjectConversationWorkspaceFilePreview).toHaveBeenCalledTimes(1)
      expect(getProjectConversationWorkspaceFilePatch).toHaveBeenCalledTimes(1)
    })
    await fireEvent.click(view.getByRole('button', { name: 'Refresh Workspace Browser' }))
    await waitFor(() => {
      expect(getProjectConversationWorkspace).toHaveBeenCalledTimes(2)
      expect(listProjectConversationWorkspaceTree).toHaveBeenCalledWith('conversation-1', {
        repoPath: 'services/openase',
        path: '',
      })
      expect(listProjectConversationWorkspaceTree).toHaveBeenCalledWith('conversation-1', {
        repoPath: 'services/openase',
        path: 'src',
      })
      expect(getProjectConversationWorkspaceFilePreview).toHaveBeenCalledTimes(2)
      expect(getProjectConversationWorkspaceFilePatch).toHaveBeenCalledTimes(2)
    })
    expect(view.container.textContent).toContain('main.ts')
    await waitFor(() => {
      expect(view.container.textContent).toContain('export const refreshed = true;')
    })
  })

  it('keeps tree and file DOM stable when a refresh returns identical data', async () => {
    mockWorkspaceMetadata(getProjectConversationWorkspace)
    listProjectConversationWorkspaceTree.mockImplementation(async (_conversationId, input) => ({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: input.path,
        entries:
          input.path === ''
            ? [{ path: 'src', name: 'src', kind: 'directory', sizeBytes: 0 }]
            : [{ path: 'src/main.ts', name: 'main.ts', kind: 'file', sizeBytes: 42 }],
      },
    }))
    getProjectConversationWorkspaceFilePreview.mockResolvedValue({
      filePreview: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'src/main.ts',
        sizeBytes: 42,
        mediaType: 'text/plain',
        previewKind: 'text',
        truncated: false,
        content: 'export const stable = true;\n',
      },
    })
    getProjectConversationWorkspaceFilePatch.mockResolvedValue({
      filePatch: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'src/main.ts',
        status: 'modified',
        diffKind: 'text',
        truncated: false,
        diff: '@@ -1 +1 @@\n+export const stable = true;\n',
      },
    })

    const view = render(ProjectConversationWorkspaceBrowser, {
      props: {
        conversationId: 'conversation-1',
        workspaceDiff,
        workspaceDiffLoading: false,
      },
    })

    await fireEvent.click(await view.findByRole('button', { name: 'src' }, { timeout: 3000 }))
    const mainFileButton = await view.findByRole('button', { name: 'main.ts' }, { timeout: 3000 })
    await fireEvent.click(mainFileButton)
    await waitFor(() => {
      expect(view.container.textContent).toContain('export const stable = true;')
    })
    await fireEvent.click(view.getByRole('button', { name: 'Refresh Workspace Browser' }))
    await waitFor(() => {
      expect(getProjectConversationWorkspace).toHaveBeenCalledTimes(2)
      expect(getProjectConversationWorkspaceFilePreview).toHaveBeenCalledTimes(2)
      expect(getProjectConversationWorkspaceFilePatch).toHaveBeenCalledTimes(2)
    })
    expect(view.getByRole('button', { name: 'main.ts' })).toBe(mainFileButton)
    expect(view.container.textContent).toContain('export const stable = true;')
    expect(view.container.textContent).not.toContain('Loading files…')
  })

  it('does not show extra tree or file loading chrome during a same-data refresh', async () => {
    mockWorkspaceMetadata(getProjectConversationWorkspace)

    const refreshDir = deferredPromise<{
      workspaceTree: {
        conversationId: string
        repoPath: string
        path: string
        entries: Array<{ path: string; name: string; kind: 'file'; sizeBytes: number }>
      }
    }>()
    const refreshPreview = deferredPromise<{
      filePreview: {
        conversationId: string
        repoPath: string
        path: string
        sizeBytes: number
        mediaType: string
        previewKind: 'text'
        truncated: boolean
        content: string
      }
    }>()
    const refreshPatch = deferredPromise<{
      filePatch: {
        conversationId: string
        repoPath: string
        path: string
        status: 'modified'
        diffKind: 'text'
        truncated: boolean
        diff: string
      }
    }>()

    let srcLoads = 0
    listProjectConversationWorkspaceTree.mockImplementation(async (_conversationId, input) => {
      if (input.path === '') {
        return {
          workspaceTree: {
            conversationId: 'conversation-1',
            repoPath: 'services/openase',
            path: '',
            entries: [{ path: 'src', name: 'src', kind: 'directory', sizeBytes: 0 }],
          },
        }
      }

      srcLoads += 1
      if (srcLoads === 1) {
        return {
          workspaceTree: {
            conversationId: 'conversation-1',
            repoPath: 'services/openase',
            path: 'src',
            entries: [{ path: 'src/main.ts', name: 'main.ts', kind: 'file', sizeBytes: 42 }],
          },
        }
      }
      return refreshDir.promise
    })

    let previewLoads = 0
    getProjectConversationWorkspaceFilePreview.mockImplementation(async () => {
      previewLoads += 1
      if (previewLoads === 1) {
        return {
          filePreview: {
            conversationId: 'conversation-1',
            repoPath: 'services/openase',
            path: 'src/main.ts',
            sizeBytes: 42,
            mediaType: 'text/plain',
            previewKind: 'text',
            truncated: false,
            content: 'export const stable = true;\n',
          },
        }
      }
      return refreshPreview.promise
    })

    let patchLoads = 0
    getProjectConversationWorkspaceFilePatch.mockImplementation(async () => {
      patchLoads += 1
      if (patchLoads === 1) {
        return {
          filePatch: {
            conversationId: 'conversation-1',
            repoPath: 'services/openase',
            path: 'src/main.ts',
            status: 'modified',
            diffKind: 'text',
            truncated: false,
            diff: '@@ -1 +1 @@\n+export const stable = true;\n',
          },
        }
      }
      return refreshPatch.promise
    })

    const view = render(ProjectConversationWorkspaceBrowser, {
      props: {
        conversationId: 'conversation-1',
        workspaceDiff,
        workspaceDiffLoading: false,
      },
    })

    await fireEvent.click(await view.findByRole('button', { name: 'src' }, { timeout: 3000 }))
    const mainFileButton = await view.findByRole('button', { name: 'main.ts' }, { timeout: 3000 })
    await fireEvent.click(mainFileButton)
    await waitFor(() => {
      expect(view.container.textContent).toContain('export const stable = true;')
    })
    await fireEvent.click(view.getByRole('button', { name: 'Refresh Workspace Browser' }))
    await waitFor(() =>
      expect(listProjectConversationWorkspaceTree).toHaveBeenCalledWith('conversation-1', {
        repoPath: 'services/openase',
        path: 'src',
      }),
    )
    expect(view.getByRole('button', { name: 'main.ts' })).toBe(mainFileButton)
    expect(view.container.textContent).toContain('export const stable = true;')
    expect(view.container.textContent).not.toContain('Loading files…')
    expect(view.container.textContent).not.toContain('Loading…')
    expect(view.container.querySelectorAll('.animate-spin')).toHaveLength(1)

    refreshDir.resolve({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'src',
        entries: [{ path: 'src/main.ts', name: 'main.ts', kind: 'file', sizeBytes: 42 }],
      },
    })

    await waitFor(() => {
      expect(getProjectConversationWorkspaceFilePreview).toHaveBeenCalledTimes(2)
      expect(getProjectConversationWorkspaceFilePatch).toHaveBeenCalledTimes(2)
    })
    expect(view.getByRole('button', { name: 'main.ts' })).toBe(mainFileButton)
    expect(view.container.textContent).toContain('export const stable = true;')
    expect(view.container.textContent).not.toContain('Loading…')
    expect(view.container.querySelectorAll('.animate-spin')).toHaveLength(1)

    refreshPreview.resolve({
      filePreview: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'src/main.ts',
        sizeBytes: 42,
        mediaType: 'text/plain',
        previewKind: 'text',
        truncated: false,
        content: 'export const stable = true;\n',
      },
    })
    refreshPatch.resolve({
      filePatch: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'src/main.ts',
        status: 'modified',
        diffKind: 'text',
        truncated: false,
        diff: '@@ -1 +1 @@\n+export const stable = true;\n',
      },
    })

    await waitFor(() => expect(getProjectConversationWorkspace).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(view.container.querySelectorAll('.animate-spin')).toHaveLength(0))
    expect(view.getByRole('button', { name: 'main.ts' })).toBe(mainFileButton)
  })

  it('ignores stale file preview responses when the user selects a newer file', async () => {
    mockWorkspaceMetadata(getProjectConversationWorkspace)
    listProjectConversationWorkspaceTree.mockResolvedValue({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: '',
        entries: [
          { path: 'README.md', name: 'README.md', kind: 'file', sizeBytes: 31 },
          { path: 'package.json', name: 'package.json', kind: 'file', sizeBytes: 48 },
        ],
      },
    })

    const readmePreview = deferredPromise<{
      filePreview: {
        conversationId: string
        repoPath: string
        path: string
        sizeBytes: number
        mediaType: string
        previewKind: 'text'
        truncated: boolean
        content: string
      }
    }>()
    const readmePatch = deferredPromise<{
      filePatch: {
        conversationId: string
        repoPath: string
        path: string
        status: 'modified'
        diffKind: 'binary'
        truncated: boolean
        diff: string
      }
    }>()

    getProjectConversationWorkspaceFilePreview.mockImplementation(
      async (_conversationId, input) => {
        if (input.path === 'README.md') {
          return readmePreview.promise
        }
        return {
          filePreview: {
            conversationId: 'conversation-1',
            repoPath: 'services/openase',
            path: 'package.json',
            sizeBytes: 48,
            mediaType: 'application/json',
            previewKind: 'text',
            truncated: false,
            content: '{"name":"latest"}\n',
          },
        }
      },
    )
    getProjectConversationWorkspaceFilePatch.mockImplementation(async (_conversationId, input) => {
      if (input.path === 'README.md') {
        return readmePatch.promise
      }
      return {
        filePatch: {
          conversationId: 'conversation-1',
          repoPath: 'services/openase',
          path: 'package.json',
          status: 'modified',
          diffKind: 'binary',
          truncated: false,
          diff: '',
        },
      }
    })

    const view = render(ProjectConversationWorkspaceBrowser, {
      props: {
        conversationId: 'conversation-1',
        workspaceDiff: {
          ...workspaceDiff,
          repos: [{ ...workspaceDiff.repos[0], files: [] }],
        },
        workspaceDiffLoading: false,
      },
    })

    await fireEvent.click(
      await view.findByRole('button', { name: /README\.md/ }, { timeout: 3000 }),
    )
    await fireEvent.click(view.getByRole('button', { name: 'package.json' }))

    await waitFor(() => {
      expect(getProjectConversationWorkspaceFilePreview).toHaveBeenCalledWith('conversation-1', {
        repoPath: 'services/openase',
        path: 'package.json',
      })
      expect(view.container.textContent).toContain('package.json')
      expect(view.container.textContent).toContain('{"name":"latest"}')
    })

    readmePreview.resolve({
      filePreview: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        sizeBytes: 31,
        mediaType: 'text/plain',
        previewKind: 'text',
        truncated: false,
        content: 'stale readme\n',
      },
    })
    readmePatch.resolve({
      filePatch: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        status: 'modified',
        diffKind: 'binary',
        truncated: false,
        diff: '',
      },
    })
    await Promise.all([readmePreview.promise, readmePatch.promise])
    await Promise.resolve()

    expect(view.container.textContent).toContain('package.json')
    expect(view.container.textContent).toContain('{"name":"latest"}')
    expect(view.container.textContent).not.toContain('stale readme')
  })
})
