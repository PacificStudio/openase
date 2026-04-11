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
})
