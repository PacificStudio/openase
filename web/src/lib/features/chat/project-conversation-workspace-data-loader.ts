import {
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  type ProjectConversationWorkspaceFilePreview,
} from '$lib/api/chat'
import {
  workspaceTabKey,
  type WorkspaceTabFileState,
} from './project-conversation-workspace-browser-state-helpers'

/**
 * Wiring needed by `loadWorkspaceFile` so it can be invoked from a stateful
 * Svelte module while remaining a plain async function the harness can test in
 * isolation.
 */
export type WorkspaceFileLoaderContext = {
  getConversationId: () => string
  hasOpenTab: (key: string) => boolean
  getCurrentLoading: (key: string) => boolean
  patchTabFileState: (key: string, patch: Partial<WorkspaceTabFileState>) => void
  syncEditorFromPreview: (
    repoPath: string,
    filePath: string,
    preview: ProjectConversationWorkspaceFilePreview,
  ) => void
}

export async function loadWorkspaceFile(
  ctx: WorkspaceFileLoaderContext,
  repoPath: string,
  filePath: string,
  options: { silent?: boolean } = {},
): Promise<void> {
  const conversationId = ctx.getConversationId()
  if (!repoPath || !filePath || !conversationId) return
  const key = workspaceTabKey({ repoPath, filePath })
  const silent = options.silent ?? false

  if (silent) {
    ctx.patchTabFileState(key, { loading: ctx.getCurrentLoading(key) })
  } else {
    ctx.patchTabFileState(key, { loading: true, error: '' })
  }

  try {
    const [previewPayload, patchPayload] = await Promise.all([
      getProjectConversationWorkspaceFilePreview(conversationId, { repoPath, path: filePath }),
      getProjectConversationWorkspaceFilePatch(conversationId, { repoPath, path: filePath }),
    ])
    if (conversationId !== ctx.getConversationId() || !ctx.hasOpenTab(key)) return
    ctx.patchTabFileState(key, {
      preview: previewPayload.filePreview,
      patch: patchPayload.filePatch,
      loading: false,
      error: '',
    })
    ctx.syncEditorFromPreview(repoPath, filePath, previewPayload.filePreview)
  } catch (error) {
    if (conversationId !== ctx.getConversationId() || !ctx.hasOpenTab(key)) return
    ctx.patchTabFileState(key, {
      preview: null,
      patch: null,
      loading: false,
      error: error instanceof Error ? error.message : 'Failed to load the workspace file details.',
    })
  }
}
