import { cleanup, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const {
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
} = vi.hoisted(() => ({
  getProjectConversationWorkspace: vi.fn(),
  getProjectConversationWorkspaceFilePatch: vi.fn(),
  getProjectConversationWorkspaceFilePreview: vi.fn(),
  listProjectConversationWorkspaceTree: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
}))

import type { ProjectConversationWorkspaceDiff } from '$lib/api/chat'
import ProjectConversationWorkspaceBrowser from './project-conversation-workspace-browser.svelte'

const workspaceDiff = {
  conversationId: 'conversation-1',
  workspacePath: '/tmp/conversation-1',
  dirty: true,
  reposChanged: 1,
  filesChanged: 1,
  added: 2,
  removed: 0,
  repos: [
    {
      name: 'openase',
      path: 'services/openase',
      branch: 'agent/conv-123',
      dirty: true,
      filesChanged: 1,
      added: 2,
      removed: 0,
      files: [{ path: 'README.md', status: 'modified', added: 2, removed: 0 }],
    },
  ],
} satisfies ProjectConversationWorkspaceDiff

describe('ProjectConversationWorkspaceBrowser', () => {
  beforeAll(() => {
    globalThis.ResizeObserver ??= class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('reloads workspace metadata after the parent diff refresh completes', async () => {
    getProjectConversationWorkspace.mockResolvedValue({
      workspace: {
        conversationId: 'conversation-1',
        available: true,
        workspacePath: '/tmp/conversation-1',
        repos: [
          {
            name: 'openase',
            path: 'services/openase',
            branch: 'agent/conv-123',
            headCommit: '123456789abc',
            headSummary: 'Add workspace browser scaffolding',
            dirty: true,
            filesChanged: 1,
            added: 2,
            removed: 0,
          },
        ],
      },
    })
    listProjectConversationWorkspaceTree.mockResolvedValue({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: '',
        entries: [{ path: 'README.md', name: 'README.md', kind: 'file', sizeBytes: 64 }],
      },
    })
    getProjectConversationWorkspaceFilePreview.mockResolvedValue({
      filePreview: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        sizeBytes: 64,
        mediaType: 'text/plain',
        previewKind: 'text',
        truncated: false,
        content: 'line one\nline two\n',
      },
    })
    getProjectConversationWorkspaceFilePatch.mockResolvedValue({
      filePatch: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        status: 'modified',
        diffKind: 'text',
        truncated: false,
        diff: '@@ -1 +1,2 @@\n line one\n+line two\n',
      },
    })

    const view = render(ProjectConversationWorkspaceBrowser, {
      props: {
        conversationId: 'conversation-1',
        workspaceDiff,
        workspaceDiffLoading: false,
      },
    })

    await waitFor(() => expect(getProjectConversationWorkspace).toHaveBeenCalledTimes(1))

    await view.rerender({
      conversationId: 'conversation-1',
      workspaceDiff,
      workspaceDiffLoading: true,
    })
    await view.rerender({
      conversationId: 'conversation-1',
      workspaceDiff,
      workspaceDiffLoading: false,
    })

    await waitFor(() => expect(getProjectConversationWorkspace).toHaveBeenCalledTimes(2))
  })
})
