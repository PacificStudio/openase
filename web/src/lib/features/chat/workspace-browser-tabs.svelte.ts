import {
  EMPTY_TAB_FILE_STATE,
  applyCloseTab,
  applyOpenTab,
  deleteTabFileStateMap,
  patchTabFileStateMap,
  pushRecentFile,
  workspaceTabKey,
  type WorkspaceRecentFile,
  type WorkspaceTab,
  type WorkspaceTabFileState,
} from './project-conversation-workspace-browser-state-helpers'
import { remapWorkspaceTabPath, workspaceActiveFilePath } from './project-conversation-workspace-browser-file-ops'

export function createWorkspaceBrowserTabs(input: {
  loadFile: (repoPath: string, filePath: string, options?: { silent?: boolean }) => Promise<void> | void
  renameEditorFileState: (repoPath: string, fromPath: string, toPath: string) => void
}) {
  let openTabs = $state<WorkspaceTab[]>([])
  let activeTabKey = $state('')
  let tabFileStates = $state<Map<string, WorkspaceTabFileState>>(new Map())
  let treeRepoPath = $state('')
  let recentFiles = $state<WorkspaceRecentFile[]>([])

  function patchTabFileState(key: string, patch: Partial<WorkspaceTabFileState>) {
    tabFileStates = patchTabFileStateMap(tabFileStates, key, patch)
  }

  function touchRecentFile(repoPath: string, filePath: string) {
    recentFiles = pushRecentFile(recentFiles, { repoPath, filePath })
  }

  function getActiveTab(): WorkspaceTab | null {
    if (!activeTabKey) return null
    return openTabs.find((tab) => workspaceTabKey(tab) === activeTabKey) ?? null
  }

  function getActiveTabFileState(): WorkspaceTabFileState {
    return tabFileStates.get(activeTabKey) ?? EMPTY_TAB_FILE_STATE
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
    const next = applyOpenTab(openTabs, repoPath, filePath)
    openTabs = next.openTabs
    activeTabKey = next.activeTabKey
    treeRepoPath = next.treeRepoPath
    touchRecentFile(repoPath, filePath)
    const cached = tabFileStates.get(next.activeTabKey)
    if (!cached || !cached.preview) {
      void input.loadFile(repoPath, filePath, { silent: cached?.preview != null })
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

  function remapTabPath(repoPath: string, fromPath: string, toPath: string) {
    remapWorkspaceTabPath({
      repoPath,
      fromPath,
      toPath,
      openTabs,
      tabFileStates,
      activeTabKey,
      recentFiles,
      setOpenTabs: (tabs: WorkspaceTab[]) => {
        openTabs = tabs
      },
      setTabFileStates: (states: Map<string, WorkspaceTabFileState>) => {
        tabFileStates = states
      },
      setActiveTabKey: (key: string) => {
        activeTabKey = key
      },
      setRecentFiles: (files: WorkspaceRecentFile[]) => {
        recentFiles = files
      },
      renameEditorFileState: input.renameEditorFileState,
    })
  }

  function activeFilePath(repoPath = treeRepoPath): string {
    return workspaceActiveFilePath({
      openTabs: openTabs.filter((tab) => tab.repoPath === repoPath),
      activeTabKey,
    })
  }

  function resetSelection() {
    treeRepoPath = ''
    closeAllTabs()
  }

  return {
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
    get treeRepoPath() {
      return treeRepoPath
    },
    setTreeRepoPath: (repoPath: string) => {
      treeRepoPath = repoPath
    },
    patchTabFileState,
    getActiveTab,
    getActiveTabFileState,
    touchRecentFile,
    openTab,
    closeTab,
    closeAllTabs,
    activateTab,
    remapTabPath,
    activeFilePath,
    resetSelection,
  }
}
