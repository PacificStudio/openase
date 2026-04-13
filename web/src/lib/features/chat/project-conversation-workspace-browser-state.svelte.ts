/* eslint-disable max-lines */
import {
  getProjectConversationWorkspaceDiff,
  type ChatDiffPayload,
  type ProjectConversationWorkspaceBranchScope,
  type ProjectConversationWorkspaceDiff,
  type ProjectConversationWorkspaceGitGraph,
  type ProjectConversationWorkspaceGitGraphCommit,
  type ProjectConversationWorkspaceMetadata,
  type ProjectConversationWorkspaceRepoRefs,
  type ProjectConversationWorkspaceSearchResult,
  type ProjectConversationWorkspaceTreeEntry,
} from '$lib/api/chat'
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
  remapWorkspaceTabPath,
  renameWorkspaceFileEntry,
  reviewWorkspacePatch,
  searchWorkspacePaths,
  workspaceActiveFilePath,
  workspaceSelectedChangedFiles,
} from './project-conversation-workspace-browser-file-ops'
import {
  EMPTY_TAB_FILE_STATE,
  applyCloseTab,
  applyOpenTab,
  areTreeEntriesEqual,
  areWorkspaceMetadataEqual,
  deleteTabFileStateMap,
  patchTabFileStateMap,
  pushRecentFile,
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

const WORKSPACE_AUTOSAVE_STORAGE_KEY = 'openase.project-conversation.workspace-autosave'

function readWorkspaceAutosavePreference() {
  if (typeof window === 'undefined') return false
  return window.localStorage.getItem(WORKSPACE_AUTOSAVE_STORAGE_KEY) === 'true'
}

function storeWorkspaceAutosavePreference(enabled: boolean) {
  if (typeof window === 'undefined') return
  window.localStorage.setItem(WORKSPACE_AUTOSAVE_STORAGE_KEY, enabled ? 'true' : 'false')
}

export function createProjectConversationWorkspaceBrowserState(input: {
  getConversationId: () => string
  getWorkspaceDiff?: () => ProjectConversationWorkspaceDiff | null
  onWorkspaceDiffUpdated?: (workspaceDiff: ProjectConversationWorkspaceDiff | null) => void
}) {
  let metadata = $state<ProjectConversationWorkspaceMetadata | null>(null)
  let metadataLoading = $state(false)
  let metadataError = $state('')

  let treeNodes = $state<Map<string, ProjectConversationWorkspaceTreeEntry[]>>(new Map())
  let expandedDirs = $state<Set<string>>(new Set())
  let loadingDirs = $state<Set<string>>(new Set())

  let openTabs = $state<WorkspaceTab[]>([])
  let activeTabKey = $state('')
  let tabFileStates = $state<Map<string, WorkspaceTabFileState>>(new Map())
  let treeRepoPath = $state('')
  let recentFiles = $state<WorkspaceRecentFile[]>([])
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

  function setMetadata(nextMetadata: ProjectConversationWorkspaceMetadata) {
    if (!areWorkspaceMetadataEqual(metadata, nextMetadata)) metadata = nextMetadata
  }

  function setTreeEntries(dirPath: string, entries: ProjectConversationWorkspaceTreeEntry[]) {
    if (areTreeEntriesEqual(treeNodes.get(dirPath), entries)) {
      return
    }

    const nextTreeNodes = new Map(treeNodes)
    nextTreeNodes.set(dirPath, entries)
    treeNodes = nextTreeNodes
  }

  function setDirLoading(dirPath: string, loading: boolean) {
    if (loadingDirs.has(dirPath) === loading) return
    const nextLoadingDirs = new Set(loadingDirs)
    if (loading) nextLoadingDirs.add(dirPath)
    else nextLoadingDirs.delete(dirPath)
    loadingDirs = nextLoadingDirs
  }

  function setDirExpanded(dirPath: string, expanded: boolean) {
    if (expandedDirs.has(dirPath) === expanded) return
    const nextExpandedDirs = new Set(expandedDirs)
    if (expanded) nextExpandedDirs.add(dirPath)
    else nextExpandedDirs.delete(dirPath)
    expandedDirs = nextExpandedDirs
  }

  function getActiveTab(): WorkspaceTab | null {
    if (!activeTabKey) return null
    return openTabs.find((tab) => workspaceTabKey(tab) === activeTabKey) ?? null
  }

  function getActiveTabFileState(): WorkspaceTabFileState {
    return tabFileStates.get(activeTabKey) ?? EMPTY_TAB_FILE_STATE
  }

  function currentWorkspaceDiff() {
    return input.getWorkspaceDiff?.() ?? null
  }

  function patchTabFileState(key: string, patch: Partial<WorkspaceTabFileState>) {
    tabFileStates = patchTabFileStateMap(tabFileStates, key, patch)
  }

  function touchRecentFile(repoPath: string, filePath: string) {
    recentFiles = pushRecentFile(recentFiles, { repoPath, filePath })
  }

  function activateTabKey(key: string) {
    if (activeTabKey === key) return
    activeTabKey = key
    const tab = getActiveTab()
    if (!tab) return
    treeRepoPath = tab.repoPath
    touchRecentFile(tab.repoPath, tab.filePath)
  }

  function openTab(repoPath: string, filePath: string) {
    if (!repoPath || !filePath) return
    detailMode = 'file'
    const next = applyOpenTab(openTabs, repoPath, filePath)
    openTabs = next.openTabs
    activeTabKey = next.activeTabKey
    treeRepoPath = next.treeRepoPath
    touchRecentFile(repoPath, filePath)
    const cached = tabFileStates.get(next.activeTabKey)
    if (!cached || !cached.preview) {
      void loadFile(repoPath, filePath, { silent: cached?.preview != null })
    }
  }

  function closeTab(repoPath: string, filePath: string) {
    const next = applyCloseTab(openTabs, activeTabKey, repoPath, filePath)
    if (!next) return
    const key = workspaceTabKey({ repoPath, filePath })
    openTabs = next.openTabs
    tabFileStates = deleteTabFileStateMap(tabFileStates, key)
    activeTabKey = next.activeTabKey
    if (next.nextTreeRepo) treeRepoPath = next.nextTreeRepo
  }

  function closeAllTabs() {
    openTabs = []
    activeTabKey = ''
    tabFileStates = new Map()
    recentFiles = []
  }

  function activateTab(repoPath: string, filePath: string) {
    const key = workspaceTabKey({ repoPath, filePath })
    if (!openTabs.some((tab) => workspaceTabKey(tab) === key)) return
    activateTabKey(key)
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
      return tabFileStates.get(key)?.preview ?? null
    },
    setPreview: (repoPath, filePath, preview) => {
      const key = workspaceTabKey({ repoPath, filePath })
      patchTabFileState(key, { preview })
    },
    reloadSelectedFile: async (repoPath, filePath) => {
      if (repoPath && filePath) {
        await loadFile(repoPath, filePath, { silent: true })
      }
    },
    refreshWorkspaceDiff,
    getAutosaveEnabled: () => autosaveEnabled,
  })

  function reset() {
    metadata = null
    metadataLoading = false
    metadataError = ''
    treeNodes = new Map()
    expandedDirs = new Set()
    loadingDirs = new Set()
    treeRepoPath = ''
    closeAllTabs()
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
      treeRepoPath,
      treeNodes,
      expandedDirs,
      openTabs,
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
        treeRepoPath = repoPath
      },
      resetTreeState: () => {
        treeNodes = new Map()
        expandedDirs = new Set()
        loadingDirs = new Set()
      },
      closeAllTabs,
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

  async function refreshRepoGitContext(repoPath = treeRepoPath) {
    const conversationId = input.getConversationId()
    await refreshWorkspaceRepoGitContext({
      conversationId,
      repoPath,
      treeRepoPath,
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
      repoPath: treeRepoPath,
      dirPath,
      requestID: externalRequestID ?? loadRequestID,
      currentRequestID: loadRequestID,
      silent: options.silent ?? false,
      treeRepoPath,
      setDirLoading,
      setTreeEntries,
    })
  }

  async function toggleDir(dirPath: string) {
    if (expandedDirs.has(dirPath)) {
      setDirExpanded(dirPath, false)
      return
    }

    setDirExpanded(dirPath, true)
    if (!treeNodes.has(dirPath)) {
      await loadDirEntries(dirPath)
    }
  }

  async function revealFileInTree(
    path: string,
    options: { requestID?: number; silent?: boolean } = {},
  ) {
    await revealWorkspaceFileInTree({
      path,
      requestID: options.requestID ?? loadRequestID,
      currentRequestID: () => loadRequestID,
      hasTreeEntries: (dirPath) => treeNodes.has(dirPath),
      setDirExpanded,
      loadDirEntries,
      options,
    })
  }

  function loadFile(repoPath: string, filePath: string, options: { silent?: boolean } = {}) {
    return loadWorkspaceFile(
      {
        getConversationId: input.getConversationId,
        hasOpenTab: (key) => openTabs.some((tab) => workspaceTabKey(tab) === key),
        getCurrentLoading: (key) => tabFileStates.get(key)?.loading ?? false,
        patchTabFileState,
        syncEditorFromPreview: editorStore.syncFromPreview,
      },
      repoPath,
      filePath,
      options,
    )
  }

  async function reloadFile(repoPath: string, filePath: string) {
    if (!repoPath || !filePath) return
    await loadFile(repoPath, filePath, { silent: true })
  }

  function openRepo(repoPath: string) {
    if (!repoPath || repoPath === treeRepoPath) return
    treeRepoPath = repoPath
    treeNodes = new Map()
    expandedDirs = new Set()
    loadingDirs = new Set()
    void loadDirEntries('')
    void refreshRepoGitContext(repoPath)
  }

  function selectFile(path: string) {
    const repoPath = treeRepoPath
    if (!path || !repoPath) return
    detailMode = 'file'
    const requestID = ++loadRequestID
    void revealFileInTree(path, { requestID, silent: true })
    openTab(repoPath, path)
  }

  async function createFile(path: string) {
    return createWorkspaceFileEntry({
      conversationId: input.getConversationId(),
      repoPath: treeRepoPath,
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
      repoPath: treeRepoPath,
      query,
      limit,
    })
  }

  function checkoutBlockers(repoPath: string): string[] {
    return computeWorkspaceCheckoutBlockers({
      repoPath,
      metadata,
      workspaceDiff: currentWorkspaceDiff(),
      openTabs,
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
      openTabs,
      getEditorState: editorStore.getEditorState,
      setMetadata,
      refreshWorkspace,
      refreshRepoGitContext: async (repoPath) => {
        await refreshRepoGitContext(repoPath)
        await refreshWorkspaceDiff()
      },
    })
  }

  function remapTabPath(repoPath: string, fromPath: string, toPath: string) {
    remapWorkspaceTabPath({
      repoPath,
      fromPath,
      toPath,
      openTabs,
      tabFileStates,
      activeTabKey,
      recentFiles,
      setOpenTabs: (tabs) => {
        openTabs = tabs
      },
      setTabFileStates: (states) => {
        tabFileStates = states
      },
      setActiveTabKey: (key) => {
        activeTabKey = key
      },
      setRecentFiles: (files) => {
        recentFiles = files
      },
      renameEditorFileState: editorStore.renameFileState,
    })
  }

  async function renameFile(fromPath: string, toPath: string) {
    return renameWorkspaceFileEntry({
      conversationId: input.getConversationId(),
      repoPath: treeRepoPath,
      fromPath,
      toPath,
      remapTabPath: () => remapTabPath(treeRepoPath, fromPath, toPath),
      refreshWorkspace,
      activateTab,
      getActiveTabKey: () => activeTabKey,
      loadFile,
    })
  }

  async function deleteFile(path: string) {
    return deleteWorkspaceFileEntry({
      conversationId: input.getConversationId(),
      repoPath: treeRepoPath,
      path,
      discardDraft: editorStore.discardDraft,
      closeTab,
      refreshWorkspace,
    })
  }

  function setAutosaveEnabled(enabled: boolean) {
    autosaveEnabled = enabled
    storeWorkspaceAutosavePreference(enabled)
  }

  function selectRelativeChangedFile(offset: 1 | -1) {
    const nextPath = relativeChangedFilePath({
      selectedChangedFiles: workspaceSelectedChangedFiles({
        repoPath: treeRepoPath,
        activeFilePath: activeFilePath(),
        workspaceDiff: currentWorkspaceDiff(),
      }),
      activeFilePath: activeFilePath(),
      offset,
    })
    if (nextPath) selectFile(nextPath)
  }

  function activeFilePath(): string {
    return workspaceActiveFilePath({
      openTabs: openTabs.filter((tab) => tab.repoPath === treeRepoPath),
      activeTabKey,
    })
  }

  async function reviewPatch(diff: ChatDiffPayload, options: { autoApply?: boolean } = {}) {
    return reviewWorkspacePatch({
      repoPath: treeRepoPath,
      diff,
      autoApply: options.autoApply ?? false,
      openTab,
      loadFile,
      reviewPatch: editorStore.reviewPatch,
      applyPendingPatch: editorStore.applyPendingPatch,
    })
  }

  return {
    get metadata() {
      return metadata
    },
    get metadataLoading() {
      return metadataLoading
    },
    get metadataError() {
      return metadataError
    },
    get treeNodes() {
      return treeNodes
    },
    get expandedDirs() {
      return expandedDirs
    },
    get loadingDirs() {
      return loadingDirs
    },
    get openTabs() {
      return openTabs
    },
    get activeTabKey() {
      return activeTabKey
    },
    get tabFileStates() {
      return tabFileStates
    },
    get recentFiles() {
      return recentFiles
    },
    get autosaveEnabled() {
      return autosaveEnabled
    },
    get detailMode() {
      return detailMode
    },
    get repoRefs() {
      return repoRefs
    },
    get repoRefsLoading() {
      return repoRefsLoading
    },
    get repoRefsError() {
      return repoRefsError
    },
    get gitGraph() {
      return gitGraph
    },
    get gitGraphLoading() {
      return gitGraphLoading
    },
    get gitGraphError() {
      return gitGraphError
    },
    get selectedGitCommit(): ProjectConversationWorkspaceGitGraphCommit | null {
      return gitGraph?.commits.find((commit) => commit.commitId === selectedGitCommitID) ?? null
    },
    get hasDirtyTabs() {
      return openTabs.some(
        (tab) => editorStore.getEditorState(tab.repoPath, tab.filePath)?.dirty === true,
      )
    },
    get preview() {
      return getActiveTabFileState().preview
    },
    get patch() {
      return getActiveTabFileState().patch
    },
    get fileLoading() {
      return getActiveTabFileState().loading
    },
    get fileError() {
      return getActiveTabFileState().error
    },
    get selectedRepoPath() {
      return treeRepoPath
    },
    get selectedFilePath() {
      return activeFilePath()
    },
    get selectedEditorState() {
      return editorStore.selectedEditorState
    },
    get selectedDraftLineDiff() {
      return editorStore.selectedDraftLineDiff
    },
    get selectedChangedFiles() {
      return workspaceSelectedChangedFiles({
        repoPath: treeRepoPath,
        activeFilePath: activeFilePath(),
        workspaceDiff: currentWorkspaceDiff(),
      })
    },
    getSelectedFocusContext: () =>
      buildWorkspaceFocusContext({
        selectedEditorState: editorStore.selectedEditorState,
        hasActiveTab: getActiveTab() != null,
        recentFiles,
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
    openTab,
    closeTab,
    closeAllTabs,
    activateTab,
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
    applySelectedPendingPatch: () => editorStore.applyPendingPatch(treeRepoPath, activeFilePath()),
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
  }
}

export type ProjectConversationWorkspaceBrowserState = ReturnType<
  typeof createProjectConversationWorkspaceBrowserState
>
