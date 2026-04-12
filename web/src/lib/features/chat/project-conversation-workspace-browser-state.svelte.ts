/* eslint-disable max-lines */
import {
  createProjectConversationWorkspaceFile,
  deleteProjectConversationWorkspaceFile,
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceDiff,
  listProjectConversationWorkspaceTree,
  renameProjectConversationWorkspaceFile,
  searchProjectConversationWorkspacePaths,
  type ChatDiffPayload,
  type ProjectConversationWorkspaceDiff,
  type ProjectConversationWorkspaceMetadata,
  type ProjectConversationWorkspaceSearchResult,
  type ProjectConversationWorkspaceTreeEntry,
} from '$lib/api/chat'
import { createWorkspaceFileEditorStore } from './project-conversation-workspace-file-editor-state.svelte'
import { loadWorkspaceFile } from './project-conversation-workspace-data-loader'
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
  type WorkspaceFocusContext,
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
  if (typeof window === 'undefined') {
    return false
  }
  return window.localStorage.getItem(WORKSPACE_AUTOSAVE_STORAGE_KEY) === 'true'
}

function storeWorkspaceAutosavePreference(enabled: boolean) {
  if (typeof window === 'undefined') {
    return
  }
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

  function setMetadata(nextMetadata: ProjectConversationWorkspaceMetadata) {
    if (!areWorkspaceMetadataEqual(metadata, nextMetadata)) {
      metadata = nextMetadata
    }
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
    if (loadingDirs.has(dirPath) === loading) {
      return
    }

    const nextLoadingDirs = new Set(loadingDirs)
    if (loading) {
      nextLoadingDirs.add(dirPath)
    } else {
      nextLoadingDirs.delete(dirPath)
    }
    loadingDirs = nextLoadingDirs
  }

  function setDirExpanded(dirPath: string, expanded: boolean) {
    if (expandedDirs.has(dirPath) === expanded) {
      return
    }

    const nextExpandedDirs = new Set(expandedDirs)
    if (expanded) {
      nextExpandedDirs.add(dirPath)
    } else {
      nextExpandedDirs.delete(dirPath)
    }
    expandedDirs = nextExpandedDirs
  }

  function getActiveTab(): WorkspaceTab | null {
    if (!activeTabKey) return null
    return openTabs.find((tab) => workspaceTabKey(tab) === activeTabKey) ?? null
  }

  function getActiveTabFileState(): WorkspaceTabFileState {
    return tabFileStates.get(activeTabKey) ?? EMPTY_TAB_FILE_STATE
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
    if (tab) {
      treeRepoPath = tab.repoPath
      touchRecentFile(tab.repoPath, tab.filePath)
    }
  }

  function openTab(repoPath: string, filePath: string) {
    if (!repoPath || !filePath) return
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
    if (!conversationId || !input.onWorkspaceDiffUpdated) {
      return
    }
    const payload = await getProjectConversationWorkspaceDiff(conversationId)
    input.onWorkspaceDiffUpdated(payload.workspaceDiff)
  }

  const editorStore = createWorkspaceFileEditorStore({
    getConversationId: input.getConversationId,
    getSelectedRepoPath: () => getActiveTab()?.repoPath ?? '',
    getSelectedFilePath: () => getActiveTab()?.filePath ?? '',
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
    editorStore.reset()
  }

  async function refreshWorkspace(preserveSelection: boolean) {
    const conversationId = input.getConversationId()
    const requestID = ++loadRequestID
    metadataLoading = true
    metadataError = ''

    try {
      const payload = await getProjectConversationWorkspace(conversationId)
      if (requestID !== loadRequestID || conversationId !== input.getConversationId()) {
        return
      }

      setMetadata(payload.workspace)
      if (!payload.workspace.available || payload.workspace.repos.length === 0) {
        treeRepoPath = ''
        treeNodes = new Map()
        expandedDirs = new Set()
        loadingDirs = new Set()
        closeAllTabs()
        return
      }

      const nextRepoPath =
        preserveSelection &&
        payload.workspace.repos.some((repo) => repo.path === treeRepoPath) &&
        treeRepoPath
          ? treeRepoPath
          : (payload.workspace.repos[0]?.path ?? '')

      const repoChanged = nextRepoPath !== treeRepoPath
      const prevExpanded = repoChanged ? [] : [...expandedDirs]
      treeRepoPath = nextRepoPath

      if (repoChanged) {
        treeNodes = new Map()
        expandedDirs = new Set()
        loadingDirs = new Set()
      }

      await loadDirEntries('', requestID, { silent: treeNodes.has('') })
      if (requestID !== loadRequestID) {
        return
      }

      if (prevExpanded.length > 0) {
        await Promise.all(
          prevExpanded.map((dirPath) =>
            loadDirEntries(dirPath, requestID, { silent: treeNodes.has(dirPath) }),
          ),
        )
      }

      if (preserveSelection && openTabs.length > 0) {
        await Promise.all(
          openTabs.map((tab) => loadFile(tab.repoPath, tab.filePath, { silent: true })),
        )
      }
    } catch (error) {
      if (requestID !== loadRequestID || conversationId !== input.getConversationId()) {
        return
      }
      metadata = null
      treeNodes = new Map()
      expandedDirs = new Set()
      loadingDirs = new Set()
      closeAllTabs()
      metadataError =
        error instanceof Error ? error.message : 'Failed to load the Project AI workspace.'
    } finally {
      if (requestID === loadRequestID && conversationId === input.getConversationId()) {
        metadataLoading = false
      }
    }
  }

  async function loadDirEntries(
    dirPath: string,
    externalRequestID?: number,
    options: { silent?: boolean } = {},
  ) {
    const conversationId = input.getConversationId()
    const repoPath = treeRepoPath
    if (!repoPath || !conversationId) {
      return
    }

    const requestID = externalRequestID ?? loadRequestID
    const silent = options.silent ?? false
    if (!silent) {
      setDirLoading(dirPath, true)
    }

    try {
      const payload = await listProjectConversationWorkspaceTree(conversationId, {
        repoPath,
        path: dirPath,
      })
      if (requestID !== loadRequestID || repoPath !== treeRepoPath) {
        return
      }
      setTreeEntries(dirPath, payload.workspaceTree.entries)
    } catch {
      setTreeEntries(dirPath, [])
    } finally {
      if (!silent) {
        setDirLoading(dirPath, false)
      }
    }
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
    if (!path) {
      return
    }

    const requestID = options.requestID ?? loadRequestID
    const ancestorDirs = path
      .split('/')
      .slice(0, -1)
      .reduce<string[]>((dirs, segment) => {
        const nextPath = dirs.length > 0 ? `${dirs[dirs.length - 1]}/${segment}` : segment
        dirs.push(nextPath)
        return dirs
      }, [])

    if (ancestorDirs.length === 0) {
      return
    }
    if (!treeNodes.has('')) {
      await loadDirEntries('', requestID, options)
    }

    for (const dirPath of ancestorDirs) {
      if (requestID !== loadRequestID) {
        return
      }
      setDirExpanded(dirPath, true)
      if (!treeNodes.has(dirPath)) {
        await loadDirEntries(dirPath, requestID, options)
      }
    }
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

  function openRepo(repoPath: string) {
    if (!repoPath || repoPath === treeRepoPath) {
      return
    }
    treeRepoPath = repoPath
    treeNodes = new Map()
    expandedDirs = new Set()
    loadingDirs = new Set()
    void loadDirEntries('')
  }

  function selectFile(path: string) {
    if (!path) {
      return
    }
    const repoPath = treeRepoPath
    if (!repoPath) {
      return
    }
    const requestID = ++loadRequestID
    void revealFileInTree(path, { requestID, silent: true })
    openTab(repoPath, path)
  }

  async function createFile(path: string) {
    const conversationId = input.getConversationId()
    const repoPath = treeRepoPath
    if (!conversationId || !repoPath || !path) {
      return false
    }
    await createProjectConversationWorkspaceFile(conversationId, { repoPath, path })
    await refreshWorkspace(true)
    selectFile(path)
    return true
  }

  async function searchPaths(
    query: string,
    limit = 20,
  ): Promise<ProjectConversationWorkspaceSearchResult[]> {
    const conversationId = input.getConversationId()
    const repoPath = treeRepoPath
    const trimmedQuery = query.trim()
    if (!conversationId || !repoPath || !trimmedQuery) {
      return []
    }
    const payload = await searchProjectConversationWorkspacePaths(conversationId, {
      repoPath,
      query: trimmedQuery,
      limit,
    })
    return payload.workspaceSearch.results
  }

  function remapTabPath(repoPath: string, fromPath: string, toPath: string) {
    const fromKey = workspaceTabKey({ repoPath, filePath: fromPath })
    const toKey = workspaceTabKey({ repoPath, filePath: toPath })
    openTabs = openTabs.map((tab) =>
      tab.repoPath === repoPath && tab.filePath === fromPath ? { repoPath, filePath: toPath } : tab,
    )
    const nextTabStates = new Map(tabFileStates)
    const existing = nextTabStates.get(fromKey)
    if (existing) {
      nextTabStates.delete(fromKey)
      nextTabStates.set(toKey, {
        ...existing,
        preview: existing.preview ? { ...existing.preview, path: toPath } : existing.preview,
        patch: existing.patch ? { ...existing.patch, path: toPath } : existing.patch,
      })
    }
    tabFileStates = nextTabStates
    if (activeTabKey === fromKey) {
      activeTabKey = toKey
    }
    recentFiles = recentFiles.map((item) =>
      item.repoPath === repoPath && item.filePath === fromPath
        ? { repoPath, filePath: toPath }
        : item,
    )
    editorStore.renameFileState(repoPath, fromPath, toPath)
  }

  async function renameFile(fromPath: string, toPath: string) {
    const conversationId = input.getConversationId()
    const repoPath = treeRepoPath
    if (!conversationId || !repoPath || !fromPath || !toPath) {
      return false
    }
    await renameProjectConversationWorkspaceFile(conversationId, { repoPath, fromPath, toPath })
    remapTabPath(repoPath, fromPath, toPath)
    await refreshWorkspace(true)
    activateTab(repoPath, toPath)
    if (activeTabKey === workspaceTabKey({ repoPath, filePath: toPath })) {
      await loadFile(repoPath, toPath, { silent: true })
    }
    return true
  }

  async function deleteFile(path: string) {
    const conversationId = input.getConversationId()
    const repoPath = treeRepoPath
    if (!conversationId || !repoPath || !path) {
      return false
    }
    await deleteProjectConversationWorkspaceFile(conversationId, { repoPath, path })
    editorStore.discardDraft(repoPath, path)
    closeTab(repoPath, path)
    await refreshWorkspace(true)
    return true
  }

  function setAutosaveEnabled(enabled: boolean) {
    autosaveEnabled = enabled
    storeWorkspaceAutosavePreference(enabled)
  }

  function selectedChangedFiles() {
    const diff = input.getWorkspaceDiff?.()
    return diff?.repos.find((repo) => repo.path === treeRepoPath)?.files ?? []
  }

  function selectRelativeChangedFile(offset: 1 | -1) {
    const files = selectedChangedFiles()
    if (files.length === 0) {
      return
    }
    const currentPath = activeFilePath()
    const currentIndex = files.findIndex((file) => file.path === currentPath)
    const baseIndex = currentIndex === -1 ? (offset === 1 ? -1 : 0) : currentIndex
    const nextIndex = (baseIndex + offset + files.length) % files.length
    selectFile(files[nextIndex]?.path ?? files[0].path)
  }

  function selectNextChangedFile() {
    selectRelativeChangedFile(1)
  }

  function selectPreviousChangedFile() {
    selectRelativeChangedFile(-1)
  }

  function activeFilePath(): string {
    const tab = getActiveTab()
    if (!tab || tab.repoPath !== treeRepoPath) return ''
    return tab.filePath
  }

  function getSelectedFocusContext(): WorkspaceFocusContext | null {
    const editor = editorStore.selectedEditorState
    const activeTab = getActiveTab()
    if (!editor || !activeTab) {
      return null
    }
    return {
      selectedArea: editor.selection ? 'selection' : 'edit',
      selection: editor.selection,
      workingSet: editorStore.buildWorkingSet(recentFiles),
    }
  }

  async function reviewPatch(diff: ChatDiffPayload, options: { autoApply?: boolean } = {}) {
    const repoPath = treeRepoPath
    if (!repoPath || !diff.file) {
      return false
    }
    openTab(repoPath, diff.file)
    await loadFile(repoPath, diff.file, { silent: true })
    const ok = editorStore.reviewPatch(repoPath, diff.file, diff)
    if (ok && options.autoApply) {
      return editorStore.applyPendingPatch(repoPath, diff.file)
    }
    return ok
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
      return selectedChangedFiles()
    },
    getSelectedFocusContext,
    getEditorState: editorStore.getEditorState,
    reset,
    refreshWorkspace,
    toggleDir,
    openRepo,
    selectFile,
    searchPaths,
    openTab,
    closeTab,
    closeAllTabs,
    activateTab,
    createFile,
    renameFile,
    deleteFile,
    setAutosaveEnabled,
    selectNextChangedFile,
    selectPreviousChangedFile,
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
