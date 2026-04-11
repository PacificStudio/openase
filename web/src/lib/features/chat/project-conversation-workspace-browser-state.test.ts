import { beforeEach, describe, expect, it, vi } from 'vitest'
import { waitFor } from '@testing-library/svelte'

import { ApiError } from '$lib/api/client'

const {
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceDiff,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
  saveProjectConversationWorkspaceFile,
} = vi.hoisted(() => ({
  getProjectConversationWorkspace: vi.fn(),
  getProjectConversationWorkspaceDiff: vi.fn(),
  getProjectConversationWorkspaceFilePatch: vi.fn(),
  getProjectConversationWorkspaceFilePreview: vi.fn(),
  listProjectConversationWorkspaceTree: vi.fn(),
  saveProjectConversationWorkspaceFile: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceDiff,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
  saveProjectConversationWorkspaceFile,
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
  })

  it('restores a persisted draft for the same conversation repo and file', async () => {
    const state = createProjectConversationWorkspaceBrowserState({
      getConversationId: () => 'conversation-1',
    })

    await state.refreshWorkspace(true)
    state.selectFile('README.md')
    await waitFor(() => expect(state.selectedEditorState?.draftContent).toBe('line one\n'))

    state.setSelectedViewMode('edit')
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
        viewMode: 'edit',
        baseSavedRevision: 'rev-1',
      }),
    )
  })

  it('keeps the local draft and enters conflict mode when save detects a stale revision', async () => {
    const state = createProjectConversationWorkspaceBrowserState({
      getConversationId: () => 'conversation-1',
    })

    await state.refreshWorkspace(true)
    state.selectFile('README.md')
    await waitFor(() => expect(state.selectedEditorState?.draftContent).toBe('line one\n'))

    state.setSelectedViewMode('edit')
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
      viewMode: 'diff',
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

    state.setSelectedViewMode('edit')
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
})
