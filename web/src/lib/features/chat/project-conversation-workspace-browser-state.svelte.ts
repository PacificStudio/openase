import {
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
  type ProjectConversationWorkspaceFilePatch,
  type ProjectConversationWorkspaceFilePreview,
  type ProjectConversationWorkspaceMetadata,
  type ProjectConversationWorkspaceTree,
  type ProjectConversationWorkspaceTreeEntry,
} from '$lib/api/chat'

export function createProjectConversationWorkspaceBrowserState(input: {
  getConversationId: () => string
}) {
  let metadata = $state<ProjectConversationWorkspaceMetadata | null>(null)
  let metadataLoading = $state(false)
  let metadataError = $state('')
  let tree = $state<ProjectConversationWorkspaceTree | null>(null)
  let treeLoading = $state(false)
  let treeError = $state('')
  let preview = $state<ProjectConversationWorkspaceFilePreview | null>(null)
  let patch = $state<ProjectConversationWorkspaceFilePatch | null>(null)
  let fileLoading = $state(false)
  let fileError = $state('')
  let selectedRepoPath = $state('')
  let currentTreePath = $state('')
  let selectedFilePath = $state('')
  let loadRequestID = 0

  function reset() {
    metadata = null
    metadataLoading = false
    metadataError = ''
    tree = null
    treeLoading = false
    treeError = ''
    preview = null
    patch = null
    fileLoading = false
    fileError = ''
    selectedRepoPath = ''
    currentTreePath = ''
    selectedFilePath = ''
  }

  async function refreshWorkspace(preserveSelection: boolean) {
    const conversationId = input.getConversationId()
    const requestID = ++loadRequestID
    metadataLoading = true
    metadataError = ''

    try {
      const payload = await getProjectConversationWorkspace(conversationId)
      if (requestID !== loadRequestID || conversationId !== input.getConversationId()) return

      metadata = payload.workspace
      if (!payload.workspace.available || payload.workspace.repos.length === 0) {
        selectedRepoPath = ''
        currentTreePath = ''
        selectedFilePath = ''
        tree = null
        preview = null
        patch = null
        treeError = ''
        fileError = ''
        return
      }

      const nextRepoPath =
        preserveSelection &&
        payload.workspace.repos.some((repo) => repo.path === selectedRepoPath) &&
        selectedRepoPath
          ? selectedRepoPath
          : (payload.workspace.repos[0]?.path ?? '')
      const nextTreePath =
        preserveSelection && nextRepoPath === selectedRepoPath ? currentTreePath : ''
      const nextSelectedFilePath =
        preserveSelection && nextRepoPath === selectedRepoPath ? selectedFilePath : ''

      selectedRepoPath = nextRepoPath
      currentTreePath = nextTreePath
      selectedFilePath = nextSelectedFilePath
      await loadTree(nextRepoPath, nextTreePath, { preserveSelectedFile: true, requestID })
    } catch (error) {
      if (requestID !== loadRequestID || conversationId !== input.getConversationId()) return
      metadata = null
      tree = null
      preview = null
      patch = null
      metadataError =
        error instanceof Error ? error.message : 'Failed to load the Project AI workspace.'
    } finally {
      if (requestID === loadRequestID && conversationId === input.getConversationId()) {
        metadataLoading = false
      }
    }
  }

  async function loadTree(
    repoPath: string,
    path: string,
    options: { preserveSelectedFile?: boolean; requestID?: number } = {},
  ) {
    const conversationId = input.getConversationId()
    if (!repoPath || !conversationId) {
      tree = null
      return
    }

    const requestID = options.requestID ?? ++loadRequestID
    treeLoading = true
    treeError = ''

    try {
      const payload = await listProjectConversationWorkspaceTree(conversationId, {
        repoPath,
        path,
      })
      if (requestID !== loadRequestID || repoPath !== selectedRepoPath) return

      tree = payload.workspaceTree
      currentTreePath = payload.workspaceTree.path

      const canKeepSelectedFile =
        options.preserveSelectedFile &&
        selectedFilePath &&
        selectedFilePath.startsWith(currentTreePath ? `${currentTreePath}/` : '')
      if (!canKeepSelectedFile) {
        selectedFilePath = ''
        preview = null
        patch = null
        fileError = ''
      }
      if (canKeepSelectedFile && selectedFilePath) {
        await loadFile(selectedFilePath, { requestID })
      }
    } catch (error) {
      if (requestID !== loadRequestID || repoPath !== selectedRepoPath) return
      tree = null
      treeError = error instanceof Error ? error.message : 'Failed to load the workspace file tree.'
    } finally {
      if (requestID === loadRequestID && repoPath === selectedRepoPath) {
        treeLoading = false
      }
    }
  }

  async function loadFile(path: string, options: { requestID?: number } = {}) {
    const conversationId = input.getConversationId()
    if (!selectedRepoPath || !conversationId) {
      preview = null
      patch = null
      return
    }

    const requestID = options.requestID ?? ++loadRequestID
    fileLoading = true
    fileError = ''
    selectedFilePath = path

    try {
      const [previewPayload, patchPayload] = await Promise.all([
        getProjectConversationWorkspaceFilePreview(conversationId, {
          repoPath: selectedRepoPath,
          path,
        }),
        getProjectConversationWorkspaceFilePatch(conversationId, {
          repoPath: selectedRepoPath,
          path,
        }),
      ])
      if (requestID !== loadRequestID || path !== selectedFilePath) return
      preview = previewPayload.filePreview
      patch = patchPayload.filePatch
    } catch (error) {
      if (requestID !== loadRequestID || path !== selectedFilePath) return
      preview = null
      patch = null
      fileError =
        error instanceof Error ? error.message : 'Failed to load the workspace file details.'
    } finally {
      if (requestID === loadRequestID && path === selectedFilePath) {
        fileLoading = false
      }
    }
  }

  function openRepo(repoPath: string) {
    if (!repoPath || repoPath === selectedRepoPath) return
    selectedRepoPath = repoPath
    currentTreePath = ''
    selectedFilePath = ''
    preview = null
    patch = null
    void loadTree(repoPath, '', { preserveSelectedFile: false })
  }

  function openTreePath(path: string) {
    void loadTree(selectedRepoPath, path, { preserveSelectedFile: true })
  }

  function openEntry(entry: ProjectConversationWorkspaceTreeEntry) {
    if (entry.kind === 'directory') {
      selectedFilePath = ''
      preview = null
      patch = null
      void loadTree(selectedRepoPath, entry.path, { preserveSelectedFile: false })
      return
    }
    void loadFile(entry.path)
  }

  function openDirtyFile(path: string) {
    if (!path) return
    selectedFilePath = path
    void loadFile(path)
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
    get tree() {
      return tree
    },
    get treeLoading() {
      return treeLoading
    },
    get treeError() {
      return treeError
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
    get currentTreePath() {
      return currentTreePath
    },
    get selectedFilePath() {
      return selectedFilePath
    },
    reset,
    refreshWorkspace,
    openRepo,
    openTreePath,
    openEntry,
    openDirtyFile,
  }
}
