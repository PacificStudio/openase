import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ProjectConversationWorkspaceDiff } from '$lib/api/chat'
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
  it('ignores stale tree responses from the previous repo after the user switches repos', async () => {
    getProjectConversationWorkspace.mockResolvedValue({
      workspace: {
        conversationId: 'conversation-1',
        available: true,
        workspacePath: '/tmp/conversation-1',
        repos: [
          {
            ...workspaceMetadata.repos[0],
          },
          {
            name: 'docs',
            path: 'services/docs',
            branch: 'agent/docs-123',
            headCommit: 'abcdef123456',
            headSummary: 'Docs branch',
            dirty: false,
            filesChanged: 0,
            added: 0,
            removed: 0,
          },
        ],
      },
    })

    const repoOneTree = deferredPromise<{
      workspaceTree: {
        conversationId: string
        repoPath: string
        path: string
        entries: Array<{ path: string; name: string; kind: 'file'; sizeBytes: number }>
      }
    }>()
    listProjectConversationWorkspaceTree.mockImplementation(async (_conversationId, input) => {
      if (input.repoPath === 'services/openase') {
        return repoOneTree.promise
      }
      return {
        workspaceTree: {
          conversationId: 'conversation-1',
          repoPath: 'services/docs',
          path: '',
          entries: [{ path: 'guide.md', name: 'guide.md', kind: 'file', sizeBytes: 20 }],
        },
      }
    })

    const multiRepoDiff = {
      ...workspaceDiff,
      reposChanged: 2,
      repos: [
        workspaceDiff.repos[0],
        {
          name: 'docs',
          path: 'services/docs',
          branch: 'agent/docs-123',
          dirty: false,
          filesChanged: 0,
          added: 0,
          removed: 0,
          files: [],
        },
      ],
    } satisfies ProjectConversationWorkspaceDiff

    const view = render(ProjectConversationWorkspaceBrowser, {
      props: {
        conversationId: 'conversation-1',
        workspaceDiff: multiRepoDiff,
        workspaceDiffLoading: false,
      },
    })

    await waitFor(() => expect(getProjectConversationWorkspace).toHaveBeenCalledTimes(1))
    await fireEvent.click(await view.findByRole('button', { name: 'docs' }))

    await waitFor(() => {
      expect(listProjectConversationWorkspaceTree).toHaveBeenCalledWith('conversation-1', {
        repoPath: 'services/docs',
        path: '',
      })
    })
    await view.findByRole('button', { name: 'guide.md' })

    repoOneTree.resolve({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: '',
        entries: [{ path: 'README.md', name: 'README.md', kind: 'file', sizeBytes: 64 }],
      },
    })
    await repoOneTree.promise
    await Promise.resolve()

    expect(view.container.textContent).toContain('guide.md')
    expect(view.queryByRole('button', { name: /README\.md/ })).toBeNull()
  })
})
