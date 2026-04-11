export function createWorkspaceDiff(conversationId: string) {
  return {
    workspaceDiff: {
      conversationId,
      workspacePath: `/tmp/${conversationId}`,
      dirty: false,
      reposChanged: 0,
      filesChanged: 0,
      added: 0,
      removed: 0,
      repos: [],
    },
  }
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

export function resolvedMuxSubscription() {
  return {
    stream: Promise.resolve(),
    connected: Promise.resolve(),
  }
}

export function startedTurnResponse(conversationId = 'conversation-1', providerId = 'provider-1') {
  return {
    turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    conversation: {
      id: conversationId,
      providerId,
      lastActivityAt: '2026-04-01T10:00:00Z',
    },
  }
}

export function seedProjectConversationTabsStorage(
  tabs: Array<{
    conversationId: string
    providerId: string
    draft?: string
    projectId?: string
  }>,
  activeTabIndex: number,
) {
  window.localStorage.setItem(
    'openase.project-conversation.global',
    JSON.stringify({
      tabs: tabs.map((tab) => ({
        projectId: tab.projectId ?? 'project-1',
        projectName: 'Project 1',
        conversationId: tab.conversationId,
        providerId: tab.providerId,
        draft: tab.draft ?? '',
      })),
      activeTabIndex,
    }),
  )
}
