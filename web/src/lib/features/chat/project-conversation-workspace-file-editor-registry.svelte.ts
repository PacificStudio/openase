import {
  deletePersistedWorkspaceFileDraft,
  loadPersistedWorkspaceFileDraft,
  savePersistedWorkspaceFileDraft,
  workspaceFileDraftStorageKey,
} from './project-conversation-workspace-file-drafts'
import type { WorkspaceFileEditorState } from './project-conversation-workspace-browser-state-helpers'

const WORKSPACE_AUTOSAVE_DELAY_MS = 1000

export function createWorkspaceFileEditorRegistry(input: {
  getConversationId: () => string
  getRepoRefCacheKey?: (repoPath: string) => string
  shouldAutosave: (
    repoPath: string,
    filePath: string,
    editorState: WorkspaceFileEditorState,
  ) => boolean
  onAutosave: (repoPath: string, filePath: string) => void
}) {
  let editorStates = $state<Map<string, WorkspaceFileEditorState>>(new Map())
  const autosaveTimers = new Map<string, ReturnType<typeof setTimeout>>()

  function fileStorageKey(repoPath: string, filePath: string) {
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

  function scheduleAutosave(
    repoPath: string,
    filePath: string,
    editorState: WorkspaceFileEditorState,
  ) {
    const key = fileStorageKey(repoPath, filePath)
    cancelAutosave(key)
    if (!input.shouldAutosave(repoPath, filePath, editorState)) {
      return
    }
    autosaveTimers.set(
      key,
      setTimeout(() => {
        autosaveTimers.delete(key)
        input.onAutosave(repoPath, filePath)
      }, WORKSPACE_AUTOSAVE_DELAY_MS),
    )
  }

  function getEditorState(repoPath?: string, filePath?: string) {
    if (!repoPath || !filePath) {
      return null
    }
    return editorStates.get(fileStorageKey(repoPath, filePath)) ?? null
  }

  function setEditorState(
    repoPath: string,
    filePath: string,
    nextState: WorkspaceFileEditorState | null,
  ) {
    const key = fileStorageKey(repoPath, filePath)
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

  function renameFileState(repoPath: string, fromPath: string, toPath: string) {
    const fromKey = fileStorageKey(repoPath, fromPath)
    const toKey = fileStorageKey(repoPath, toPath)
    const editor = editorStates.get(fromKey)
    const persisted = loadPersistedWorkspaceFileDraft(fromKey)
    const nextStates = new Map(editorStates)
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

  function reset() {
    for (const key of autosaveTimers.keys()) {
      cancelAutosave(key)
    }
    editorStates = new Map()
  }

  return {
    fileStorageKey,
    getEditorState,
    setEditorState,
    renameFileState,
    readPersistedDraft: (repoPath: string, filePath: string) =>
      loadPersistedWorkspaceFileDraft(fileStorageKey(repoPath, filePath)),
    reset,
  }
}
