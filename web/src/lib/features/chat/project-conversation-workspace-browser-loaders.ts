import {
  getProjectConversationWorkspaceDiff,
  type ProjectConversationWorkspaceDiff,
  type ProjectConversationWorkspaceGitGraph,
  type ProjectConversationWorkspaceMetadata,
  type ProjectConversationWorkspaceRepoRefs,
  type ProjectConversationWorkspaceTreeEntry,
} from '$lib/api/chat'
import { refreshWorkspaceBrowserState } from './project-conversation-workspace-browser-loader'
import {
  loadWorkspaceDirEntries,
  refreshWorkspaceRepoGitContext,
  revealWorkspaceFileInTree,
} from './project-conversation-workspace-browser-repo-ops'

type WorkspaceFileLoader = (
  repoPath: string,
  filePath: string,
  options?: { silent?: boolean },
) => Promise<void>

export function createWorkspaceBrowserLoaders(input: {
  getConversationId: () => string
  getCurrentRequestID: () => number
  reserveRequestID: () => number
  onWorkspaceDiffUpdated?: (workspaceDiff: ProjectConversationWorkspaceDiff | null) => void
  getTreeRepoPath: () => string
  setTreeRepoPath: (repoPath: string) => void
  getTreeNodes: () => Map<string, ProjectConversationWorkspaceTreeEntry[]>
  getExpandedDirs: () => Set<string>
  getOpenTabs: () => Array<{ repoPath: string; filePath: string }>
  setMetadataLoading: (loading: boolean) => void
  setMetadataError: (error: string) => void
  setMetadata: (metadata: ProjectConversationWorkspaceMetadata) => void
  clearMetadata: () => void
  resetTreeState: () => void
  closeAllTabs: () => void
  setRepoRefsLoading: (loading: boolean) => void
  setRepoRefsError: (error: string) => void
  setRepoRefs: (repoRefs: ProjectConversationWorkspaceRepoRefs | null) => void
  setGitGraphLoading: (loading: boolean) => void
  setGitGraphError: (error: string) => void
  setGitGraph: (gitGraph: ProjectConversationWorkspaceGitGraph | null) => void
  getSelectedGitCommitID: () => string
  setSelectedGitCommitID: (commitId: string) => void
  setDirLoading: (dirPath: string, loading: boolean) => void
  setTreeEntries: (dirPath: string, entries: ProjectConversationWorkspaceTreeEntry[]) => void
  setDirExpanded: (dirPath: string, expanded: boolean) => void
  loadFile: WorkspaceFileLoader
}) {
  async function refreshWorkspaceDiff() {
    const conversationId = input.getConversationId()
    if (!conversationId || !input.onWorkspaceDiffUpdated) return
    const payload = await getProjectConversationWorkspaceDiff(conversationId)
    input.onWorkspaceDiffUpdated(payload.workspaceDiff)
  }

  async function refreshRepoGitContext(repoPath = input.getTreeRepoPath()) {
    const conversationId = input.getConversationId()
    await refreshWorkspaceRepoGitContext({
      conversationId,
      repoPath,
      treeRepoPath: input.getTreeRepoPath(),
      selectedGitCommitID: input.getSelectedGitCommitID(),
      setRepoRefsLoading: input.setRepoRefsLoading,
      setRepoRefsError: input.setRepoRefsError,
      setRepoRefs: input.setRepoRefs,
      setGitGraphLoading: input.setGitGraphLoading,
      setGitGraphError: input.setGitGraphError,
      setGitGraph: input.setGitGraph,
      setSelectedGitCommitID: input.setSelectedGitCommitID,
      isCurrentConversation: () => input.getConversationId() === conversationId,
    })
  }

  async function loadDirEntries(
    dirPath: string,
    externalRequestID?: number,
    options: { silent?: boolean } = {},
  ) {
    await loadWorkspaceDirEntries({
      conversationId: input.getConversationId(),
      repoPath: input.getTreeRepoPath(),
      dirPath,
      requestID: externalRequestID ?? input.getCurrentRequestID(),
      currentRequestID: input.getCurrentRequestID(),
      silent: options.silent ?? false,
      treeRepoPath: input.getTreeRepoPath(),
      setDirLoading: input.setDirLoading,
      setTreeEntries: input.setTreeEntries,
    })
  }

  async function toggleDir(dirPath: string) {
    const expandedDirs = input.getExpandedDirs()
    if (expandedDirs.has(dirPath)) {
      input.setDirExpanded(dirPath, false)
      return
    }
    input.setDirExpanded(dirPath, true)
    if (!input.getTreeNodes().has(dirPath)) await loadDirEntries(dirPath)
  }

  async function revealFileInTree(
    path: string,
    options: { requestID?: number; silent?: boolean } = {},
  ) {
    await revealWorkspaceFileInTree({
      path,
      requestID: options.requestID ?? input.getCurrentRequestID(),
      currentRequestID: input.getCurrentRequestID,
      hasTreeEntries: (dirPath) => input.getTreeNodes().has(dirPath),
      setDirExpanded: input.setDirExpanded,
      loadDirEntries,
      options,
    })
  }

  async function reloadFile(repoPath: string, filePath: string) {
    if (!repoPath || !filePath) return
    await input.loadFile(repoPath, filePath, { silent: true })
  }

  async function refreshWorkspace(preserveSelection: boolean) {
    const conversationId = input.getConversationId()
    const requestID = input.reserveRequestID()
    await refreshWorkspaceBrowserState({
      conversationId,
      requestID,
      getCurrentRequestID: input.getCurrentRequestID,
      getCurrentConversationId: input.getConversationId,
      preserveSelection,
      treeRepoPath: input.getTreeRepoPath(),
      treeNodes: input.getTreeNodes(),
      expandedDirs: input.getExpandedDirs(),
      openTabs: input.getOpenTabs(),
      setMetadataLoading: input.setMetadataLoading,
      setMetadataError: input.setMetadataError,
      setMetadata: input.setMetadata,
      clearMetadata: input.clearMetadata,
      setTreeRepoPath: input.setTreeRepoPath,
      resetTreeState: input.resetTreeState,
      closeAllTabs: input.closeAllTabs,
      clearGitContext: () => {
        input.setRepoRefs(null)
        input.setRepoRefsError('')
        input.setGitGraph(null)
        input.setGitGraphError('')
        input.setSelectedGitCommitID('')
      },
      loadDirEntries,
      loadFile: input.loadFile,
      refreshRepoGitContext,
    })
  }

  return {
    refreshWorkspaceDiff,
    refreshRepoGitContext,
    loadDirEntries,
    toggleDir,
    revealFileInTree,
    reloadFile,
    refreshWorkspace,
  }
}
