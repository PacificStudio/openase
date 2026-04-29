import { beforeEach, describe, expect, it, vi } from 'vitest'
import { waitFor } from '@testing-library/svelte'

const {
  checkoutProjectConversationWorkspaceBranch,
  createProjectConversationWorkspaceFile,
  deleteProjectConversationWorkspaceFile,
  getProjectConversationWorkspaceGitGraph,
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceDiff,
  getProjectConversationWorkspaceRepoRefs,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
  renameProjectConversationWorkspaceFile,
  saveProjectConversationWorkspaceFile,
  searchProjectConversationWorkspacePaths,
} = vi.hoisted(() => ({
  checkoutProjectConversationWorkspaceBranch: vi.fn(),
  createProjectConversationWorkspaceFile: vi.fn(),
  deleteProjectConversationWorkspaceFile: vi.fn(),
  getProjectConversationWorkspaceGitGraph: vi.fn(),
  getProjectConversationWorkspace: vi.fn(),
  getProjectConversationWorkspaceDiff: vi.fn(),
  getProjectConversationWorkspaceRepoRefs: vi.fn(),
  getProjectConversationWorkspaceFilePatch: vi.fn(),
  getProjectConversationWorkspaceFilePreview: vi.fn(),
  listProjectConversationWorkspaceTree: vi.fn(),
  renameProjectConversationWorkspaceFile: vi.fn(),
  saveProjectConversationWorkspaceFile: vi.fn(),
  searchProjectConversationWorkspacePaths: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  checkoutProjectConversationWorkspaceBranch,
  createProjectConversationWorkspaceFile,
  deleteProjectConversationWorkspaceFile,
  getProjectConversationWorkspaceGitGraph,
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceDiff,
  getProjectConversationWorkspaceRepoRefs,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
  renameProjectConversationWorkspaceFile,
  saveProjectConversationWorkspaceFile,
  searchProjectConversationWorkspacePaths,
}))

import { createProjectConversationWorkspaceBrowserState } from './project-conversation-workspace-browser-state.svelte'

function mockWorkspaceMetadata() {
  getProjectConversationWorkspace.mockResolvedValue({
    workspace: {
      conversationId: 'conversation-1',
      available: true,
      workspacePath: '/tmp/conversation-1',
      repos: [
        {
          name: 'openase',
          path: 'services/openase',
          branch: 'feat/ase-162-workspace-editor',
          currentRef: {
            kind: 'branch',
            displayName: 'feat/ase-162-workspace-editor',
            cacheKey: 'branch:refs/heads/feat/ase-162-workspace-editor',
            branchName: 'feat/ase-162-workspace-editor',
            branchFullName: 'refs/heads/feat/ase-162-workspace-editor',
            commitId: '123456789abc',
            shortCommitId: '123456789abc',
            subject: 'Workspace editor',
          },
          headCommit: '123456789abc',
          headSummary: 'Workspace editor',
          dirty: true,
          filesChanged: 1,
          added: 1,
          removed: 0,
        },
      ],
    },
  })
}

function mockTree() {
  listProjectConversationWorkspaceTree.mockResolvedValue({
    workspaceTree: {
      conversationId: 'conversation-1',
      repoPath: 'services/openase',
      path: '',
      entries: [{ path: 'README.md', name: 'README.md', kind: 'file', sizeBytes: 12 }],
    },
  })
}

function buildPreview(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    conversationId: 'conversation-1',
    repoPath: 'services/openase',
    path: 'README.md',
    sizeBytes: 12,
    mediaType: 'text/plain',
    previewKind: 'text',
    truncated: false,
    content: 'line one\n',
    revision: 'rev-1',
    writable: true,
    readOnlyReason: '',
    encoding: 'utf-8',
    lineEnding: 'lf',
    ...overrides,
  }
}

function mockPatch() {
  getProjectConversationWorkspaceFilePatch.mockResolvedValue({
    filePatch: {
      conversationId: 'conversation-1',
      repoPath: 'services/openase',
      path: 'README.md',
      status: 'modified',
      diffKind: 'text',
      truncated: false,
      diff: '@@ -1 +1 @@\n-line one\n+line two\n',
    },
  })
}

describe('createProjectConversationWorkspaceBrowserState', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    window.localStorage.clear()
    createProjectConversationWorkspaceFile.mockResolvedValue({
      file: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        revision: 'rev-new',
        sizeBytes: 0,
        encoding: 'utf-8',
        lineEnding: 'lf',
      },
    })
    deleteProjectConversationWorkspaceFile.mockResolvedValue({
      file: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
      },
    })
    mockWorkspaceMetadata()
    mockTree()
    mockPatch()
    getProjectConversationWorkspaceFilePreview.mockResolvedValue({
      filePreview: buildPreview(),
    })
    saveProjectConversationWorkspaceFile.mockResolvedValue({
      file: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        revision: 'rev-2',
        sizeBytes: 21,
        encoding: 'utf-8',
        lineEnding: 'lf',
      },
    })
    renameProjectConversationWorkspaceFile.mockResolvedValue({
      file: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        fromPath: 'README.md',
        toPath: 'docs/README.md',
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue({
      workspaceDiff: {
        conversationId: 'conversation-1',
        workspacePath: '/tmp/conversation-1',
        dirty: true,
        reposChanged: 1,
        filesChanged: 1,
        added: 1,
        removed: 0,
        repos: [],
      },
    })
    searchProjectConversationWorkspacePaths.mockResolvedValue({
      workspaceSearch: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        query: 'readme',
        truncated: false,
        results: [{ path: 'docs/README.md', name: 'README.md' }],
      },
    })
    getProjectConversationWorkspaceRepoRefs.mockResolvedValue({
      repoRefs: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        currentRef: {
          kind: 'branch',
          displayName: 'feat/ase-162-workspace-editor',
          cacheKey: 'branch:refs/heads/feat/ase-162-workspace-editor',
          branchName: 'feat/ase-162-workspace-editor',
          branchFullName: 'refs/heads/feat/ase-162-workspace-editor',
          commitId: '123456789abc',
          shortCommitId: '123456789abc',
          subject: 'Workspace editor',
        },
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
        currentRef: {
          kind: 'branch',
          displayName: 'feat/ase-162-workspace-editor',
          cacheKey: 'branch:refs/heads/feat/ase-162-workspace-editor',
          branchName: 'feat/ase-162-workspace-editor',
          branchFullName: 'refs/heads/feat/ase-162-workspace-editor',
          commitId: '123456789abc',
          shortCommitId: '123456789abc',
          subject: 'Workspace editor',
        },
        createdLocalBranch: '',
      },
    })
  })
  it('restores a persisted draft for the same conversation repo and file', async () => {
    const state = createProjectConversationWorkspaceBrowserState({
      getConversationId: () => 'conversation-1',
    })

    await state.refreshWorkspace(true)
    state.selectFile('README.md')
    await waitFor(() => expect(state.selectedEditorState?.draftContent).toBe('line one\n'))

    state.updateSelectedDraft('line one\nline two\n')

    const restored = createProjectConversationWorkspaceBrowserState({
      getConversationId: () => 'conversation-1',
    })
    await restored.refreshWorkspace(true)
    restored.selectFile('README.md')

    await waitFor(() =>
      expect(restored.selectedEditorState).toMatchObject({
        draftContent: 'line one\nline two\n',
        dirty: true,
        baseSavedRevision: 'rev-1',
      }),
    )
  })

  it('does not reuse a persisted draft after the repo branch context changes', async () => {
    const state = createProjectConversationWorkspaceBrowserState({
      getConversationId: () => 'conversation-1',
    })

    await state.refreshWorkspace(true)
    state.selectFile('README.md')
    await waitFor(() => expect(state.selectedEditorState?.draftContent).toBe('line one\n'))
    state.updateSelectedDraft('branch A draft\n')

    getProjectConversationWorkspace.mockResolvedValueOnce({
      workspace: {
        conversationId: 'conversation-1',
        available: true,
        workspacePath: '/tmp/conversation-1',
        repos: [
          {
            name: 'openase',
            path: 'services/openase',
            branch: 'feat/ase-170-other-branch',
            currentRef: {
              kind: 'branch',
              displayName: 'feat/ase-170-other-branch',
              cacheKey: 'branch:refs/heads/feat/ase-170-other-branch',
              branchName: 'feat/ase-170-other-branch',
              branchFullName: 'refs/heads/feat/ase-170-other-branch',
              commitId: 'abcdef123456',
              shortCommitId: 'abcdef123456',
              subject: 'Other branch',
            },
            headCommit: 'abcdef123456',
            headSummary: 'Other branch',
            dirty: false,
            filesChanged: 0,
            added: 0,
            removed: 0,
          },
        ],
      },
    })
    getProjectConversationWorkspaceRepoRefs.mockResolvedValueOnce({
      repoRefs: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        currentRef: {
          kind: 'branch',
          displayName: 'feat/ase-170-other-branch',
          cacheKey: 'branch:refs/heads/feat/ase-170-other-branch',
          branchName: 'feat/ase-170-other-branch',
          branchFullName: 'refs/heads/feat/ase-170-other-branch',
          commitId: 'abcdef123456',
          shortCommitId: 'abcdef123456',
          subject: 'Other branch',
        },
        localBranches: [],
        remoteBranches: [],
      },
    })
    getProjectConversationWorkspaceFilePreview.mockResolvedValueOnce({
      filePreview: buildPreview({
        content: 'branch b content\n',
        revision: 'rev-b-1',
      }),
    })

    await state.refreshWorkspace(true)

    await waitFor(() =>
      expect(state.selectedEditorState).toMatchObject({
        draftContent: 'branch b content\n',
        dirty: false,
        baseSavedRevision: 'rev-b-1',
      }),
    )

    const storage = window.localStorage.getItem(
      'openase.project-conversation.workspace-file-drafts',
    )
    expect(storage).toContain('branch:refs/heads/feat/ase-162-workspace-editor')
    expect(storage).not.toContain('branch:refs/heads/feat/ase-170-other-branch::README.md')
  })

  it('searches repo paths through the workspace search API for the selected repo', async () => {
    const state = createProjectConversationWorkspaceBrowserState({
      getConversationId: () => 'conversation-1',
    })

    await state.refreshWorkspace(true)

    const results = await state.searchPaths('readme', 10)

    expect(searchProjectConversationWorkspacePaths).toHaveBeenCalledWith('conversation-1', {
      repoPath: 'services/openase',
      query: 'readme',
      limit: 10,
    })
    expect(results).toEqual([{ path: 'docs/README.md', name: 'README.md' }])
  })
})
