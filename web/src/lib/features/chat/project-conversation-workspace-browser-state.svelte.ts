import {
  getProjectConversationWorkspaceDiff,
  type ChatDiffPayload,
  type ProjectConversationWorkspaceBranchScope,
  type ProjectConversationWorkspaceDiff,
  type ProjectConversationWorkspaceGitGraph,
  type ProjectConversationWorkspaceMetadata,
  type ProjectConversationWorkspaceRepoRefs,
  type ProjectConversationWorkspaceSearchResult,
} from '$lib/api/chat'
import { buildProjectConversationWorkspaceBrowserStateView } from './workspace-browser-state-view'
import { readWorkspaceAutosavePreference, storeWorkspaceAutosavePreference } from './workspace-browser-autosave'
import { createWorkspaceBrowserTreeState } from './workspace-browser-tree-state.svelte'
import { createWorkspaceBrowserTabs } from './workspace-browser-tabs.svelte'
import { createWorkspaceFileEditorStore } from './project-conversation-workspace-file-editor-state.svelte'
import { refreshWorkspaceBrowserState } from './project-conversation-workspace-browser-loader'
import { loadWorkspaceFile } from './project-conversation-workspace-data-loader'
import {
  checkoutWorkspaceBranch as runCheckoutWorkspaceBranch,
  computeWorkspaceCheckoutBlockers,
  loadWorkspaceDirEntries,
  refreshWorkspaceRepoGitContext,
  revealWorkspaceFileInTree,
} from './project-conversation-workspace-browser-repo-ops'
import {
  buildWorkspaceFocusContext,
  createWorkspaceFileEntry,
  deleteWorkspaceFileEntry,
  relativeChangedFilePath,
  renameWorkspaceFileEntry,
  reviewWorkspacePatch,
  searchWorkspacePaths,
  workspaceSelectedChangedFiles,
} from './project-conversation-workspace-browser-file-ops'
import {
  areWorkspaceMetadataEqual,
  workspaceTabKey,
  type WorkspaceRecentFile,
  type WorkspaceTab,
  type WorkspaceTabFileState,
} from './project-conversation-workspace-browser-state-helpers'
export type {
  WorkspaceFileEditorState,
  WorkspaceTab,
  WorkspaceTabFileState,
} from './project-conversation-workspace-browser-state-helpers'
export { workspaceTabKey } from './project-conversation-workspace-browser-state-helpers'
export function createProjectConversationWorkspaceBrowserState(input: {
  getConversationId: () => string
  getWorkspaceDiff?: () => ProjectConversationWorkspaceDiff | null
  onWorkspaceDiffUpdated?: (workspaceDiff: ProjectConversationWorkspaceDiff | null) => void
}) {
  let metadata = $state<ProjectConversationWorkspaceMetadata | null>(null)
  let metadataLoading = $state(false)
  let metadataError = $state('')
  const tree = createWorkspaceBrowserTreeState()
  let autosaveEnabled = $state(readWorkspaceAutosavePreference())
  let loadRequestID = 0
  let repoRefs = $state<ProjectConversationWorkspaceRepoRefs | null>(null)
  let repoRefsLoading = $state(false)
  let repoRefsError = $state('')
  let gitGraph = $state<ProjectConversationWorkspaceGitGraph | null>(null)
  let gitGraphLoading = $state(false)
  let gitGraphError = $state('')
  let selectedGitCommitID = $state('')
  let detailMode = $state<'file' | 'git_graph'>('file')
  function loadFile(repoPath: string, filePath: string, options: { silent?: boolean } = {}) {
    return loadWorkspaceFile(
      {
        getConversationId: input.getConversationId,
        hasOpenTab: (key) => tabs.openTabs.some((tab) => workspaceTabKey(tab) === key),
        getCurrentLoading: (key) => tabs.tabFileStates.get(key)?.loading ?? false,
        patchTabFileState,
        syncEditorFromPreview: editorStore.syncFromPreview,
      },
      repoPath,
      filePath,
      options,
    )
  }
  const tabs = createWorkspaceBrowserTabs({
    loadFile,
    renameEditorFileState: (repoPath, fromPath, toPath) =>
      editorStore.renameFileState(repoPath, fromPath, toPath),
  })
  function patchTabFileState(key: string, patch: Partial<WorkspaceTabFileState>) {
    tabs.patchTabFileState(key, patch)
  }
  function getActiveTab(): WorkspaceTab | null {
    return tabs.getActiveTab()
  }
  function getActiveTabFileState(): WorkspaceTabFileState {
    return tabs.getActiveTabFileState()
  }
  function currentWorkspaceDiff() {
    return input.getWorkspaceDiff?.() ?? null
  }
  function setMetadata(nextMetadata: ProjectConversationWorkspaceMetadata) {
    if (!areWorkspaceMetadataEqual(metadata, nextMetadata)) metadata = nextMetadata
  }
  async function refreshWorkspaceDiff() {
    const conversationId = input.getConversationId()
    if (!conversationId || !input.onWorkspaceDiffUpdated) return
    const payload = await getProjectConversationWorkspaceDiff(conversationId)
    input.onWorkspaceDiffUpdated(payload.workspaceDiff)
  }
  const editorStore = createWorkspaceFileEditorStore({
    getConversationId: input.getConversationId,
    getSelectedRepoPath: () => getActiveTab()?.repoPath ?? '',
    getSelectedFilePath: () => getActiveTab()?.filePath ?? '',
    getRepoRefCacheKey: (repoPath) =>
      metadata?.repos.find((repo) => repo.path === repoPath)?.currentRef.cacheKey ?? '',
    getPreview: (repoPath, filePath) => {
      const key = workspaceTabKey({ repoPath, filePath })
      return tabs.tabFileStates.get(key)?.preview ?? null
    },
    setPreview: (repoPath, filePath, preview) => {
      const key = workspaceTabKey({ repoPath, filePath })
      patchTabFileState(key, { preview })
    },
    reloadSelectedFile: async (repoPath, filePath) => {
      if (repoPath && filePath) await loadFile(repoPath, filePath, { silent: true })
    },
    refreshWorkspaceDiff,
    getAutosaveEnabled: () => autosaveEnabled,
  })
  function reset() {
    metadata = null
    metadataLoading = false
    metadataError = ''
    tree.reset()
    tabs.resetSelection()
    repoRefs = null
    repoRefsLoading = false
    repoRefsError = ''
    gitGraph = null
    gitGraphLoading = false
    gitGraphError = ''
    selectedGitCommitID = ''
    detailMode = 'file'
    editorStore.reset()
  }
  async function refreshWorkspace(preserveSelection: boolean) {
    const conversationId = input.getConversationId()
    const requestID = ++loadRequestID
    await refreshWorkspaceBrowserState({
      conversationId,
      requestID,
      getCurrentRequestID: () => loadRequestID,
      getCurrentConversationId: input.getConversationId,
      preserveSelection,
      treeRepoPath: tabs.treeRepoPath,
      treeNodes: tree.treeNodes,
      expandedDirs: tree.expandedDirs,
      openTabs: tabs.openTabs,
      setMetadataLoading: (loading) => {
        metadataLoading = loading
      },
      setMetadataError: (error) => {
        metadataError = error
      },
      setMetadata,
      clearMetadata: () => {
        metadata = null
      },
      setTreeRepoPath: (repoPath) => {
        tabs.setTreeRepoPath(repoPath)
      },
      resetTreeState: () => {
        tree.reset()
      },
      closeAllTabs: tabs.closeAllTabs,
      clearGitContext: () => {
        repoRefs = null
        repoRefsError = ''
        gitGraph = null
        gitGraphError = ''
        selectedGitCommitID = ''
      },
      loadDirEntries,
      loadFile,
      refreshRepoGitContext,
    })
  }
  async function refreshRepoGitContext(repoPath = tabs.treeRepoPath) {
    const conversationId = input.getConversationId()
    await refreshWorkspaceRepoGitContext({
      conversationId,
      repoPath,
      treeRepoPath: tabs.treeRepoPath,
      selectedGitCommitID,
      setRepoRefsLoading: (loading) => {
        repoRefsLoading = loading
      },
      setRepoRefsError: (error) => {
        repoRefsError = error
      },
      setRepoRefs: (value) => {
        repoRefs = value
      },
      setGitGraphLoading: (loading) => {
        gitGraphLoading = loading
      },
      setGitGraphError: (error) => {
        gitGraphError = error
      },
      setGitGraph: (value) => {
        gitGraph = value
      },
      setSelectedGitCommitID: (commitId) => {
        selectedGitCommitID = commitId
      },
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
      repoPath: tabs.treeRepoPath,
      dirPath,
      requestID: externalRequestID ?? loadRequestID,
      currentRequestID: loadRequestID,
      silent: options.silent ?? false,
      treeRepoPath: tabs.treeRepoPath,
      setDirLoading: tree.setDirLoading,
      setTreeEntries: tree.setTreeEntries,
    })
  }
  async function toggleDir(dirPath: string) {
    if (tree.expandedDirs.has(dirPath)) {
      tree.setDirExpanded(dirPath, false)
      return
    }
    tree.setDirExpanded(dirPath, true)
    if (!tree.treeNodes.has(dirPath)) await loadDirEntries(dirPath)
  }
  async function revealFileInTree(
    path: string,
    options: { requestID?: number; silent?: boolean } = {},
  ) {
    await revealWorkspaceFileInTree({
      path,
      requestID: options.requestID ?? loadRequestID,
      currentRequestID: () => loadRequestID,
      hasTreeEntries: (dirPath) => tree.treeNodes.has(dirPath),
      setDirExpanded: tree.setDirExpanded,
      loadDirEntries,
      options,
    })
  }
  async function reloadFile(repoPath: string, filePath: string) {
    if (!repoPath || !filePath) return
    await loadFile(repoPath, filePath, { silent: true })
  }
  function openRepo(repoPath: string) {
    if (!repoPath || repoPath === tabs.treeRepoPath) return
    tabs.setTreeRepoPath(repoPath)
    tree.reset()
    void loadDirEntries('')
    void refreshRepoGitContext(repoPath)
  }
  function selectFile(path: string) {
    const repoPath = tabs.treeRepoPath
    if (!path || !repoPath) return
    detailMode = 'file'
    const requestID = ++loadRequestID
    void revealFileInTree(path, { requestID, silent: true })
    tabs.openTab(repoPath, path)
  }
  async function createFile(path: string) {
    return createWorkspaceFileEntry({
      conversationId: input.getConversationId(),
      repoPath: tabs.treeRepoPath,
      path,
      refreshWorkspace,
      selectFile,
    })
  }
  async function searchPaths(
    query: string,
    limit = 20,
  ): Promise<ProjectConversationWorkspaceSearchResult[]> {
    return searchWorkspacePaths({
      conversationId: input.getConversationId(),
      repoPath: tabs.treeRepoPath,
      query,
      limit,
    })
  }
  function checkoutBlockers(repoPath: string): string[] {
    return computeWorkspaceCheckoutBlockers({
      repoPath,
      metadata,
      workspaceDiff: currentWorkspaceDiff(),
      openTabs: tabs.openTabs,
      getEditorState: editorStore.getEditorState,
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
      repoRefs,
      metadata,
      workspaceDiff: currentWorkspaceDiff(),
      openTabs: tabs.openTabs,
      getEditorState: editorStore.getEditorState,
      setMetadata,
      refreshWorkspace,
      refreshRepoGitContext: async (repoPath) => {
        await refreshRepoGitContext(repoPath)
        await refreshWorkspaceDiff()
      },
    })
  }
  async function renameFile(fromPath: string, toPath: string) {
    const repoPath = tabs.treeRepoPath
    return renameWorkspaceFileEntry({
      conversationId: input.getConversationId(),
      repoPath,
      fromPath,
      toPath,
      remapTabPath: () => tabs.remapTabPath(repoPath, fromPath, toPath),
      refreshWorkspace,
      activateTab: tabs.activateTab,
      getActiveTabKey: () => tabs.activeTabKey,
      loadFile,
    })
  }
  async function deleteFile(path: string) {
    return deleteWorkspaceFileEntry({
      conversationId: input.getConversationId(),
      repoPath: tabs.treeRepoPath,
      path,
      discardDraft: editorStore.discardDraft,
      closeTab: tabs.closeTab,
      refreshWorkspace,
    })
  }
  function setAutosaveEnabled(enabled: boolean) {
    autosaveEnabled = enabled
    storeWorkspaceAutosavePreference(enabled)
  }
  function activeFilePath() {
    return tabs.activeFilePath(tabs.treeRepoPath)
  }
  function selectRelativeChangedFile(offset: 1 | -1) {
    const nextPath = relativeChangedFilePath({
      selectedChangedFiles: workspaceSelectedChangedFiles({
        repoPath: tabs.treeRepoPath,
        activeFilePath: activeFilePath(),
        workspaceDiff: currentWorkspaceDiff(),
      }),
      activeFilePath: activeFilePath(),
      offset,
    })
    if (nextPath) selectFile(nextPath)
  }
  async function reviewPatch(diff: ChatDiffPayload, options: { autoApply?: boolean } = {}) {
    return reviewWorkspacePatch({
      repoPath: tabs.treeRepoPath,
      diff,
      autoApply: options.autoApply ?? false,
      openTab: tabs.openTab,
      loadFile,
      reviewPatch: editorStore.reviewPatch,
      applyPendingPatch: editorStore.applyPendingPatch,
    })
  }
  return buildProjectConversationWorkspaceBrowserStateView({
    getMetadata: () => metadata,
    getMetadataLoading: () => metadataLoading,
    getMetadataError: () => metadataError,
    getTreeNodes: () => tree.treeNodes,
    getExpandedDirs: () => tree.expandedDirs,
    getLoadingDirs: () => tree.loadingDirs,
    getOpenTabs: () => tabs.openTabs,
    getActiveTabKey: () => tabs.activeTabKey,
    getTabFileStates: () => tabs.tabFileStates,
    getRecentFiles: () => tabs.recentFiles,
    getAutosaveEnabled: () => autosaveEnabled,
    getDetailMode: () => detailMode,
    getRepoRefs: () => repoRefs,
    getRepoRefsLoading: () => repoRefsLoading,
    getRepoRefsError: () => repoRefsError,
    getGitGraph: () => gitGraph,
    getGitGraphLoading: () => gitGraphLoading,
    getGitGraphError: () => gitGraphError,
    getSelectedGitCommit: () =>
      gitGraph?.commits.find((commit) => commit.commitId === selectedGitCommitID) ?? null,
    getHasDirtyTabs: () =>
      tabs.openTabs.some((tab) => editorStore.getEditorState(tab.repoPath, tab.filePath)?.dirty === true),
    getPreview: () => getActiveTabFileState().preview,
    getPatch: () => getActiveTabFileState().patch,
    getFileLoading: () => getActiveTabFileState().loading,
    getFileError: () => getActiveTabFileState().error,
    getSelectedRepoPath: () => tabs.treeRepoPath,
    getSelectedFilePath: activeFilePath,
    getSelectedEditorState: () => editorStore.selectedEditorState,
    getSelectedDraftLineDiff: () => editorStore.selectedDraftLineDiff,
    getSelectedChangedFiles: () =>
      workspaceSelectedChangedFiles({
        repoPath: tabs.treeRepoPath,
        activeFilePath: activeFilePath(),
        workspaceDiff: currentWorkspaceDiff(),
      }),
    getSelectedFocusContext: () =>
      buildWorkspaceFocusContext({
        selectedEditorState: editorStore.selectedEditorState,
        hasActiveTab: getActiveTab() != null,
        recentFiles: tabs.recentFiles as WorkspaceRecentFile[],
        buildWorkingSet: editorStore.buildWorkingSet,
      }),
    getEditorState: editorStore.getEditorState,
    reset,
    refreshWorkspace,
    refreshRepoGitContext,
    reloadFile,
    toggleDir,
    openRepo,
    selectFile,
    selectGitCommit: (commitId: string) => (selectedGitCommitID = commitId),
    setDetailMode: (mode: 'file' | 'git_graph') => (detailMode = mode),
    searchPaths,
    openTab: (repoPath: string, filePath: string) => {
      detailMode = 'file'
      tabs.openTab(repoPath, filePath)
    },
    closeTab: tabs.closeTab,
    closeAllTabs: tabs.closeAllTabs,
    activateTab: tabs.activateTab,
    createFile,
    renameFile,
    deleteFile,
    refreshWorkspaceDiff,
    checkoutBlockers,
    checkoutBranch,
    setAutosaveEnabled,
    selectNextChangedFile: () => selectRelativeChangedFile(1),
    selectPreviousChangedFile: () => selectRelativeChangedFile(-1),
    reviewPatch,
    applySelectedPendingPatch: () => editorStore.applyPendingPatch(tabs.treeRepoPath, activeFilePath()),
    discardSelectedPendingPatch: editorStore.discardPendingPatch,
    updateSelectedDraft: editorStore.updateSelectedDraft,
    updateSelectedSelection: editorStore.updateSelectedSelection,
    formatSelectedDocument: editorStore.formatSelectedDocument,
    formatSelectedSelection: editorStore.formatSelectedSelection,
    revertSelectedDraft: editorStore.revertSelectedDraft,
    keepSelectedDraft: editorStore.keepSelectedDraft,
    reloadSelectedSavedVersion: editorStore.reloadSelectedSavedVersion,
    saveSelectedFile: editorStore.saveSelectedFile,
    saveFile: editorStore.saveFile,
    discardDraft: editorStore.discardDraft,
  })
}
export type ProjectConversationWorkspaceBrowserState = ReturnType<
  typeof createProjectConversationWorkspaceBrowserState
>
