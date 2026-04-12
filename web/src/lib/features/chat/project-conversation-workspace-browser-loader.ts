import {
  getProjectConversationWorkspace,
  type ProjectConversationWorkspaceMetadata,
  type ProjectConversationWorkspaceTreeEntry,
} from '$lib/api/chat'
import type { WorkspaceTab } from './project-conversation-workspace-browser-state-helpers'

export async function refreshWorkspaceBrowserState(input: {
  conversationId: string
  requestID: number
  getCurrentRequestID: () => number
  getCurrentConversationId: () => string
  preserveSelection: boolean
  treeRepoPath: string
  treeNodes: Map<string, ProjectConversationWorkspaceTreeEntry[]>
  expandedDirs: Set<string>
  openTabs: WorkspaceTab[]
  setMetadataLoading: (loading: boolean) => void
  setMetadataError: (error: string) => void
  setMetadata: (metadata: ProjectConversationWorkspaceMetadata) => void
  clearMetadata: () => void
  setTreeRepoPath: (repoPath: string) => void
  resetTreeState: () => void
  closeAllTabs: () => void
  clearGitContext: () => void
  loadDirEntries: (
    dirPath: string,
    requestID?: number,
    options?: { silent?: boolean },
  ) => Promise<void>
  loadFile: (repoPath: string, filePath: string, options?: { silent?: boolean }) => Promise<void>
  refreshRepoGitContext: (repoPath?: string) => Promise<void>
}) {
  input.setMetadataLoading(true)
  input.setMetadataError('')

  try {
    const payload = await getProjectConversationWorkspace(input.conversationId)
    if (
      input.requestID !== input.getCurrentRequestID() ||
      input.conversationId !== input.getCurrentConversationId()
    ) {
      return
    }

    input.setMetadata(payload.workspace)
    if (!payload.workspace.available || payload.workspace.repos.length === 0) {
      input.setTreeRepoPath('')
      input.resetTreeState()
      input.clearGitContext()
      input.closeAllTabs()
      return
    }

    const nextRepoPath =
      input.preserveSelection &&
      payload.workspace.repos.some((repo) => repo.path === input.treeRepoPath) &&
      input.treeRepoPath
        ? input.treeRepoPath
        : (payload.workspace.repos[0]?.path ?? '')

    const repoChanged = nextRepoPath !== input.treeRepoPath
    const prevExpanded = repoChanged ? [] : [...input.expandedDirs]
    input.setTreeRepoPath(nextRepoPath)

    if (repoChanged) {
      input.resetTreeState()
    }

    await input.loadDirEntries('', input.requestID, { silent: input.treeNodes.has('') })
    if (input.requestID !== input.getCurrentRequestID()) {
      return
    }

    if (prevExpanded.length > 0) {
      await Promise.all(
        prevExpanded.map((dirPath) =>
          input.loadDirEntries(dirPath, input.requestID, {
            silent: input.treeNodes.has(dirPath),
          }),
        ),
      )
    }

    if (input.preserveSelection && input.openTabs.length > 0) {
      await Promise.all(
        input.openTabs.map((tab) => input.loadFile(tab.repoPath, tab.filePath, { silent: true })),
      )
    }

    await input.refreshRepoGitContext(nextRepoPath)
  } catch (error) {
    if (
      input.requestID !== input.getCurrentRequestID() ||
      input.conversationId !== input.getCurrentConversationId()
    ) {
      return
    }
    input.clearMetadata()
    input.resetTreeState()
    input.closeAllTabs()
    input.clearGitContext()
    input.setMetadataError(
      error instanceof Error ? error.message : 'Failed to load the Project AI workspace.',
    )
  } finally {
    if (
      input.requestID === input.getCurrentRequestID() &&
      input.conversationId === input.getCurrentConversationId()
    ) {
      input.setMetadataLoading(false)
    }
  }
}
