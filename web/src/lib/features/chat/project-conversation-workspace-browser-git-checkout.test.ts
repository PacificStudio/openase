import { beforeEach, describe, expect, it, vi } from 'vitest'
import { waitFor } from '@testing-library/svelte'

import { ApiError } from '$lib/api/client'

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
  it('blocks branch checkout when the selected repo still has unsaved drafts', async () => {
    const state = createProjectConversationWorkspaceBrowserState({
      getConversationId: () => 'conversation-1',
    })

    await state.refreshWorkspace(true)
    state.selectFile('README.md')
    await waitFor(() => expect(state.selectedEditorState?.draftContent).toBe('line one\n'))
    state.updateSelectedDraft('unsaved draft\n')

    const result = await state.checkoutBranch({
      repoPath: 'services/openase',
      targetKind: 'local_branch',
      targetName: 'main',
      createTrackingBranch: false,
    })

    expect(result.ok).toBe(false)
    expect(result.blockers.join(' ')).toContain('Unsaved drafts')
    expect(checkoutProjectConversationWorkspaceBranch).not.toHaveBeenCalled()
  })

  it('falls back to the existing local branch when a remote tracking branch already has a local checkout', async () => {
    getProjectConversationWorkspace.mockResolvedValueOnce({
      workspace: {
        conversationId: 'conversation-1',
        available: true,
        workspacePath: '/tmp/conversation-1',
        repos: [
          {
            name: 'openase',
            path: 'services/openase',
            branch: 'feature/current',
            currentRef: {
              kind: 'branch',
              displayName: 'feature/current',
              cacheKey: 'branch:refs/heads/feature/current',
              branchName: 'feature/current',
              branchFullName: 'refs/heads/feature/current',
              commitId: '123456789abc',
              shortCommitId: '123456789abc',
              subject: 'Current branch',
            },
            headCommit: '123456789abc',
            headSummary: 'Current branch',
            dirty: false,
            filesChanged: 0,
            added: 0,
            removed: 0,
          },
        ],
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValueOnce({
      workspaceDiff: {
        conversationId: 'conversation-1',
        workspacePath: '/tmp/conversation-1',
        dirty: false,
        reposChanged: 0,
        filesChanged: 0,
        added: 0,
        removed: 0,
        repos: [],
      },
    })
    getProjectConversationWorkspaceRepoRefs.mockResolvedValueOnce({
      repoRefs: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        currentRef: {
          kind: 'branch',
          displayName: 'feature/current',
          cacheKey: 'branch:refs/heads/feature/current',
          branchName: 'feature/current',
          branchFullName: 'refs/heads/feature/current',
          commitId: '123456789abc',
          shortCommitId: '123456789abc',
          subject: 'Current branch',
        },
        localBranches: [
          {
            name: 'main',
            fullName: 'refs/heads/main',
            scope: 'local_branch',
            current: false,
            commitId: 'abcdef012345',
            shortCommitId: 'abcdef012345',
            subject: 'Main branch',
            upstreamName: 'origin/main',
            ahead: 0,
            behind: 0,
            suggestedLocalBranchName: '',
          },
        ],
        remoteBranches: [
          {
            name: 'origin/main',
            fullName: 'refs/remotes/origin/main',
            scope: 'remote_tracking_branch',
            current: false,
            commitId: 'abcdef012345',
            shortCommitId: 'abcdef012345',
            subject: 'Main branch',
            upstreamName: '',
            ahead: 0,
            behind: 0,
            suggestedLocalBranchName: 'main',
          },
        ],
      },
    })

    const state = createProjectConversationWorkspaceBrowserState({
      getConversationId: () => 'conversation-1',
    })

    checkoutProjectConversationWorkspaceBranch.mockResolvedValueOnce({
      checkout: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        currentRef: {
          kind: 'branch',
          displayName: 'main',
          cacheKey: 'branch:refs/heads/main',
          branchName: 'main',
          branchFullName: 'refs/heads/main',
          commitId: 'abcdef012345',
          shortCommitId: 'abcdef012345',
          subject: 'Main branch',
        },
        createdLocalBranch: '',
      },
    })
    getProjectConversationWorkspace.mockResolvedValueOnce({
      workspace: {
        conversationId: 'conversation-1',
        available: true,
        workspacePath: '/tmp/conversation-1',
        repos: [
          {
            name: 'openase',
            path: 'services/openase',
            branch: 'main',
            currentRef: {
              kind: 'branch',
              displayName: 'main',
              cacheKey: 'branch:refs/heads/main',
              branchName: 'main',
              branchFullName: 'refs/heads/main',
              commitId: 'abcdef012345',
              shortCommitId: 'abcdef012345',
              subject: 'Main branch',
            },
            headCommit: 'abcdef012345',
            headSummary: 'Main branch',
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
          displayName: 'main',
          cacheKey: 'branch:refs/heads/main',
          branchName: 'main',
          branchFullName: 'refs/heads/main',
          commitId: 'abcdef012345',
          shortCommitId: 'abcdef012345',
          subject: 'Main branch',
        },
        localBranches: [
          {
            name: 'main',
            fullName: 'refs/heads/main',
            scope: 'local_branch',
            current: true,
            commitId: 'abcdef012345',
            shortCommitId: 'abcdef012345',
            subject: 'Main branch',
            upstreamName: 'origin/main',
            ahead: 0,
            behind: 0,
            suggestedLocalBranchName: '',
          },
        ],
        remoteBranches: [
          {
            name: 'origin/main',
            fullName: 'refs/remotes/origin/main',
            scope: 'remote_tracking_branch',
            current: false,
            commitId: 'abcdef012345',
            shortCommitId: 'abcdef012345',
            subject: 'Main branch',
            upstreamName: '',
            ahead: 0,
            behind: 0,
            suggestedLocalBranchName: 'main',
          },
        ],
      },
    })
    getProjectConversationWorkspaceGitGraph.mockResolvedValueOnce({
      gitGraph: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        limit: 40,
        commits: [],
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValueOnce({
      workspaceDiff: {
        conversationId: 'conversation-1',
        workspacePath: '/tmp/conversation-1',
        dirty: false,
        reposChanged: 0,
        filesChanged: 0,
        added: 0,
        removed: 0,
        repos: [],
      },
    })

    await state.refreshWorkspace(true)

    const result = await state.checkoutBranch({
      repoPath: 'services/openase',
      targetKind: 'remote_tracking_branch',
      targetName: 'origin/main',
      createTrackingBranch: true,
      localBranchName: 'main',
    })

    expect(result.ok).toBe(true)
    expect(checkoutProjectConversationWorkspaceBranch).toHaveBeenCalledWith('conversation-1', {
      repoPath: 'services/openase',
      targetKind: 'local_branch',
      targetName: 'main',
      createTrackingBranch: false,
      localBranchName: undefined,
      expectedCleanWorkspace: true,
    })
    await waitFor(() => expect(state.metadata?.repos[0]?.branch).toBe('main'))
  })

  it('refreshes the tree, file preview, and git context after a successful branch checkout', async () => {
    const state = createProjectConversationWorkspaceBrowserState({
      getConversationId: () => 'conversation-1',
    })

    getProjectConversationWorkspace.mockResolvedValueOnce({
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
            dirty: false,
            filesChanged: 0,
            added: 0,
            removed: 0,
          },
        ],
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValueOnce({
      workspaceDiff: {
        conversationId: 'conversation-1',
        workspacePath: '/tmp/conversation-1',
        dirty: false,
        reposChanged: 0,
        filesChanged: 0,
        added: 0,
        removed: 0,
        repos: [],
      },
    })
    await state.refreshWorkspace(true)
    state.selectFile('README.md')
    await waitFor(() => expect(state.selectedEditorState?.draftContent).toBe('line one\n'))

    checkoutProjectConversationWorkspaceBranch.mockResolvedValueOnce({
      checkout: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        currentRef: {
          kind: 'branch',
          displayName: 'feature/checked-out',
          cacheKey: 'branch:refs/heads/feature/checked-out',
          branchName: 'feature/checked-out',
          branchFullName: 'refs/heads/feature/checked-out',
          commitId: 'feedface1234',
          shortCommitId: 'feedface1234',
          subject: 'Checked out branch',
        },
        createdLocalBranch: '',
      },
    })
    getProjectConversationWorkspace.mockResolvedValueOnce({
      workspace: {
        conversationId: 'conversation-1',
        available: true,
        workspacePath: '/tmp/conversation-1',
        repos: [
          {
            name: 'openase',
            path: 'services/openase',
            branch: 'feature/checked-out',
            currentRef: {
              kind: 'branch',
              displayName: 'feature/checked-out',
              cacheKey: 'branch:refs/heads/feature/checked-out',
              branchName: 'feature/checked-out',
              branchFullName: 'refs/heads/feature/checked-out',
              commitId: 'feedface1234',
              shortCommitId: 'feedface1234',
              subject: 'Checked out branch',
            },
            headCommit: 'feedface1234',
            headSummary: 'Checked out branch',
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
          displayName: 'feature/checked-out',
          cacheKey: 'branch:refs/heads/feature/checked-out',
          branchName: 'feature/checked-out',
          branchFullName: 'refs/heads/feature/checked-out',
          commitId: 'feedface1234',
          shortCommitId: 'feedface1234',
          subject: 'Checked out branch',
        },
        localBranches: [],
        remoteBranches: [],
      },
    })
    getProjectConversationWorkspaceGitGraph.mockResolvedValueOnce({
      gitGraph: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        limit: 40,
        commits: [],
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValueOnce({
      workspaceDiff: {
        conversationId: 'conversation-1',
        workspacePath: '/tmp/conversation-1',
        dirty: false,
        reposChanged: 0,
        filesChanged: 0,
        added: 0,
        removed: 0,
        repos: [],
      },
    })
    getProjectConversationWorkspaceFilePreview.mockResolvedValueOnce({
      filePreview: buildPreview({
        content: 'branch b content\n',
        revision: 'rev-b-1',
      }),
    })
    getProjectConversationWorkspaceFilePatch.mockResolvedValueOnce({
      filePatch: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        status: 'modified',
        diffKind: 'text',
        truncated: false,
        diff: '',
      },
    })

    const result = await state.checkoutBranch({
      repoPath: 'services/openase',
      targetKind: 'local_branch',
      targetName: 'feature/checked-out',
      createTrackingBranch: false,
    })

    expect(result.ok).toBe(true)
    expect(checkoutProjectConversationWorkspaceBranch).toHaveBeenCalledWith('conversation-1', {
      repoPath: 'services/openase',
      targetKind: 'local_branch',
      targetName: 'feature/checked-out',
      createTrackingBranch: false,
      localBranchName: undefined,
      expectedCleanWorkspace: true,
    })
    await waitFor(() => expect(state.metadata?.repos[0]?.branch).toBe('feature/checked-out'))
    await waitFor(() =>
      expect(state.selectedEditorState).toMatchObject({
        draftContent: 'branch b content\n',
        dirty: false,
        baseSavedRevision: 'rev-b-1',
      }),
    )
  })

  it('shows a branch-specific missing file message when the active file disappears after checkout', async () => {
    const state = createProjectConversationWorkspaceBrowserState({
      getConversationId: () => 'conversation-1',
    })

    getProjectConversationWorkspace.mockResolvedValueOnce({
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
            dirty: false,
            filesChanged: 0,
            added: 0,
            removed: 0,
          },
        ],
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValueOnce({
      workspaceDiff: {
        conversationId: 'conversation-1',
        workspacePath: '/tmp/conversation-1',
        dirty: false,
        reposChanged: 0,
        filesChanged: 0,
        added: 0,
        removed: 0,
        repos: [],
      },
    })
    await state.refreshWorkspace(true)
    state.selectFile('README.md')
    await waitFor(() => expect(state.selectedEditorState?.draftContent).toBe('line one\n'))

    checkoutProjectConversationWorkspaceBranch.mockResolvedValueOnce({
      checkout: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        currentRef: {
          kind: 'branch',
          displayName: 'feature/missing-file',
          cacheKey: 'branch:refs/heads/feature/missing-file',
          branchName: 'feature/missing-file',
          branchFullName: 'refs/heads/feature/missing-file',
          commitId: 'feedface5678',
          shortCommitId: 'feedface5678',
          subject: 'Missing file branch',
        },
        createdLocalBranch: '',
      },
    })
    getProjectConversationWorkspace.mockResolvedValueOnce({
      workspace: {
        conversationId: 'conversation-1',
        available: true,
        workspacePath: '/tmp/conversation-1',
        repos: [
          {
            name: 'openase',
            path: 'services/openase',
            branch: 'feature/missing-file',
            currentRef: {
              kind: 'branch',
              displayName: 'feature/missing-file',
              cacheKey: 'branch:refs/heads/feature/missing-file',
              branchName: 'feature/missing-file',
              branchFullName: 'refs/heads/feature/missing-file',
              commitId: 'feedface5678',
              shortCommitId: 'feedface5678',
              subject: 'Missing file branch',
            },
            headCommit: 'feedface5678',
            headSummary: 'Missing file branch',
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
          displayName: 'feature/missing-file',
          cacheKey: 'branch:refs/heads/feature/missing-file',
          branchName: 'feature/missing-file',
          branchFullName: 'refs/heads/feature/missing-file',
          commitId: 'feedface5678',
          shortCommitId: 'feedface5678',
          subject: 'Missing file branch',
        },
        localBranches: [],
        remoteBranches: [],
      },
    })
    getProjectConversationWorkspaceGitGraph.mockResolvedValueOnce({
      gitGraph: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        limit: 40,
        commits: [],
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValueOnce({
      workspaceDiff: {
        conversationId: 'conversation-1',
        workspacePath: '/tmp/conversation-1',
        dirty: false,
        reposChanged: 0,
        filesChanged: 0,
        added: 0,
        removed: 0,
        repos: [],
      },
    })
    getProjectConversationWorkspaceFilePreview.mockRejectedValueOnce(
      new ApiError(
        404,
        'project conversation workspace entry not found',
        'PROJECT_CONVERSATION_WORKSPACE_NOT_FOUND',
      ),
    )

    const result = await state.checkoutBranch({
      repoPath: 'services/openase',
      targetKind: 'local_branch',
      targetName: 'feature/missing-file',
      createTrackingBranch: false,
    })

    expect(result.ok).toBe(true)
    await waitFor(() =>
      expect(state.fileError).toBe('This file is not present on the current branch.'),
    )
  })
})
