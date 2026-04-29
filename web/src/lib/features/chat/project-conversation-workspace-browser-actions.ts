import type {
  ChatDiffPayload,
  ProjectConversationWorkspaceBranchScope,
  ProjectConversationWorkspaceDiff,
  ProjectConversationWorkspaceMetadata,
  ProjectConversationWorkspaceRepoRefs,
  ProjectConversationWorkspaceSearchResult,
} from '$lib/api/chat'
import {
  checkoutWorkspaceBranch as runCheckoutWorkspaceBranch,
  computeWorkspaceCheckoutBlockers,
} from './project-conversation-workspace-browser-repo-ops'
import {
  createWorkspaceFileEntry,
  deleteWorkspaceFileEntry,
  relativeChangedFilePath,
  renameWorkspaceFileEntry,
  reviewWorkspacePatch,
  searchWorkspacePaths,
  workspaceSelectedChangedFiles,
} from './project-conversation-workspace-browser-file-ops'
import type {
  WorkspaceFileEditorState,
  WorkspaceTab,
} from './project-conversation-workspace-browser-state-helpers'

type WorkspaceBrowserEditorStore = {
  getEditorState: (repoPath?: string, filePath?: string) => WorkspaceFileEditorState | null
  reviewPatch: (repoPath: string, filePath: string, diff: ChatDiffPayload) => boolean
  applyPendingPatch: (repoPath: string, filePath: string) => boolean
  discardDraft: (repoPath: string, filePath: string) => void
}

export function createWorkspaceBrowserActions(input: {
  getConversationId: () => string
  currentWorkspaceDiff: () => ProjectConversationWorkspaceDiff | null
  getMetadata: () => ProjectConversationWorkspaceMetadata | null
  setMetadata: (nextMetadata: ProjectConversationWorkspaceMetadata) => void
  getRepoRefs: () => ProjectConversationWorkspaceRepoRefs | null
  getTreeRepoPath: () => string
  setTreeRepoPath: (repoPath: string) => void
  resetTree: () => void
  getOpenTabs: () => WorkspaceTab[]
  getActiveFilePath: () => string
  getActiveTabKey: () => string
  reserveRequestID: () => number
  setDetailMode: (mode: 'file' | 'git_graph') => void
  revealFileInTree: (
    path: string,
    options?: { requestID?: number; silent?: boolean },
  ) => Promise<void>
  refreshRepoGitContext: (repoPath?: string) => Promise<void>
  refreshRepoGitContextAfterCheckout?: (repoPath?: string) => Promise<void>
  refreshWorkspace: (preserveSelection: boolean) => Promise<void>
  loadDirEntries: (
    dirPath: string,
    requestID?: number,
    options?: { silent?: boolean },
  ) => Promise<void>
  loadFile: (repoPath: string, filePath: string, options?: { silent?: boolean }) => Promise<void>
  openTab: (repoPath: string, filePath: string) => void
  activateTab: (repoPath: string, filePath: string) => void
  closeTab: (repoPath: string, filePath: string) => void
  remapTabPath: (repoPath: string, fromPath: string, toPath: string) => void
  editorStore: WorkspaceBrowserEditorStore
}) {
  function activeFilePath() {
    return input.getActiveFilePath()
  }

  function selectFile(path: string) {
    const repoPath = input.getTreeRepoPath()
    if (!path || !repoPath) return
    input.setDetailMode('file')
    const requestID = input.reserveRequestID()
    void input.revealFileInTree(path, { requestID, silent: true })
    input.openTab(repoPath, path)
  }

  function openRepo(repoPath: string) {
    if (!repoPath || repoPath === input.getTreeRepoPath()) return
    input.setTreeRepoPath(repoPath)
    input.resetTree()
    void input.loadDirEntries('')
    void input.refreshRepoGitContext(repoPath)
  }

  async function createFile(path: string) {
    return createWorkspaceFileEntry({
      conversationId: input.getConversationId(),
      repoPath: input.getTreeRepoPath(),
      path,
      refreshWorkspace: input.refreshWorkspace,
      selectFile,
    })
  }

  async function searchPaths(
    query: string,
    limit = 20,
  ): Promise<ProjectConversationWorkspaceSearchResult[]> {
    return searchWorkspacePaths({
      conversationId: input.getConversationId(),
      repoPath: input.getTreeRepoPath(),
      query,
      limit,
    })
  }

  function checkoutBlockers(repoPath: string): string[] {
    return computeWorkspaceCheckoutBlockers({
      repoPath,
      metadata: input.getMetadata(),
      workspaceDiff: input.currentWorkspaceDiff(),
      openTabs: input.getOpenTabs(),
      getEditorState: input.editorStore.getEditorState,
    })
  }

  async function checkoutBranch(request: {
    repoPath: string
    targetKind: ProjectConversationWorkspaceBranchScope
    targetName: string
    createTrackingBranch: boolean
    localBranchName?: string
  }): Promise<{
    ok: boolean
    blockers: string[]
  }> {
    const conversationId = input.getConversationId()
    if (!conversationId || !request.repoPath) {
      return { ok: false, blockers: ['Workspace conversation is unavailable.'] }
    }
    return runCheckoutWorkspaceBranch({
      conversationId,
      ...request,
      repoRefs: input.getRepoRefs(),
      metadata: input.getMetadata(),
      workspaceDiff: input.currentWorkspaceDiff(),
      openTabs: input.getOpenTabs(),
      getEditorState: input.editorStore.getEditorState,
      setMetadata: input.setMetadata,
      refreshWorkspace: input.refreshWorkspace,
      refreshRepoGitContext:
        input.refreshRepoGitContextAfterCheckout ?? input.refreshRepoGitContext,
    })
  }

  async function renameFile(fromPath: string, toPath: string) {
    const repoPath = input.getTreeRepoPath()
    return renameWorkspaceFileEntry({
      conversationId: input.getConversationId(),
      repoPath,
      fromPath,
      toPath,
      remapTabPath: () => input.remapTabPath(repoPath, fromPath, toPath),
      refreshWorkspace: input.refreshWorkspace,
      activateTab: input.activateTab,
      getActiveTabKey: input.getActiveTabKey,
      loadFile: input.loadFile,
    })
  }

  async function deleteFile(path: string) {
    return deleteWorkspaceFileEntry({
      conversationId: input.getConversationId(),
      repoPath: input.getTreeRepoPath(),
      path,
      discardDraft: input.editorStore.discardDraft,
      closeTab: input.closeTab,
      refreshWorkspace: input.refreshWorkspace,
    })
  }

  function selectRelativeChangedFile(offset: 1 | -1) {
    const nextPath = relativeChangedFilePath({
      selectedChangedFiles: workspaceSelectedChangedFiles({
        repoPath: input.getTreeRepoPath(),
        activeFilePath: activeFilePath(),
        workspaceDiff: input.currentWorkspaceDiff(),
      }),
      activeFilePath: activeFilePath(),
      offset,
    })
    if (nextPath) selectFile(nextPath)
  }

  async function reviewPatch(diff: ChatDiffPayload, options: { autoApply?: boolean } = {}) {
    return reviewWorkspacePatch({
      repoPath: input.getTreeRepoPath(),
      diff,
      autoApply: options.autoApply ?? false,
      openTab: input.openTab,
      loadFile: input.loadFile,
      reviewPatch: input.editorStore.reviewPatch,
      applyPendingPatch: input.editorStore.applyPendingPatch,
    })
  }

  return {
    openRepo,
    selectFile,
    createFile,
    searchPaths,
    checkoutBlockers,
    checkoutBranch,
    renameFile,
    deleteFile,
    selectNextChangedFile: () => selectRelativeChangedFile(1),
    selectPreviousChangedFile: () => selectRelativeChangedFile(-1),
    reviewPatch,
  }
}
