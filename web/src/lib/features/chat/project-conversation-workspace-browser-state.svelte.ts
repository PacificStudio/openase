import {
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceDiff,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
  type ProjectConversationWorkspaceDiff,
  type ProjectConversationWorkspaceFilePatch,
  type ProjectConversationWorkspaceFilePreview,
  type ProjectConversationWorkspaceMetadata,
  type ProjectConversationWorkspaceTreeEntry,
} from '$lib/api/chat'
import { createWorkspaceFileEditorStore } from './project-conversation-workspace-file-editor-state.svelte'
import {
  areFilePatchEqual,
  areFilePreviewEqual,
  areTreeEntriesEqual,
  areWorkspaceMetadataEqual,
} from './project-conversation-workspace-browser-state-helpers'
export type { WorkspaceFileEditorState } from './project-conversation-workspace-browser-state-helpers'

export function createProjectConversationWorkspaceBrowserState(input: {
  getConversationId: () => string
  onWorkspaceDiffUpdated?: (workspaceDiff: ProjectConversationWorkspaceDiff | null) => void
}) {
  let metadata = $state<ProjectConversationWorkspaceMetadata | null>(null)
  let metadataLoading = $state(false)
  let metadataError = $state('')

  let treeNodes = $state<Map<string, ProjectConversationWorkspaceTreeEntry[]>>(new Map())
  let expandedDirs = $state<Set<string>>(new Set())
  let loadingDirs = $state<Set<string>>(new Set())

  let preview = $state<ProjectConversationWorkspaceFilePreview | null>(null)
  let patch = $state<ProjectConversationWorkspaceFilePatch | null>(null)
  let fileLoading = $state(false)
  let fileError = $state('')
  let selectedRepoPath = $state('')
  let selectedFilePath = $state('')
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

  function setPreview(nextPreview: ProjectConversationWorkspaceFilePreview | null) {
    if (nextPreview == null) {
      if (preview !== null) {
        preview = null
      }
      return
    }
    if (!areFilePreviewEqual(preview, nextPreview)) {
      preview = nextPreview
    }
  }

  function setPatch(nextPatch: ProjectConversationWorkspaceFilePatch | null) {
    if (nextPatch == null) {
      if (patch !== null) {
        patch = null
      }
      return
    }
    if (!areFilePatchEqual(patch, nextPatch)) {
      patch = nextPatch
    }
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
    getSelectedRepoPath: () => selectedRepoPath,
    getSelectedFilePath: () => selectedFilePath,
    getPreview: () => preview,
    setPreview,
    reloadSelectedFile: async () => {
      if (selectedFilePath) {
        await loadFile(selectedFilePath, { silent: true })
      }
    },
    refreshWorkspaceDiff,
  })

  function clearSelectionState() {
    selectedFilePath = ''
    setPreview(null)
    setPatch(null)
    fileError = ''
  }

  function resetTreeState() {
    treeNodes = new Map()
    expandedDirs = new Set()
    loadingDirs = new Set()
  }

  function reset() {
    metadata = null
    metadataLoading = false
    metadataError = ''
    resetTreeState()
    clearSelectionState()
    fileLoading = false
    selectedRepoPath = ''
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
        selectedRepoPath = ''
        resetTreeState()
        clearSelectionState()
        return
      }

      const nextRepoPath =
        preserveSelection &&
        payload.workspace.repos.some((repo) => repo.path === selectedRepoPath) &&
        selectedRepoPath
          ? selectedRepoPath
          : (payload.workspace.repos[0]?.path ?? '')

      const repoChanged = nextRepoPath !== selectedRepoPath
      const prevExpanded = repoChanged ? [] : [...expandedDirs]
      selectedRepoPath = nextRepoPath

      if (repoChanged) {
        resetTreeState()
        clearSelectionState()
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

      if (preserveSelection && selectedFilePath && !repoChanged) {
        await loadFile(selectedFilePath, {
          requestID,
          silent: preview != null || patch != null,
        })
      }
    } catch (error) {
      if (requestID !== loadRequestID || conversationId !== input.getConversationId()) {
        return
      }
      metadata = null
      resetTreeState()
      clearSelectionState()
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
    const repoPath = selectedRepoPath
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
      if (requestID !== loadRequestID || repoPath !== selectedRepoPath) {
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
      if (requestID !== loadRequestID || path !== selectedFilePath) {
        return
      }
      setDirExpanded(dirPath, true)
      if (!treeNodes.has(dirPath)) {
        await loadDirEntries(dirPath, requestID, options)
      }
    }
  }

  async function loadFile(path: string, options: { requestID?: number; silent?: boolean } = {}) {
    const conversationId = input.getConversationId()
    const repoPath = selectedRepoPath
    if (!repoPath || !conversationId) {
      clearSelectionState()
      return
    }

    const requestID = options.requestID ?? ++loadRequestID
    const silent = options.silent ?? false
    if (!silent) {
      fileLoading = true
      fileError = ''
    }
    selectedFilePath = path

    try {
      const [previewPayload, patchPayload] = await Promise.all([
        getProjectConversationWorkspaceFilePreview(conversationId, { repoPath, path }),
        getProjectConversationWorkspaceFilePatch(conversationId, { repoPath, path }),
      ])
      if (requestID !== loadRequestID || path !== selectedFilePath) {
        return
      }
      setPreview(previewPayload.filePreview)
      setPatch(patchPayload.filePatch)
      editorStore.syncFromPreview(repoPath, path, previewPayload.filePreview)
    } catch (error) {
      if (requestID !== loadRequestID || path !== selectedFilePath) {
        return
      }
      setPreview(null)
      setPatch(null)
      fileError =
        error instanceof Error ? error.message : 'Failed to load the workspace file details.'
    } finally {
      if (!silent && requestID === loadRequestID && path === selectedFilePath) {
        fileLoading = false
      }
    }
  }

  function openRepo(repoPath: string) {
    if (!repoPath || repoPath === selectedRepoPath) {
      return
    }
    selectedRepoPath = repoPath
    resetTreeState()
    clearSelectionState()
    void loadDirEntries('')
  }

  function selectFile(path: string) {
    if (!path) {
      return
    }
    const requestID = ++loadRequestID
    selectedFilePath = path
    void revealFileInTree(path, { requestID, silent: true })
    void loadFile(path, { requestID })
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
    get preview() {
      return preview
    },
    get patch() {
      return patch
    },
    get fileLoading() {
      return fileLoading
    },
    get fileError() {
      return fileError
    },
    get selectedRepoPath() {
      return selectedRepoPath
    },
    get selectedFilePath() {
      return selectedFilePath
    },
    get selectedEditorState() {
      return editorStore.selectedEditorState
    },
    get selectedDraftDiff() {
      return editorStore.selectedDraftDiff
    },
    reset,
    refreshWorkspace,
    toggleDir,
    openRepo,
    selectFile,
    setSelectedViewMode: editorStore.setSelectedViewMode,
    updateSelectedDraft: editorStore.updateSelectedDraft,
    revertSelectedDraft: editorStore.revertSelectedDraft,
    keepSelectedDraft: editorStore.keepSelectedDraft,
    reloadSelectedSavedVersion: editorStore.reloadSelectedSavedVersion,
    saveSelectedFile: editorStore.saveSelectedFile,
  }
}

export type ProjectConversationWorkspaceBrowserState = ReturnType<
  typeof createProjectConversationWorkspaceBrowserState
>
