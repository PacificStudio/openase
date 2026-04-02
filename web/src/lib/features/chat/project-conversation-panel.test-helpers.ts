export function createWorkspaceDiff(conversationId: string, dirty = false) {
  return {
    workspaceDiff: {
      conversationId,
      workspacePath: `/tmp/${conversationId}`,
      dirty,
      reposChanged: dirty ? 1 : 0,
      filesChanged: dirty ? 1 : 0,
      added: dirty ? 4 : 0,
      removed: dirty ? 1 : 0,
      repos: dirty
        ? [
            {
              name: 'openase',
              path: 'services/openase',
              branch: 'agent/conv-123',
              dirty: true,
              filesChanged: 1,
              added: 4,
              removed: 1,
              files: [
                {
                  path: 'web/src/app.ts',
                  status: 'modified',
                  added: 4,
                  removed: 1,
                },
              ],
            },
          ]
        : [],
    },
  }
}
