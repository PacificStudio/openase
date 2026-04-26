import { type ChatDiffPayload, type ProjectConversationWorkspaceFilePreview } from '$lib/api/chat'
import { saveWorkspaceFile } from './workspace-file-editor-save'
import {
  deletePersistedWorkspaceFileDraft,
  loadPersistedWorkspaceFileDraft,
  savePersistedWorkspaceFileDraft,
  workspaceFileDraftStorageKey,
} from './project-conversation-workspace-file-drafts'
import { type WorkspaceSelectionInput } from './project-conversation-workspace-editor-helpers'
import {
  type WorkspaceFileEditorState,
  type WorkspaceFileLineDiffMarkers,
  type WorkspaceRecentFile,
} from './project-conversation-workspace-browser-state-helpers'
import {
  buildWorkspaceEditorWorkingSet,
  getSelectedWorkspaceDraftLineDiff,
} from './project-conversation-workspace-file-editor-derived'
import {
  applyWorkspaceEditorPendingPatch,
  formatWorkspaceEditorDocument,
  formatWorkspaceEditorSelection,
  keepWorkspaceEditorDraft,
  revertWorkspaceEditorDraft,
  reviewWorkspaceEditorPatch,
  syncWorkspaceEditorStateFromPreview,
  updateWorkspaceEditorDraft,
  updateWorkspaceEditorSelection,
} from './project-conversation-workspace-file-editor-state-transforms'
import { createWorkspaceFileEditorStoreApi } from './project-conversation-workspace-file-editor-store-api'

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
    setEditorState(
      repoPath,
      filePath,
      syncWorkspaceEditorStateFromPreview({
        existing: editorStates.get(key) ?? null,
        persisted: loadPersistedWorkspaceFileDraft(key),
        nextPreview,
      }),
    )
  }
  function reset() {
    for (const key of autosaveTimers.keys()) {
      cancelAutosave(key)
    }
    editorStates = new Map()
  }
  function withSelectedEditor<TResult>(
    fallback: TResult,
    run: (repoPath: string, filePath: string, editor: WorkspaceFileEditorState) => TResult,
  ): TResult {
    const repoPath = input.getSelectedRepoPath()
    const filePath = input.getSelectedFilePath()
    const editor = getEditorState(repoPath, filePath)
    if (!editor || !repoPath || !filePath) {
      return fallback
    }
    return run(repoPath, filePath, editor)
  }
  function updateSelectedEditor(
    transform: (editor: WorkspaceFileEditorState) => WorkspaceFileEditorState | null,
  ) {
    withSelectedEditor(undefined, (repoPath, filePath, editor) => {
      setEditorState(repoPath, filePath, transform(editor))
      return undefined
    })
  }
  function updateSelectedDraft(nextDraftContent: string) {
    updateSelectedEditor((editor) => updateWorkspaceEditorDraft(editor, nextDraftContent))
  }
  function updateSelectedSelection(selection: WorkspaceSelectionInput | null) {
    updateSelectedEditor((editor) => updateWorkspaceEditorSelection(editor, selection))
  }
  function revertSelectedDraft() {
    updateSelectedEditor(revertWorkspaceEditorDraft)
  }
  function keepSelectedDraft() {
    updateSelectedEditor(keepWorkspaceEditorDraft)
  }
  function discardSelectedDraft() {
    withSelectedEditor(undefined, (repoPath, filePath) => {
      setEditorState(repoPath, filePath, null)
      return undefined
    })
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
    const result = reviewWorkspaceEditorPatch({ editor, diff })
    setEditorState(repoPath, filePath, result.nextState)
    return result.ok
  }
  function applyPendingPatch(repoPath: string, filePath: string) {
    const editor = getEditorState(repoPath, filePath)
    if (!editor) {
      return false
    }
    const result = applyWorkspaceEditorPendingPatch(editor)
    setEditorState(repoPath, filePath, result.nextState)
    return result.ok
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
    return withSelectedEditor(false, (repoPath, filePath, editor) => {
      const result = formatWorkspaceEditorDocument({ filePath, editor })
      setEditorState(repoPath, filePath, result.nextState)
      return result.ok
    })
  }
  function formatSelectedSelection() {
    return withSelectedEditor(false, (repoPath, filePath, editor) => {
      const result = formatWorkspaceEditorSelection({ filePath, editor })
      setEditorState(repoPath, filePath, result.nextState)
      return result.ok
    })
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
    return buildWorkspaceEditorWorkingSet({
      recentFiles,
      getEditorState,
      getPreview: input.getPreview,
    })
  }
  async function saveFile(repoPath: string, filePath: string): Promise<boolean> {
    const conversationId = input.getConversationId()
    const preview = input.getPreview(repoPath, filePath)
    const editor = getEditorState(repoPath, filePath)
    if (!conversationId || !repoPath || !filePath || !editor) {
      return false
    }
    return saveWorkspaceFile({
      conversationId,
      repoPath,
      filePath,
      preview,
      editor,
      getEditorState,
      setEditorState,
      reloadSelectedFile: input.reloadSelectedFile,
      refreshWorkspaceDiff: input.refreshWorkspaceDiff,
      setPreview: input.setPreview,
    })
  }
  async function saveSelectedFile(): Promise<boolean> {
    return saveFile(input.getSelectedRepoPath(), input.getSelectedFilePath())
  }
  return createWorkspaceFileEditorStoreApi({
    getSelectedEditorState: () => getEditorState(),
    getSelectedDraftLineDiff: (): WorkspaceFileLineDiffMarkers | null =>
      getSelectedWorkspaceDraftLineDiff({
        repoPath: input.getSelectedRepoPath(),
        filePath: input.getSelectedFilePath(),
        getEditorState,
      }),
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
  })
}
