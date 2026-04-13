/* eslint-disable max-lines */
import { ApiError } from '$lib/api/client'
import {
  saveProjectConversationWorkspaceFile,
  type ChatDiffPayload,
  type ProjectConversationWorkspaceFilePreview,
} from '$lib/api/chat'
import {
  deletePersistedWorkspaceFileDraft,
  loadPersistedWorkspaceFileDraft,
  savePersistedWorkspaceFileDraft,
  workspaceFileDraftStorageKey,
} from './project-conversation-workspace-file-drafts'
import {
  buildWorkspaceSelection,
  buildWorkspaceWorkingSet,
  createWorkspacePatchProposal,
  formatWorkspaceDocument,
  formatWorkspaceSelection,
  type WorkspaceSelectionInput,
} from './project-conversation-workspace-editor-helpers'
import {
  computeDraftLineDiff,
  createInitialEditorState,
  type WorkspaceFileEditorState,
  type WorkspaceFileLineDiffMarkers,
  type WorkspaceRecentFile,
} from './project-conversation-workspace-browser-state-helpers'
import { chatT } from './i18n'

const WORKSPACE_AUTOSAVE_DELAY_MS = 1000

export function createWorkspaceFileEditorStore(input: {
  getConversationId: () => string
  getSelectedRepoPath: () => string
  getSelectedFilePath: () => string
  getRepoRefCacheKey?: (repoPath: string) => string
  getPreview: (repoPath: string, filePath: string) => ProjectConversationWorkspaceFilePreview | null
  setPreview: (
    repoPath: string,
    filePath: string,
    preview: ProjectConversationWorkspaceFilePreview | null,
  ) => void
  reloadSelectedFile: (repoPath: string, filePath: string) => Promise<void>
  refreshWorkspaceDiff?: () => Promise<void>
  getAutosaveEnabled?: () => boolean
}) {
  let editorStates = $state<Map<string, WorkspaceFileEditorState>>(new Map())
  const autosaveTimers = new Map<string, ReturnType<typeof setTimeout>>()

  function selectedFileStorageKey(
    repoPath = input.getSelectedRepoPath(),
    filePath = input.getSelectedFilePath(),
  ) {
    return workspaceFileDraftStorageKey({
      conversationId: input.getConversationId(),
      repoPath,
      refCacheKey: input.getRepoRefCacheKey?.(repoPath) ?? '',
      filePath,
    })
  }

  function cancelAutosave(key: string) {
    const existing = autosaveTimers.get(key)
    if (!existing) {
      return
    }
    clearTimeout(existing)
    autosaveTimers.delete(key)
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
          encoding: nextState.encoding,
          lineEnding: nextState.lineEnding,
          updatedAt: new Date().toISOString(),
        })
      } else {
        deletePersistedWorkspaceFileDraft(key)
      }
      scheduleAutosave(repoPath, filePath, nextState)
    } else {
      cancelAutosave(key)
      nextEditorStates.delete(key)
      deletePersistedWorkspaceFileDraft(key)
    }

    editorStates = nextEditorStates
  }

  function scheduleAutosave(
    repoPath: string,
    filePath: string,
    editorState: WorkspaceFileEditorState,
  ) {
    const key = selectedFileStorageKey(repoPath, filePath)
    cancelAutosave(key)

    if (
      !input.getAutosaveEnabled?.() ||
      !editorState.dirty ||
      editorState.savePhase === 'saving' ||
      editorState.savePhase === 'conflict' ||
      editorState.externalChange ||
      input.getPreview(repoPath, filePath)?.writable !== true
    ) {
      return
    }

    autosaveTimers.set(
      key,
      setTimeout(() => {
        autosaveTimers.delete(key)
        void saveFile(repoPath, filePath)
      }, WORKSPACE_AUTOSAVE_DELAY_MS),
    )
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
        savePhase: 'idle',
        externalChange: dirty && persisted.baseSavedRevision !== nextPreview.revision,
        errorMessage: '',
        encoding: nextPreview.encoding,
        lineEnding: nextPreview.lineEnding,
        lastSavedAt: '',
        selection: null,
        pendingPatch: null,
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
        selection: buildWorkspaceSelection(existing.draftContent, existing.selection),
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
      selection: null,
      pendingPatch: null,
    })
  }

  function reset() {
    for (const key of autosaveTimers.keys()) {
      cancelAutosave(key)
    }
    editorStates = new Map()
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
      savePhase: editor.savePhase === 'saving' ? editor.savePhase : 'idle',
      errorMessage: '',
      selection: buildWorkspaceSelection(nextDraftContent, editor.selection),
      pendingPatch: null,
    })
  }

  function updateSelectedSelection(selection: WorkspaceSelectionInput | null) {
    const repoPath = input.getSelectedRepoPath()
    const filePath = input.getSelectedFilePath()
    const editor = getEditorState(repoPath, filePath)
    if (!editor || !repoPath || !filePath) {
      return
    }

    setEditorState(repoPath, filePath, {
      ...editor,
      selection: buildWorkspaceSelection(editor.draftContent, selection),
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
      selection: null,
      pendingPatch: null,
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
    })
  }

  function discardSelectedDraft() {
    const repoPath = input.getSelectedRepoPath()
    const filePath = input.getSelectedFilePath()
    if (!repoPath || !filePath) return
    setEditorState(repoPath, filePath, null)
  }

  function discardDraft(repoPath: string, filePath: string) {
    if (!repoPath || !filePath) return
    setEditorState(repoPath, filePath, null)
  }

  function reloadSelectedSavedVersion() {
    revertSelectedDraft()
  }

  function reviewPatch(repoPath: string, filePath: string, diff: ChatDiffPayload) {
    const editor = getEditorState(repoPath, filePath)
    if (!editor) {
      return false
    }
    const proposal = createWorkspacePatchProposal(editor.draftContent, diff)
    if (!proposal) {
      setEditorState(repoPath, filePath, {
        ...editor,
        errorMessage: 'The draft changed and this Project AI patch no longer applies cleanly.',
        pendingPatch: null,
      })
      return false
    }
    setEditorState(repoPath, filePath, {
      ...editor,
      errorMessage: '',
      pendingPatch: proposal,
    })
    return true
  }

  function applyPendingPatch(repoPath: string, filePath: string) {
    const editor = getEditorState(repoPath, filePath)
    const proposal = editor?.pendingPatch
    if (!editor || !proposal) {
      return false
    }
    setEditorState(repoPath, filePath, {
      ...editor,
      draftContent: proposal.proposedContent,
      dirty: proposal.proposedContent !== editor.latestSavedContent,
      savePhase: 'idle',
      errorMessage: '',
      selection: buildWorkspaceSelection(proposal.proposedContent, editor.selection),
      pendingPatch: null,
    })
    return true
  }

  function discardPendingPatch(
    repoPath = input.getSelectedRepoPath(),
    filePath = input.getSelectedFilePath(),
  ) {
    const editor = getEditorState(repoPath, filePath)
    if (!editor || !repoPath || !filePath) {
      return
    }
    setEditorState(repoPath, filePath, {
      ...editor,
      pendingPatch: null,
      errorMessage: '',
    })
  }

  function formatSelectedDocument() {
    const repoPath = input.getSelectedRepoPath()
    const filePath = input.getSelectedFilePath()
    const editor = getEditorState(repoPath, filePath)
    if (!editor || !repoPath || !filePath) {
      return false
    }
    try {
      const formatted = formatWorkspaceDocument(filePath, editor.draftContent)
      if (formatted == null) {
        setEditorState(repoPath, filePath, {
          ...editor,
          errorMessage: chatT('chat.formatDocumentUnavailable'),
        })
        return false
      }
      setEditorState(repoPath, filePath, {
        ...editor,
        draftContent: formatted,
        dirty: formatted !== editor.latestSavedContent,
        errorMessage: '',
        selection: buildWorkspaceSelection(formatted, editor.selection),
      })
      return true
    } catch (error) {
      setEditorState(repoPath, filePath, {
        ...editor,
        errorMessage: error instanceof Error ? error.message : chatT('chat.formatDocumentFailed'),
      })
      return false
    }
  }

  function formatSelectedSelection() {
    const repoPath = input.getSelectedRepoPath()
    const filePath = input.getSelectedFilePath()
    const editor = getEditorState(repoPath, filePath)
    if (!editor || !repoPath || !filePath) {
      return false
    }
    try {
      const formatted = formatWorkspaceSelection(filePath, editor.draftContent, editor.selection)
      if (formatted == null) {
        setEditorState(repoPath, filePath, {
          ...editor,
          errorMessage: chatT('chat.formatSelectionUnavailable'),
        })
        return false
      }
      setEditorState(repoPath, filePath, {
        ...editor,
        draftContent: formatted.content,
        dirty: formatted.content !== editor.latestSavedContent,
        errorMessage: '',
        selection: buildWorkspaceSelection(formatted.content, formatted.selection),
      })
      return true
    } catch (error) {
      setEditorState(repoPath, filePath, {
        ...editor,
        errorMessage: error instanceof Error ? error.message : chatT('chat.formatSelectionFailed'),
      })
      return false
    }
  }

  function renameFileState(repoPath: string, fromPath: string, toPath: string) {
    const fromKey = selectedFileStorageKey(repoPath, fromPath)
    const toKey = selectedFileStorageKey(repoPath, toPath)
    const editor = editorStates.get(fromKey)
    const nextStates = new Map(editorStates)
    const persisted = loadPersistedWorkspaceFileDraft(fromKey)
    cancelAutosave(fromKey)
    if (editor) {
      nextStates.delete(fromKey)
      nextStates.set(toKey, editor)
    }
    editorStates = nextStates
    if (persisted) {
      savePersistedWorkspaceFileDraft(toKey, persisted)
      deletePersistedWorkspaceFileDraft(fromKey)
    }
  }

  function buildWorkingSet(recentFiles: WorkspaceRecentFile[]) {
    return buildWorkspaceWorkingSet(
      recentFiles
        .map((item) => {
          const editor = getEditorState(item.repoPath, item.filePath)
          const preview = input.getPreview(item.repoPath, item.filePath)
          const content = editor?.draftContent ?? preview?.content ?? ''
          if (!content) {
            return null
          }
          return {
            filePath: item.filePath,
            content,
            dirty: editor?.dirty ?? false,
          }
        })
        .filter(
          (item): item is { filePath: string; content: string; dirty: boolean } => item != null,
        ),
    )
  }

  async function saveFile(repoPath: string, filePath: string): Promise<boolean> {
    const conversationId = input.getConversationId()
    const preview = input.getPreview(repoPath, filePath)
    const editor = getEditorState(repoPath, filePath)

    if (!conversationId || !repoPath || !filePath || !editor || !preview?.writable) {
      return false
    }
    if (!editor.dirty || editor.savePhase === 'saving') {
      return !editor.dirty
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
        return false
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
        pendingPatch: null,
      })

      await input.reloadSelectedFile(repoPath, filePath)
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
      return true
    } catch (error) {
      const latestEditor = getEditorState(repoPath, filePath)
      if (!latestEditor) {
        return false
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
            errorMessage:
              'The workspace file changed before your save completed. Autosave is paused until you resolve the conflict.',
            encoding: currentFile.encoding,
            lineEnding: currentFile.lineEnding,
          })
          input.setPreview(repoPath, filePath, currentFile)
        } else {
          setEditorState(repoPath, filePath, {
            ...latestEditor,
            savePhase: 'conflict',
            externalChange: true,
            errorMessage: error.message,
          })
        }
        return false
      }

      setEditorState(repoPath, filePath, {
        ...latestEditor,
        savePhase: 'error',
        errorMessage: error instanceof Error ? error.message : chatT('chat.workspace.errors.save'),
      })
      return false
    }
  }

  async function saveSelectedFile(): Promise<boolean> {
    return saveFile(input.getSelectedRepoPath(), input.getSelectedFilePath())
  }

  return {
    get selectedEditorState() {
      return getEditorState()
    },
    get selectedDraftLineDiff(): WorkspaceFileLineDiffMarkers | null {
      const repoPath = input.getSelectedRepoPath()
      const filePath = input.getSelectedFilePath()
      const editor = getEditorState(repoPath, filePath)
      if (!editor || !filePath) return null
      return computeDraftLineDiff(editor.latestSavedContent, editor.draftContent)
    },
    getEditorState,
    reset,
    syncFromPreview,
    updateSelectedDraft,
    updateSelectedSelection,
    revertSelectedDraft,
    keepSelectedDraft,
    reloadSelectedSavedVersion,
    reviewPatch,
    applyPendingPatch,
    discardPendingPatch,
    formatSelectedDocument,
    formatSelectedSelection,
    renameFileState,
    buildWorkingSet,
    saveSelectedFile,
    saveFile,
    discardSelectedDraft,
    discardDraft,
  }
}
