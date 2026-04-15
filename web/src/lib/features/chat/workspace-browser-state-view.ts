import type {
  ChatDiffPayload,
  ProjectConversationWorkspaceBranchScope,
  ProjectConversationWorkspaceFilePatch,
  ProjectConversationWorkspaceFilePreview,
  ProjectConversationWorkspaceGitGraph,
  ProjectConversationWorkspaceGitGraphCommit,
  ProjectConversationWorkspaceMetadata,
  ProjectConversationWorkspaceRepoRefs,
  ProjectConversationWorkspaceSearchResult,
  ProjectConversationWorkspaceTreeEntry,
} from '$lib/api/chat'
import type {
  WorkspaceFileEditorState,
  WorkspaceFileLineDiffMarkers,
  WorkspaceTab,
  WorkspaceTabFileState,
} from './project-conversation-workspace-browser-state-helpers'

export function buildProjectConversationWorkspaceBrowserStateView<
  TSelectedChangedFiles,
  TFocusContext,
  TSelection,
>(input: {
  getMetadata: () => ProjectConversationWorkspaceMetadata | null
  getMetadataLoading: () => boolean
  getMetadataError: () => string
  getTreeNodes: () => Map<string, ProjectConversationWorkspaceTreeEntry[]>
  getExpandedDirs: () => Set<string>
  getLoadingDirs: () => Set<string>
  getOpenTabs: () => WorkspaceTab[]
  getActiveTabKey: () => string
  getTabFileStates: () => Map<string, WorkspaceTabFileState>
  getRecentFiles: () => Array<{ repoPath: string; filePath: string }>
  getAutosaveEnabled: () => boolean
  getDetailMode: () => 'file' | 'git_graph'
  getRepoRefs: () => ProjectConversationWorkspaceRepoRefs | null
  getRepoRefsLoading: () => boolean
  getRepoRefsError: () => string
  getGitGraph: () => ProjectConversationWorkspaceGitGraph | null
  getGitGraphLoading: () => boolean
  getGitGraphError: () => string
  getSelectedGitCommit: () => ProjectConversationWorkspaceGitGraphCommit | null
  getHasDirtyTabs: () => boolean
  getPreview: () => ProjectConversationWorkspaceFilePreview | null
  getPatch: () => ProjectConversationWorkspaceFilePatch | null
  getFileLoading: () => boolean
  getFileError: () => string
  getSelectedRepoPath: () => string
  getSelectedFilePath: () => string
  getSelectedEditorState: () => WorkspaceFileEditorState | null
  getSelectedDraftLineDiff: () => WorkspaceFileLineDiffMarkers | null
  getSelectedChangedFiles: () => TSelectedChangedFiles
  getSelectedFocusContext: () => TFocusContext
  getEditorState: (repoPath?: string, filePath?: string) => WorkspaceFileEditorState | null
  reset: () => void
  refreshWorkspace: (preserveSelection: boolean) => Promise<void>
  refreshRepoGitContext: (repoPath?: string) => Promise<void>
  reloadFile: (repoPath: string, filePath: string) => Promise<void>
  toggleDir: (dirPath: string) => Promise<void>
  openRepo: (repoPath: string) => void
  selectFile: (path: string) => void
  selectGitCommit: (commitId: string) => void
  setDetailMode: (mode: 'file' | 'git_graph') => void
  searchPaths: (query: string, limit?: number) => Promise<ProjectConversationWorkspaceSearchResult[]>
  openTab: (repoPath: string, filePath: string) => void
  closeTab: (repoPath: string, filePath: string) => void
  closeAllTabs: () => void
  activateTab: (repoPath: string, filePath: string) => void
  createFile: (path: string) => Promise<unknown>
  renameFile: (fromPath: string, toPath: string) => Promise<unknown>
  deleteFile: (path: string) => Promise<unknown>
  refreshWorkspaceDiff: () => Promise<void>
  checkoutBlockers: (repoPath: string) => string[]
  checkoutBranch: (request: {
    repoPath: string
    targetKind: ProjectConversationWorkspaceBranchScope
    targetName: string
    createTrackingBranch: boolean
    localBranchName?: string
  }) => Promise<{ ok: boolean; blockers: string[] }>
  setAutosaveEnabled: (enabled: boolean) => void
  selectNextChangedFile: () => void
  selectPreviousChangedFile: () => void
  reviewPatch: (diff: ChatDiffPayload, options?: { autoApply?: boolean }) => Promise<unknown>
  applySelectedPendingPatch: () => boolean
  discardSelectedPendingPatch: (repoPath?: string, filePath?: string) => void
  updateSelectedDraft: (nextDraftContent: string) => void
  updateSelectedSelection: (selection: TSelection) => void
  formatSelectedDocument: () => boolean
  formatSelectedSelection: () => boolean
  revertSelectedDraft: () => void
  keepSelectedDraft: () => void
  reloadSelectedSavedVersion: () => void
  saveSelectedFile: () => Promise<boolean>
  saveFile: (repoPath: string, filePath: string) => Promise<boolean>
  discardDraft: (repoPath: string, filePath: string) => void
}) {
  return {
    get metadata() {
      return input.getMetadata()
    },
    get metadataLoading() {
      return input.getMetadataLoading()
    },
    get metadataError() {
      return input.getMetadataError()
    },
    get treeNodes() {
      return input.getTreeNodes()
    },
    get expandedDirs() {
      return input.getExpandedDirs()
    },
    get loadingDirs() {
      return input.getLoadingDirs()
    },
    get openTabs() {
      return input.getOpenTabs()
    },
    get activeTabKey() {
      return input.getActiveTabKey()
    },
    get tabFileStates() {
      return input.getTabFileStates()
    },
    get recentFiles() {
      return input.getRecentFiles()
    },
    get autosaveEnabled() {
      return input.getAutosaveEnabled()
    },
    get detailMode() {
      return input.getDetailMode()
    },
    get repoRefs() {
      return input.getRepoRefs()
    },
    get repoRefsLoading() {
      return input.getRepoRefsLoading()
    },
    get repoRefsError() {
      return input.getRepoRefsError()
    },
    get gitGraph() {
      return input.getGitGraph()
    },
    get gitGraphLoading() {
      return input.getGitGraphLoading()
    },
    get gitGraphError() {
      return input.getGitGraphError()
    },
    get selectedGitCommit() {
      return input.getSelectedGitCommit()
    },
    get hasDirtyTabs() {
      return input.getHasDirtyTabs()
    },
    get preview() {
      return input.getPreview()
    },
    get patch() {
      return input.getPatch()
    },
    get fileLoading() {
      return input.getFileLoading()
    },
    get fileError() {
      return input.getFileError()
    },
    get selectedRepoPath() {
      return input.getSelectedRepoPath()
    },
    get selectedFilePath() {
      return input.getSelectedFilePath()
    },
    get selectedEditorState() {
      return input.getSelectedEditorState()
    },
    get selectedDraftLineDiff() {
      return input.getSelectedDraftLineDiff()
    },
    get selectedChangedFiles() {
      return input.getSelectedChangedFiles()
    },
    getSelectedFocusContext: input.getSelectedFocusContext,
    getEditorState: input.getEditorState,
    reset: input.reset,
    refreshWorkspace: input.refreshWorkspace,
    refreshRepoGitContext: input.refreshRepoGitContext,
    reloadFile: input.reloadFile,
    toggleDir: input.toggleDir,
    openRepo: input.openRepo,
    selectFile: input.selectFile,
    selectGitCommit: input.selectGitCommit,
    setDetailMode: input.setDetailMode,
    searchPaths: input.searchPaths,
    openTab: input.openTab,
    closeTab: input.closeTab,
    closeAllTabs: input.closeAllTabs,
    activateTab: input.activateTab,
    createFile: input.createFile,
    renameFile: input.renameFile,
    deleteFile: input.deleteFile,
    refreshWorkspaceDiff: input.refreshWorkspaceDiff,
    checkoutBlockers: input.checkoutBlockers,
    checkoutBranch: input.checkoutBranch,
    setAutosaveEnabled: input.setAutosaveEnabled,
    selectNextChangedFile: input.selectNextChangedFile,
    selectPreviousChangedFile: input.selectPreviousChangedFile,
    reviewPatch: input.reviewPatch,
    applySelectedPendingPatch: input.applySelectedPendingPatch,
    discardSelectedPendingPatch: input.discardSelectedPendingPatch,
    updateSelectedDraft: input.updateSelectedDraft,
    updateSelectedSelection: input.updateSelectedSelection,
    formatSelectedDocument: input.formatSelectedDocument,
    formatSelectedSelection: input.formatSelectedSelection,
    revertSelectedDraft: input.revertSelectedDraft,
    keepSelectedDraft: input.keepSelectedDraft,
    reloadSelectedSavedVersion: input.reloadSelectedSavedVersion,
    saveSelectedFile: input.saveSelectedFile,
    saveFile: input.saveFile,
    discardDraft: input.discardDraft,
  }
}
