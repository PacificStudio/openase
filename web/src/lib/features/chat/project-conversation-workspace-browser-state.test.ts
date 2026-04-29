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
  it('keeps the local draft and enters conflict mode when save detects a stale revision', async () => {
    const state = createProjectConversationWorkspaceBrowserState({
      getConversationId: () => 'conversation-1',
    })

    await state.refreshWorkspace(true)
    state.selectFile('README.md')
    await waitFor(() => expect(state.selectedEditorState?.draftContent).toBe('line one\n'))

    state.updateSelectedDraft('my local draft\n')

    saveProjectConversationWorkspaceFile.mockRejectedValue(
      new ApiError(
        409,
        'The workspace file changed before your save completed.',
        'PROJECT_CONVERSATION_WORKSPACE_FILE_CONFLICT',
        {
          current_file: buildPreview({
            content: 'server version\n',
            revision: 'rev-2',
          }),
        },
      ),
    )

    await state.saveSelectedFile()

    expect(state.selectedEditorState).toMatchObject({
      draftContent: 'my local draft\n',
      latestSavedContent: 'server version\n',
      latestSavedRevision: 'rev-2',
      dirty: true,
      savePhase: 'conflict',
      externalChange: true,
    })
  })

  it('clears the dirty draft and refreshes workspace diff after a successful save', async () => {
    const onWorkspaceDiffUpdated = vi.fn()
    const state = createProjectConversationWorkspaceBrowserState({
      getConversationId: () => 'conversation-1',
      onWorkspaceDiffUpdated,
    })

    getProjectConversationWorkspaceFilePreview.mockResolvedValueOnce({
      filePreview: buildPreview(),
    })
    getProjectConversationWorkspaceFilePreview.mockResolvedValueOnce({
      filePreview: buildPreview({
        content: 'line one\nline two\n',
        revision: 'rev-2',
        sizeBytes: 21,
      }),
    })

    await state.refreshWorkspace(true)
    state.selectFile('README.md')
    await waitFor(() => expect(state.selectedEditorState?.draftContent).toBe('line one\n'))

    state.updateSelectedDraft('line one\nline two\n')
    await state.saveSelectedFile()

    expect(saveProjectConversationWorkspaceFile).toHaveBeenCalledWith('conversation-1', {
      repoPath: 'services/openase',
      path: 'README.md',
      baseRevision: 'rev-1',
      content: 'line one\nline two\n',
      encoding: 'utf-8',
      lineEnding: 'lf',
    })
    await waitFor(() =>
      expect(state.selectedEditorState).toMatchObject({
        draftContent: 'line one\nline two\n',
        baseSavedRevision: 'rev-2',
        latestSavedRevision: 'rev-2',
        dirty: false,
        savePhase: 'idle',
        externalChange: false,
      }),
    )
    expect(onWorkspaceDiffUpdated).toHaveBeenCalledTimes(1)
    expect(window.localStorage.getItem('openase.project-conversation.workspace-file-drafts')).toBe(
      null,
    )
  })

  it('keeps drafts isolated across tabs and can save a non-active dirty tab', async () => {
    listProjectConversationWorkspaceTree.mockResolvedValue({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: '',
        entries: [
          { path: 'README.md', name: 'README.md', kind: 'file', sizeBytes: 12 },
          { path: 'package.json', name: 'package.json', kind: 'file', sizeBytes: 18 },
        ],
      },
    })
    getProjectConversationWorkspaceFilePatch.mockImplementation(async (_conversationId, input) => {
      return {
        filePatch: {
          conversationId: 'conversation-1',
          repoPath: 'services/openase',
          path: input.path,
          status: 'modified',
          diffKind: 'text',
          truncated: false,
          diff: '',
        },
      }
    })
    saveProjectConversationWorkspaceFile.mockResolvedValue({
      file: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        revision: 'rev-2',
        sizeBytes: 13,
        encoding: 'utf-8',
        lineEnding: 'lf',
      },
    })

    let readmeLoads = 0
    getProjectConversationWorkspaceFilePreview.mockImplementation(
      async (_conversationId, input) => {
        if (input.path === 'README.md') {
          readmeLoads += 1
          return {
            filePreview:
              readmeLoads === 1
                ? buildPreview()
                : buildPreview({
                    content: 'readme updated\n',
                    revision: 'rev-2',
                    sizeBytes: 15,
                  }),
          }
        }
        return {
          filePreview: buildPreview({
            path: 'package.json',
            sizeBytes: 18,
            mediaType: 'application/json',
            content: '{"name":"pkg"}\n',
            revision: 'pkg-rev-1',
          }),
        }
      },
    )

    const state = createProjectConversationWorkspaceBrowserState({
      getConversationId: () => 'conversation-1',
    })

    await state.refreshWorkspace(true)
    state.selectFile('README.md')
    await waitFor(() => expect(state.selectedEditorState?.draftContent).toBe('line one\n'))

    state.updateSelectedDraft('readme updated\n')

    state.selectFile('package.json')
    await waitFor(() => expect(state.selectedEditorState?.draftContent).toBe('{"name":"pkg"}\n'))

    expect(state.openTabs).toHaveLength(2)
    expect(state.getEditorState('services/openase', 'README.md')).toMatchObject({
      draftContent: 'readme updated\n',
      dirty: true,
    })
    expect(state.getEditorState('services/openase', 'package.json')).toMatchObject({
      draftContent: '{"name":"pkg"}\n',
      dirty: false,
    })

    const saved = await state.saveFile('services/openase', 'README.md')
    expect(saved).toBe(true)

    expect(saveProjectConversationWorkspaceFile).toHaveBeenCalledWith('conversation-1', {
      repoPath: 'services/openase',
      path: 'README.md',
      baseRevision: 'rev-1',
      content: 'readme updated\n',
      encoding: 'utf-8',
      lineEnding: 'lf',
    })
    await waitFor(() =>
      expect(state.getEditorState('services/openase', 'README.md')).toMatchObject({
        draftContent: 'readme updated\n',
        baseSavedRevision: 'rev-2',
        dirty: false,
      }),
    )
    expect(state.selectedEditorState).toMatchObject({
      draftContent: '{"name":"pkg"}\n',
      dirty: false,
    })
  })

  it('creates, renames, and deletes files while remapping editor state', async () => {
    listProjectConversationWorkspaceTree.mockResolvedValue({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: '',
        entries: [{ path: 'README.md', name: 'README.md', kind: 'file', sizeBytes: 12 }],
      },
    })
    getProjectConversationWorkspaceFilePreview.mockImplementation(
      async (_conversationId, input) => ({
        filePreview: buildPreview({ path: input.path, content: '', revision: `rev-${input.path}` }),
      }),
    )
    getProjectConversationWorkspaceFilePatch.mockImplementation(async (_conversationId, input) => ({
      filePatch: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: input.path,
        status: 'untracked',
        diffKind: 'text',
        truncated: false,
        diff: '',
      },
    }))

    const state = createProjectConversationWorkspaceBrowserState({
      getConversationId: () => 'conversation-1',
    })

    await state.refreshWorkspace(true)
    await state.createFile('notes/todo.md')
    expect(createProjectConversationWorkspaceFile).toHaveBeenCalledWith('conversation-1', {
      repoPath: 'services/openase',
      path: 'notes/todo.md',
    })

    state.selectFile('notes/todo.md')
    await waitFor(() => expect(state.selectedEditorState?.draftContent).toBe(''))
    state.updateSelectedDraft('draft note')

    await state.renameFile('notes/todo.md', 'notes/archive/todo.md')
    expect(renameProjectConversationWorkspaceFile).toHaveBeenCalledWith('conversation-1', {
      repoPath: 'services/openase',
      fromPath: 'notes/todo.md',
      toPath: 'notes/archive/todo.md',
    })
    expect(state.getEditorState('services/openase', 'notes/archive/todo.md')).toMatchObject({
      draftContent: 'draft note',
      dirty: true,
    })

    await state.deleteFile('notes/archive/todo.md')
    expect(deleteProjectConversationWorkspaceFile).toHaveBeenCalledWith('conversation-1', {
      repoPath: 'services/openase',
      path: 'notes/archive/todo.md',
    })
    expect(state.getEditorState('services/openase', 'notes/archive/todo.md')).toBeNull()
  })

  it('autosaves after idle when enabled and stops on conflict', async () => {
    const state = createProjectConversationWorkspaceBrowserState({
      getConversationId: () => 'conversation-1',
    })

    await state.refreshWorkspace(true)
    state.selectFile('README.md')
    await waitFor(() => expect(state.selectedEditorState?.draftContent).toBe('line one\n'))
    vi.useFakeTimers()
    try {
      state.setAutosaveEnabled(true)
      state.updateSelectedDraft('line one\nline two\n')
      await vi.advanceTimersByTimeAsync(1100)

      expect(saveProjectConversationWorkspaceFile).toHaveBeenCalledTimes(1)

      saveProjectConversationWorkspaceFile.mockRejectedValueOnce(
        new ApiError(
          409,
          'The workspace file changed before your save completed.',
          'PROJECT_CONVERSATION_WORKSPACE_FILE_CONFLICT',
          {
            current_file: buildPreview({
              content: 'server version\n',
              revision: 'rev-3',
            }),
          },
        ),
      )

      state.updateSelectedDraft('local change again\n')
      await vi.advanceTimersByTimeAsync(1100)

      expect(state.selectedEditorState?.savePhase).toBe('conflict')
      saveProjectConversationWorkspaceFile.mockClear()
      await vi.advanceTimersByTimeAsync(1100)
      expect(saveProjectConversationWorkspaceFile).not.toHaveBeenCalled()
    } finally {
      vi.useRealTimers()
    }
  })

  it('reviews and applies a Project AI patch into the local draft', async () => {
    const state = createProjectConversationWorkspaceBrowserState({
      getConversationId: () => 'conversation-1',
    })

    await state.refreshWorkspace(true)
    state.selectFile('README.md')
    await waitFor(() => expect(state.selectedEditorState?.draftContent).toBe('line one\n'))

    const reviewed = await state.reviewPatch({
      type: 'diff',
      file: 'README.md',
      hunks: [
        {
          oldStart: 1,
          oldLines: 1,
          newStart: 1,
          newLines: 2,
          lines: [
            { op: 'context', text: 'line one' },
            { op: 'add', text: 'line two' },
          ],
        },
      ],
    })

    expect(reviewed).toBe(true)
    expect(state.selectedEditorState?.pendingPatch?.proposedContent).toBe('line one\nline two')

    expect(state.applySelectedPendingPatch()).toBe(true)
    expect(state.selectedEditorState).toMatchObject({
      draftContent: 'line one\nline two',
      dirty: true,
      pendingPatch: null,
    })
  })
})
