import type { ProjectConversationWorkspaceDiff } from '$lib/api/chat'

export const workspaceMetadata = {
  conversationId: 'conversation-1',
  available: true,
  workspacePath: '/tmp/conversation-1',
  repos: [
    {
      name: 'openase',
      path: 'services/openase',
      branch: 'agent/conv-123',
      currentRef: {
        kind: 'branch',
        displayName: 'agent/conv-123',
        cacheKey: 'branch:refs/heads/agent/conv-123',
        branchName: 'agent/conv-123',
        branchFullName: 'refs/heads/agent/conv-123',
        commitId: '123456789abc',
        shortCommitId: '123456789abc',
        subject: 'Add workspace browser scaffolding',
      },
      headCommit: '123456789abc',
      headSummary: 'Add workspace browser scaffolding',
      dirty: true,
      filesChanged: 1,
      added: 2,
      removed: 0,
    },
  ],
}

export const workspaceDiff = {
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

export function mockWorkspaceMetadata(getProjectConversationWorkspace: {
  mockResolvedValue: (value: unknown) => void
}) {
  getProjectConversationWorkspace.mockResolvedValue({
    workspace: structuredClone(workspaceMetadata),
  })
}

export function deferredPromise<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

export function ensureResizeObserver() {
  globalThis.ResizeObserver ??= class {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
}
