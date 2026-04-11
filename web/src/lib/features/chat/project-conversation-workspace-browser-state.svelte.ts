/* eslint-disable max-lines */

import { ApiError } from '$lib/api/client'
import {
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceDiff,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
  saveProjectConversationWorkspaceFile,
  type ProjectConversationWorkspaceDiff,
  type ProjectConversationWorkspaceFilePatch,
  type ProjectConversationWorkspaceFilePreview,
  type ProjectConversationWorkspaceMetadata,
  type ProjectConversationWorkspaceTreeEntry,
} from '$lib/api/chat'
import {
  deletePersistedWorkspaceFileDraft,
  loadPersistedWorkspaceFileDraft,
  savePersistedWorkspaceFileDraft,
  type WorkspaceFileSavePhase,
  type WorkspaceFileViewMode,
  workspaceFileDraftStorageKey,
} from './project-conversation-workspace-file-drafts'

export type WorkspaceFileEditorState = {
  baseSavedContent: string
  baseSavedRevision: string
  latestSavedContent: string
  latestSavedRevision: string
  draftContent: string
  dirty: boolean
  viewMode: WorkspaceFileViewMode
  savePhase: WorkspaceFileSavePhase
  externalChange: boolean
  errorMessage: string
  encoding: 'utf-8'
  lineEnding: 'lf' | 'crlf'
  lastSavedAt: string
}

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
    left.content === right.content &&
    left.revision === right.revision &&
    left.writable === right.writable &&
    left.readOnlyReason === right.readOnlyReason &&
    left.encoding === right.encoding &&
    left.lineEnding === right.lineEnding
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

function createInitialEditorState(
  preview: ProjectConversationWorkspaceFilePreview,
): WorkspaceFileEditorState {
  return {
    baseSavedContent: preview.content,
    baseSavedRevision: preview.revision,
    latestSavedContent: preview.content,
    latestSavedRevision: preview.revision,
    draftContent: preview.content,
    dirty: false,
    viewMode: 'preview',
    savePhase: 'idle',
    externalChange: false,
    errorMessage: '',
    encoding: preview.encoding,
    lineEnding: preview.lineEnding,
    lastSavedAt: '',
  }
}

function buildWholeFileDiff(filePath: string, savedContent: string, draftContent: string) {
  if (savedContent === draftContent) {
    return ''
  }
  const oldLines = savedContent.split('\n')
  const newLines = draftContent.split('\n')
  const oldCount = savedContent === '' ? 0 : oldLines.length
  const newCount = draftContent === '' ? 0 : newLines.length
  const diffLines = [
    `--- saved/${filePath}`,
    `+++ draft/${filePath}`,
    `@@ -1,${oldCount} +1,${newCount} @@`,
    ...oldLines.map((line) => `-${line}`),
    ...newLines.map((line) => `+${line}`),
  ]
  return diffLines.join('\n')
}

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
  let editorStates = $state<Map<string, WorkspaceFileEditorState>>(new Map())

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

  function selectedFileStorageKey(repoPath = selectedRepoPath, filePath = selectedFilePath) {
    return workspaceFileDraftStorageKey({
      conversationId: input.getConversationId(),
      repoPath,
      filePath,
    })
  }

  function getEditorState(repoPath = selectedRepoPath, filePath = selectedFilePath) {
    if (!repoPath || !filePath) {
      return null
    }
    return editorStates.get(selectedFileStorageKey(repoPath, filePath)) ?? null
  }

  function setEditorState(
    repoPath: string,
    filePath: string,
    nextState: WorkspaceFileEditorState | null,
  ) {
    const key = selectedFileStorageKey(repoPath, filePath)
    const nextEditorStates = new Map(editorStates)
    if (nextState) {
      nextEditorStates.set(key, nextState)
      if (nextState.dirty) {
        savePersistedWorkspaceFileDraft(key, {
          draftContent: nextState.draftContent,
          baseSavedContent: nextState.baseSavedContent,
          baseSavedRevision: nextState.baseSavedRevision,
          viewMode: nextState.viewMode,
          encoding: nextState.encoding,
          lineEnding: nextState.lineEnding,
          updatedAt: new Date().toISOString(),
        })
      } else {
        deletePersistedWorkspaceFileDraft(key)
      }
    } else {
      nextEditorStates.delete(key)
      deletePersistedWorkspaceFileDraft(key)
    }
    editorStates = nextEditorStates
  }

  function syncEditorStateFromPreview(
    repoPath: string,
    filePath: string,
    nextPreview: ProjectConversationWorkspaceFilePreview,
  ) {
    const key = selectedFileStorageKey(repoPath, filePath)
    const existing = editorStates.get(key)
    if (!existing) {
      const persisted = loadPersistedWorkspaceFileDraft(key)
      if (!persisted) {
        setEditorState(repoPath, filePath, createInitialEditorState(nextPreview))
        return
      }
      const dirty = persisted.draftContent !== nextPreview.content
      setEditorState(repoPath, filePath, {
        baseSavedContent: persisted.baseSavedContent,
        baseSavedRevision: persisted.baseSavedRevision,
        latestSavedContent: nextPreview.content,
        latestSavedRevision: nextPreview.revision,
        draftContent: persisted.draftContent,
        dirty,
        viewMode: dirty ? 'edit' : persisted.viewMode,
        savePhase: 'idle',
        externalChange: dirty && persisted.baseSavedRevision !== nextPreview.revision,
        errorMessage: '',
        encoding: nextPreview.encoding,
        lineEnding: nextPreview.lineEnding,
        lastSavedAt: '',
      })
      return
    }

    if (existing.dirty) {
      const latestChanged = existing.latestSavedRevision !== nextPreview.revision
      setEditorState(repoPath, filePath, {
        ...existing,
        latestSavedContent: nextPreview.content,
        latestSavedRevision: nextPreview.revision,
        dirty: existing.draftContent !== nextPreview.content,
        externalChange: latestChanged || existing.baseSavedRevision !== nextPreview.revision,
        encoding: nextPreview.encoding,
        lineEnding: nextPreview.lineEnding,
      })
      return
    }

    setEditorState(repoPath, filePath, {
      ...existing,
      baseSavedContent: nextPreview.content,
      baseSavedRevision: nextPreview.revision,
      latestSavedContent: nextPreview.content,
      latestSavedRevision: nextPreview.revision,
      draftContent: nextPreview.content,
      dirty: false,
      externalChange: false,
      savePhase: 'idle',
      errorMessage: '',
      encoding: nextPreview.encoding,
      lineEnding: nextPreview.lineEnding,
    })
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
    editorStates = new Map()
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

      if (prevExpanded.length > 0) {
        await Promise.all(
          prevExpanded.map((dir) => loadDirEntries(dir, requestID, { silent: treeNodes.has(dir) })),
        )
      }

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
      syncEditorStateFromPreview(selectedRepoPath, path, previewPayload.filePreview)
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

  function setSelectedViewMode(viewMode: WorkspaceFileViewMode) {
    const editor = getEditorState()
    if (!editor || !selectedRepoPath || !selectedFilePath) {
      return
    }
    setEditorState(selectedRepoPath, selectedFilePath, { ...editor, viewMode })
  }

  function updateSelectedDraft(nextDraftContent: string) {
    const editor = getEditorState()
    if (!editor || !selectedRepoPath || !selectedFilePath) {
      return
    }
    const dirty = nextDraftContent !== editor.latestSavedContent
    setEditorState(selectedRepoPath, selectedFilePath, {
      ...editor,
      draftContent: nextDraftContent,
      dirty,
      viewMode: 'edit',
      savePhase: editor.savePhase === 'saving' ? editor.savePhase : 'idle',
      errorMessage: '',
    })
  }

  function revertSelectedDraft() {
    const editor = getEditorState()
    if (!editor || !selectedRepoPath || !selectedFilePath) {
      return
    }
    setEditorState(selectedRepoPath, selectedFilePath, {
      ...editor,
      baseSavedContent: editor.latestSavedContent,
      baseSavedRevision: editor.latestSavedRevision,
      draftContent: editor.latestSavedContent,
      dirty: false,
      savePhase: 'idle',
      externalChange: false,
      errorMessage: '',
      viewMode: 'preview',
    })
  }

  function keepSelectedDraft() {
    const editor = getEditorState()
    if (!editor || !selectedRepoPath || !selectedFilePath) {
      return
    }
    setEditorState(selectedRepoPath, selectedFilePath, {
      ...editor,
      baseSavedContent: editor.latestSavedContent,
      baseSavedRevision: editor.latestSavedRevision,
      dirty: editor.draftContent !== editor.latestSavedContent,
      savePhase: 'idle',
      externalChange: false,
      errorMessage: '',
      viewMode: 'edit',
    })
  }

  function reloadSelectedSavedVersion() {
    revertSelectedDraft()
  }

  async function refreshWorkspaceDiff() {
    const conversationId = input.getConversationId()
    if (!conversationId || !input.onWorkspaceDiffUpdated) {
      return
    }
    const payload = await getProjectConversationWorkspaceDiff(conversationId)
    input.onWorkspaceDiffUpdated(payload.workspaceDiff)
  }

  async function saveSelectedFile() {
    const conversationId = input.getConversationId()
    const editor = getEditorState()
    if (
      !conversationId ||
      !selectedRepoPath ||
      !selectedFilePath ||
      !editor ||
      !preview?.writable
    ) {
      return
    }
    if (!editor.dirty || editor.savePhase === 'saving') {
      return
    }

    setEditorState(selectedRepoPath, selectedFilePath, {
      ...editor,
      savePhase: 'saving',
      errorMessage: '',
    })

    try {
      const payload = await saveProjectConversationWorkspaceFile(conversationId, {
        repoPath: selectedRepoPath,
        path: selectedFilePath,
        baseRevision: editor.baseSavedRevision,
        content: editor.draftContent,
        encoding: editor.encoding,
        lineEnding: editor.lineEnding,
      })
      const nextEditor = getEditorState()
      if (!nextEditor) {
        return
      }
      const now = new Date().toISOString()
      setEditorState(selectedRepoPath, selectedFilePath, {
        ...nextEditor,
        baseSavedContent: nextEditor.draftContent,
        baseSavedRevision: payload.file.revision,
        latestSavedContent: nextEditor.draftContent,
        latestSavedRevision: payload.file.revision,
        dirty: false,
        savePhase: 'idle',
        externalChange: false,
        errorMessage: '',
        lastSavedAt: now,
      })
      await loadFile(selectedFilePath, { silent: true })
      try {
        await refreshWorkspaceDiff()
      } catch {
        const refreshedEditor = getEditorState()
        if (refreshedEditor) {
          setEditorState(selectedRepoPath, selectedFilePath, {
            ...refreshedEditor,
            errorMessage: 'Saved, but the workspace summary could not be refreshed.',
          })
        }
      }
    } catch (error) {
      const latestEditor = getEditorState()
      if (!latestEditor) {
        return
      }
      if (
        error instanceof ApiError &&
        error.status === 409 &&
        error.code === 'PROJECT_CONVERSATION_WORKSPACE_FILE_CONFLICT'
      ) {
        const currentFile = (
          error.details as { current_file?: ProjectConversationWorkspaceFilePreview } | undefined
        )?.current_file
        if (currentFile) {
          setEditorState(selectedRepoPath, selectedFilePath, {
            ...latestEditor,
            latestSavedContent: currentFile.content,
            latestSavedRevision: currentFile.revision,
            dirty: latestEditor.draftContent !== currentFile.content,
            savePhase: 'conflict',
            externalChange: true,
            errorMessage: 'The workspace file changed before your save completed.',
            encoding: currentFile.encoding,
            lineEnding: currentFile.lineEnding,
            viewMode: 'diff',
          })
          setPreview(currentFile)
        } else {
          setEditorState(selectedRepoPath, selectedFilePath, {
            ...latestEditor,
            savePhase: 'conflict',
            externalChange: true,
            errorMessage: error.message,
          })
        }
        return
      }
      setEditorState(selectedRepoPath, selectedFilePath, {
        ...latestEditor,
        savePhase: 'error',
        errorMessage: error instanceof Error ? error.message : 'Failed to save the workspace file.',
      })
    }
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
      return getEditorState()
    },
    get selectedDraftDiff() {
      const editor = getEditorState()
      if (!editor || !selectedFilePath) {
        return ''
      }
      return buildWholeFileDiff(selectedFilePath, editor.latestSavedContent, editor.draftContent)
    },
    reset,
    refreshWorkspace,
    toggleDir,
    openRepo,
    selectFile,
    setSelectedViewMode,
    updateSelectedDraft,
    revertSelectedDraft,
    keepSelectedDraft,
    reloadSelectedSavedVersion,
    saveSelectedFile,
  }
}
