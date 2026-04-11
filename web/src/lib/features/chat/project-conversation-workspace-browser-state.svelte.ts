import {
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
  type ProjectConversationWorkspaceFilePatch,
  type ProjectConversationWorkspaceFilePreview,
  type ProjectConversationWorkspaceMetadata,
  type ProjectConversationWorkspaceTreeEntry,
} from '$lib/api/chat'

function areTreeEntriesEqual(
  left: ProjectConversationWorkspaceTreeEntry[] | undefined,
  right: ProjectConversationWorkspaceTreeEntry[],
) {
  if (!left) {
    return false
  }
  if (left.length !== right.length) {
    return false
  }

  return left.every((entry, index) => {
    const next = right[index]
    return (
      next != null &&
      entry.path === next.path &&
      entry.name === next.name &&
      entry.kind === next.kind &&
      entry.sizeBytes === next.sizeBytes
    )
  })
}

function areWorkspaceMetadataEqual(
  left: ProjectConversationWorkspaceMetadata | null,
  right: ProjectConversationWorkspaceMetadata,
) {
  if (!left) {
    return false
  }
  if (
    left.conversationId !== right.conversationId ||
    left.available !== right.available ||
    left.workspacePath !== right.workspacePath ||
    left.repos.length !== right.repos.length
  ) {
    return false
  }

  return left.repos.every((repo, index) => {
    const next = right.repos[index]
    return (
      next != null &&
      repo.name === next.name &&
      repo.path === next.path &&
      repo.branch === next.branch &&
      repo.headCommit === next.headCommit &&
      repo.headSummary === next.headSummary &&
      repo.dirty === next.dirty &&
      repo.filesChanged === next.filesChanged &&
      repo.added === next.added &&
      repo.removed === next.removed
    )
  })
}

function areFilePreviewEqual(
  left: ProjectConversationWorkspaceFilePreview | null,
  right: ProjectConversationWorkspaceFilePreview,
) {
  return (
    !!left &&
    left.conversationId === right.conversationId &&
    left.repoPath === right.repoPath &&
    left.path === right.path &&
    left.sizeBytes === right.sizeBytes &&
    left.mediaType === right.mediaType &&
    left.previewKind === right.previewKind &&
    left.truncated === right.truncated &&
    left.content === right.content
  )
}

function areFilePatchEqual(
  left: ProjectConversationWorkspaceFilePatch | null,
  right: ProjectConversationWorkspaceFilePatch,
) {
  return (
    !!left &&
    left.conversationId === right.conversationId &&
    left.repoPath === right.repoPath &&
    left.path === right.path &&
    left.status === right.status &&
    left.diffKind === right.diffKind &&
    left.truncated === right.truncated &&
    left.diff === right.diff
  )
}

export function createProjectConversationWorkspaceBrowserState(input: {
  getConversationId: () => string
}) {
  let metadata = $state<ProjectConversationWorkspaceMetadata | null>(null)
  let metadataLoading = $state(false)
  let metadataError = $state('')

  // Recursive tree: directory path → child entries
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
    const currentEntries = treeNodes.get(dirPath)
    if (areTreeEntriesEqual(currentEntries, entries)) {
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

  function reset() {
    metadata = null
    metadataLoading = false
    metadataError = ''
    treeNodes = new Map()
    expandedDirs = new Set()
    loadingDirs = new Set()
    preview = null
    patch = null
    fileLoading = false
    fileError = ''
    selectedRepoPath = ''
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

      setMetadata(payload.workspace)
      if (!payload.workspace.available || payload.workspace.repos.length === 0) {
        selectedRepoPath = ''
        selectedFilePath = ''
        treeNodes = new Map()
        expandedDirs = new Set()
        loadingDirs = new Set()
        setPreview(null)
        setPatch(null)
        fileError = ''
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
        selectedFilePath = ''
        expandedDirs = new Set()
        setPreview(null)
        setPatch(null)
        treeNodes = new Map()
      }

      await loadDirEntries('', requestID, { silent: treeNodes.has('') })
      if (requestID !== loadRequestID) return

      // Re-expand previously expanded directories
      if (prevExpanded.length > 0) {
        await Promise.all(
          prevExpanded.map((dir) => loadDirEntries(dir, requestID, { silent: treeNodes.has(dir) })),
        )
      }

      // Reload selected file if preserved
      if (preserveSelection && selectedFilePath && !repoChanged) {
        await loadFile(selectedFilePath, {
          requestID,
          silent: preview != null || patch != null,
        })
      }
    } catch (error) {
      if (requestID !== loadRequestID || conversationId !== input.getConversationId()) return
      metadata = null
      treeNodes = new Map()
      setPreview(null)
      setPatch(null)
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
    if (!repoPath || !conversationId) return

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
      if (requestID !== loadRequestID || repoPath !== selectedRepoPath) return

      setTreeEntries(dirPath, payload.workspaceTree.entries)
    } catch {
      // Individual directory load failures are silent — the directory shows as empty.
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
    if (!path) return

    const requestID = options.requestID ?? loadRequestID
    const silent = options.silent ?? false
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
      await loadDirEntries('', requestID, { silent })
    }

    for (const dirPath of ancestorDirs) {
      if (requestID !== loadRequestID || path !== selectedFilePath) {
        return
      }

      setDirExpanded(dirPath, true)
      if (!treeNodes.has(dirPath)) {
        await loadDirEntries(dirPath, requestID, { silent })
      }
    }
  }

  async function loadFile(path: string, options: { requestID?: number; silent?: boolean } = {}) {
    const conversationId = input.getConversationId()
    if (!selectedRepoPath || !conversationId) {
      setPreview(null)
      setPatch(null)
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
      setPreview(previewPayload.filePreview)
      setPatch(patchPayload.filePatch)
    } catch (error) {
      if (requestID !== loadRequestID || path !== selectedFilePath) return
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
    if (!repoPath || repoPath === selectedRepoPath) return
    selectedRepoPath = repoPath
    selectedFilePath = ''
    expandedDirs = new Set()
    treeNodes = new Map()
    setPreview(null)
    setPatch(null)
    void loadDirEntries('')
  }

  function selectFile(path: string) {
    if (!path) return
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
    reset,
    refreshWorkspace,
    toggleDir,
    openRepo,
    selectFile,
  }
}
