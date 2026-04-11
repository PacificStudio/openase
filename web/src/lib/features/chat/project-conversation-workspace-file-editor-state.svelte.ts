import { ApiError } from '$lib/api/client'
import {
  saveProjectConversationWorkspaceFile,
  type ProjectConversationWorkspaceFilePreview,
} from '$lib/api/chat'
import {
  deletePersistedWorkspaceFileDraft,
  loadPersistedWorkspaceFileDraft,
  savePersistedWorkspaceFileDraft,
  workspaceFileDraftStorageKey,
  type WorkspaceFileViewMode,
} from './project-conversation-workspace-file-drafts'
import {
  buildWholeFileDiff,
  createInitialEditorState,
  type WorkspaceFileEditorState,
} from './project-conversation-workspace-browser-state-helpers'

export function createWorkspaceFileEditorStore(input: {
  getConversationId: () => string
  getSelectedRepoPath: () => string
  getSelectedFilePath: () => string
  getPreview: () => ProjectConversationWorkspaceFilePreview | null
  setPreview: (preview: ProjectConversationWorkspaceFilePreview | null) => void
  reloadSelectedFile: () => Promise<void>
  refreshWorkspaceDiff?: () => Promise<void>
}) {
  let editorStates = $state<Map<string, WorkspaceFileEditorState>>(new Map())

  function selectedFileStorageKey(
    repoPath = input.getSelectedRepoPath(),
    filePath = input.getSelectedFilePath(),
  ) {
    return workspaceFileDraftStorageKey({
      conversationId: input.getConversationId(),
      repoPath,
      filePath,
    })
  }

  function getEditorState(
    repoPath = input.getSelectedRepoPath(),
    filePath = input.getSelectedFilePath(),
  ) {
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

  function syncFromPreview(
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
    editorStates = new Map()
  }

  function setSelectedViewMode(viewMode: WorkspaceFileViewMode) {
    const repoPath = input.getSelectedRepoPath()
    const filePath = input.getSelectedFilePath()
    const editor = getEditorState(repoPath, filePath)
    if (!editor || !repoPath || !filePath) {
      return
    }
    setEditorState(repoPath, filePath, { ...editor, viewMode })
  }

  function updateSelectedDraft(nextDraftContent: string) {
    const repoPath = input.getSelectedRepoPath()
    const filePath = input.getSelectedFilePath()
    const editor = getEditorState(repoPath, filePath)
    if (!editor || !repoPath || !filePath) {
      return
    }

    setEditorState(repoPath, filePath, {
      ...editor,
      draftContent: nextDraftContent,
      dirty: nextDraftContent !== editor.latestSavedContent,
      viewMode: 'edit',
      savePhase: editor.savePhase === 'saving' ? editor.savePhase : 'idle',
      errorMessage: '',
    })
  }

  function revertSelectedDraft() {
    const repoPath = input.getSelectedRepoPath()
    const filePath = input.getSelectedFilePath()
    const editor = getEditorState(repoPath, filePath)
    if (!editor || !repoPath || !filePath) {
      return
    }

    setEditorState(repoPath, filePath, {
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
    const repoPath = input.getSelectedRepoPath()
    const filePath = input.getSelectedFilePath()
    const editor = getEditorState(repoPath, filePath)
    if (!editor || !repoPath || !filePath) {
      return
    }

    setEditorState(repoPath, filePath, {
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

  async function saveSelectedFile() {
    const conversationId = input.getConversationId()
    const repoPath = input.getSelectedRepoPath()
    const filePath = input.getSelectedFilePath()
    const preview = input.getPreview()
    const editor = getEditorState(repoPath, filePath)

    if (!conversationId || !repoPath || !filePath || !editor || !preview?.writable) {
      return
    }
    if (!editor.dirty || editor.savePhase === 'saving') {
      return
    }

    setEditorState(repoPath, filePath, {
      ...editor,
      savePhase: 'saving',
      errorMessage: '',
    })

    try {
      const payload = await saveProjectConversationWorkspaceFile(conversationId, {
        repoPath,
        path: filePath,
        baseRevision: editor.baseSavedRevision,
        content: editor.draftContent,
        encoding: editor.encoding,
        lineEnding: editor.lineEnding,
      })
      const nextEditor = getEditorState(repoPath, filePath)
      if (!nextEditor) {
        return
      }

      setEditorState(repoPath, filePath, {
        ...nextEditor,
        baseSavedContent: nextEditor.draftContent,
        baseSavedRevision: payload.file.revision,
        latestSavedContent: nextEditor.draftContent,
        latestSavedRevision: payload.file.revision,
        dirty: false,
        savePhase: 'idle',
        externalChange: false,
        errorMessage: '',
        lastSavedAt: new Date().toISOString(),
      })

      await input.reloadSelectedFile()
      try {
        await input.refreshWorkspaceDiff?.()
      } catch {
        const refreshedEditor = getEditorState(repoPath, filePath)
        if (refreshedEditor) {
          setEditorState(repoPath, filePath, {
            ...refreshedEditor,
            errorMessage: 'Saved, but the workspace summary could not be refreshed.',
          })
        }
      }
    } catch (error) {
      const latestEditor = getEditorState(repoPath, filePath)
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
          setEditorState(repoPath, filePath, {
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
          input.setPreview(currentFile)
        } else {
          setEditorState(repoPath, filePath, {
            ...latestEditor,
            savePhase: 'conflict',
            externalChange: true,
            errorMessage: error.message,
          })
        }
        return
      }

      setEditorState(repoPath, filePath, {
        ...latestEditor,
        savePhase: 'error',
        errorMessage: error instanceof Error ? error.message : 'Failed to save the workspace file.',
      })
    }
  }

  return {
    get selectedEditorState() {
      return getEditorState()
    },
    get selectedDraftDiff() {
      const filePath = input.getSelectedFilePath()
      const editor = getEditorState(input.getSelectedRepoPath(), filePath)
      if (!editor || !filePath) {
        return ''
      }
      return buildWholeFileDiff(filePath, editor.latestSavedContent, editor.draftContent)
    },
    reset,
    syncFromPreview,
    setSelectedViewMode,
    updateSelectedDraft,
    revertSelectedDraft,
    keepSelectedDraft,
    reloadSelectedSavedVersion,
    saveSelectedFile,
  }
}
