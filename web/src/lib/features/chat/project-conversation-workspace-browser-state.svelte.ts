import { type ProjectConversationWorkspaceDiff } from '$lib/api/chat'
import { buildProjectConversationWorkspaceBrowserStateView } from './workspace-browser-state-view'
import { createWorkspaceBrowserTreeState } from './workspace-browser-tree-state.svelte'
import { createWorkspaceBrowserTabs } from './workspace-browser-tabs.svelte'
import { createWorkspaceBrowserSessionState } from './workspace-browser-session-state.svelte'
import { createWorkspaceFileEditorStore } from './project-conversation-workspace-file-editor-state.svelte'
import { createWorkspaceBrowserActions } from './project-conversation-workspace-browser-actions'
import { createWorkspaceBrowserLoaders } from './project-conversation-workspace-browser-loaders'
import { loadWorkspaceFile } from './project-conversation-workspace-data-loader'
import {
  buildWorkspaceFocusContext,
  workspaceSelectedChangedFiles,
} from './project-conversation-workspace-browser-file-ops'
import {
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
  const session = createWorkspaceBrowserSessionState()
  const tree = createWorkspaceBrowserTreeState()
  let loadRequestID = 0

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

  const {
    refreshWorkspaceDiff,
    refreshRepoGitContext,
    loadDirEntries,
    toggleDir,
    revealFileInTree,
    reloadFile,
    refreshWorkspace,
  } = createWorkspaceBrowserLoaders({
    getConversationId: input.getConversationId,
    getCurrentRequestID: () => loadRequestID,
    reserveRequestID: () => ++loadRequestID,
    onWorkspaceDiffUpdated: input.onWorkspaceDiffUpdated,
    getTreeRepoPath: () => tabs.treeRepoPath,
    setTreeRepoPath: tabs.setTreeRepoPath,
    getTreeNodes: () => tree.treeNodes,
    getExpandedDirs: () => tree.expandedDirs,
    getOpenTabs: () => tabs.openTabs,
    setMetadataLoading: session.setMetadataLoading,
    setMetadataError: session.setMetadataError,
    setMetadata: session.setMetadata,
    clearMetadata: session.clearMetadata,
    resetTreeState: tree.reset,
    closeAllTabs: tabs.closeAllTabs,
    setRepoRefsLoading: session.setRepoRefsLoading,
    setRepoRefsError: session.setRepoRefsError,
    setRepoRefs: session.setRepoRefs,
    setGitGraphLoading: session.setGitGraphLoading,
    setGitGraphError: session.setGitGraphError,
    setGitGraph: session.setGitGraph,
    getSelectedGitCommitID: () => session.selectedGitCommitID,
    setSelectedGitCommitID: session.setSelectedGitCommitID,
    setDirLoading: tree.setDirLoading,
    setTreeEntries: tree.setTreeEntries,
    setDirExpanded: tree.setDirExpanded,
    loadFile,
  })
  const editorStore = createWorkspaceFileEditorStore({
    getConversationId: input.getConversationId,
    getSelectedRepoPath: () => getActiveTab()?.repoPath ?? '',
    getSelectedFilePath: () => getActiveTab()?.filePath ?? '',
    getRepoRefCacheKey: (repoPath) =>
      session.metadata?.repos.find((repo) => repo.path === repoPath)?.currentRef.cacheKey ?? '',
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
    getAutosaveEnabled: () => session.autosaveEnabled,
  })

  function reset() {
    tree.reset()
    tabs.resetSelection()
    session.reset()
    editorStore.reset()
  }

  const workspaceActions = createWorkspaceBrowserActions({
    getConversationId: input.getConversationId,
    currentWorkspaceDiff,
    getMetadata: () => session.metadata,
    setMetadata: session.setMetadata,
    getRepoRefs: () => session.repoRefs,
    getTreeRepoPath: () => tabs.treeRepoPath,
    setTreeRepoPath: tabs.setTreeRepoPath,
    resetTree: tree.reset,
    getOpenTabs: () => tabs.openTabs,
    getActiveFilePath: () => tabs.activeFilePath(tabs.treeRepoPath),
    getActiveTabKey: () => tabs.activeTabKey,
    reserveRequestID: () => ++loadRequestID,
    setDetailMode: session.setDetailMode,
    revealFileInTree,
    refreshRepoGitContext,
    refreshRepoGitContextAfterCheckout: async (repoPath) => {
      await refreshRepoGitContext(repoPath)
      await refreshWorkspaceDiff()
    },
    refreshWorkspace,
    loadDirEntries,
    loadFile,
    openTab: tabs.openTab,
    activateTab: tabs.activateTab,
    closeTab: tabs.closeTab,
    remapTabPath: (repoPath, fromPath, toPath) => tabs.remapTabPath(repoPath, fromPath, toPath),
    editorStore,
  })
  const {
    openRepo,
    selectFile,
    createFile,
    searchPaths,
    checkoutBlockers,
    checkoutBranch,
    renameFile,
    deleteFile,
    selectNextChangedFile,
    selectPreviousChangedFile,
    reviewPatch,
  } = workspaceActions

  function activeFilePath() {
    return tabs.activeFilePath(tabs.treeRepoPath)
  }

  return buildProjectConversationWorkspaceBrowserStateView({
    getMetadata: () => session.metadata,
    getMetadataLoading: () => session.metadataLoading,
    getMetadataError: () => session.metadataError,
    getTreeNodes: () => tree.treeNodes,
    getExpandedDirs: () => tree.expandedDirs,
    getLoadingDirs: () => tree.loadingDirs,
    getOpenTabs: () => tabs.openTabs,
    getActiveTabKey: () => tabs.activeTabKey,
    getTabFileStates: () => tabs.tabFileStates,
    getRecentFiles: () => tabs.recentFiles,
    getAutosaveEnabled: () => session.autosaveEnabled,
    getDetailMode: () => session.detailMode,
    getRepoRefs: () => session.repoRefs,
    getRepoRefsLoading: () => session.repoRefsLoading,
    getRepoRefsError: () => session.repoRefsError,
    getGitGraph: () => session.gitGraph,
    getGitGraphLoading: () => session.gitGraphLoading,
    getGitGraphError: () => session.gitGraphError,
    getSelectedGitCommit: () =>
      session.gitGraph?.commits.find((commit) => commit.commitId === session.selectedGitCommitID) ??
      null,
    getHasDirtyTabs: () =>
      tabs.openTabs.some(
        (tab) => editorStore.getEditorState(tab.repoPath, tab.filePath)?.dirty === true,
      ),
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
    selectGitCommit: session.setSelectedGitCommitID,
    setDetailMode: session.setDetailMode,
    searchPaths,
    openTab: (repoPath: string, filePath: string) => {
      session.setDetailMode('file')
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
    setAutosaveEnabled: session.setAutosaveEnabled,
    selectNextChangedFile,
    selectPreviousChangedFile,
    reviewPatch,
    applySelectedPendingPatch: () =>
      editorStore.applyPendingPatch(tabs.treeRepoPath, activeFilePath()),
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
