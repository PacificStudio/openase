import { type ChatDiffPayload, type ProjectConversationWorkspaceFilePreview } from '$lib/api/chat'
import { saveWorkspaceFile } from './workspace-file-editor-save'
import {
  buildWorkspaceWorkingSet,
  type WorkspaceSelectionInput,
} from './project-conversation-workspace-editor-helpers'
import {
  computeDraftLineDiff,
  type WorkspaceFileLineDiffMarkers,
  type WorkspaceRecentFile,
} from './project-conversation-workspace-browser-state-helpers'
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
import { createWorkspaceFileEditorRegistry } from './project-conversation-workspace-file-editor-registry.svelte'
import { createWorkspaceFileEditorStoreApi } from './project-conversation-workspace-file-editor-store-api'

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
  const registry = createWorkspaceFileEditorRegistry({
    getConversationId: input.getConversationId,
    getRepoRefCacheKey: input.getRepoRefCacheKey,
    shouldAutosave: (repoPath, filePath, editorState) =>
      input.getAutosaveEnabled?.() === true &&
      editorState.dirty &&
      editorState.savePhase !== 'saving' &&
      editorState.savePhase !== 'conflict' &&
      !editorState.externalChange &&
      input.getPreview(repoPath, filePath)?.writable === true,
    onAutosave: (repoPath, filePath) => {
      void saveFile(repoPath, filePath)
    },
  })

  function getSelectedEditorContext() {
    const repoPath = input.getSelectedRepoPath()
    const filePath = input.getSelectedFilePath()
    const editor = registry.getEditorState(repoPath, filePath)
    if (!editor || !repoPath || !filePath) {
      return null
    }
    return { repoPath, filePath, editor }
  }

  function syncFromPreview(
    repoPath: string,
    filePath: string,
    nextPreview: ProjectConversationWorkspaceFilePreview,
  ) {
    registry.setEditorState(
      repoPath,
      filePath,
      syncWorkspaceEditorStateFromPreview({
        existing: registry.getEditorState(repoPath, filePath),
        persisted: registry.readPersistedDraft(repoPath, filePath),
        nextPreview,
      }),
    )
  }

  function reset() {
    registry.reset()
  }

  function updateSelectedDraft(nextDraftContent: string) {
    const selected = getSelectedEditorContext()
    if (!selected) {
      return
    }
    registry.setEditorState(
      selected.repoPath,
      selected.filePath,
      updateWorkspaceEditorDraft(selected.editor, nextDraftContent),
    )
  }

  function updateSelectedSelection(selection: WorkspaceSelectionInput | null) {
    const selected = getSelectedEditorContext()
    if (!selected) {
      return
    }
    registry.setEditorState(
      selected.repoPath,
      selected.filePath,
      updateWorkspaceEditorSelection(selected.editor, selection),
    )
  }

  function revertSelectedDraft() {
    const selected = getSelectedEditorContext()
    if (!selected) {
      return
    }
    registry.setEditorState(
      selected.repoPath,
      selected.filePath,
      revertWorkspaceEditorDraft(selected.editor),
    )
  }

  function keepSelectedDraft() {
    const selected = getSelectedEditorContext()
    if (!selected) {
      return
    }
    registry.setEditorState(
      selected.repoPath,
      selected.filePath,
      keepWorkspaceEditorDraft(selected.editor),
    )
  }

  function discardSelectedDraft() {
    const repoPath = input.getSelectedRepoPath()
    const filePath = input.getSelectedFilePath()
    if (!repoPath || !filePath) return
    registry.setEditorState(repoPath, filePath, null)
  }

  function discardDraft(repoPath: string, filePath: string) {
    if (!repoPath || !filePath) return
    registry.setEditorState(repoPath, filePath, null)
  }

  function reloadSelectedSavedVersion() {
    revertSelectedDraft()
  }

  function reviewPatch(repoPath: string, filePath: string, diff: ChatDiffPayload) {
    const editor = registry.getEditorState(repoPath, filePath)
    if (!editor) {
      return false
    }
    const result = reviewWorkspaceEditorPatch({ editor, diff })
    registry.setEditorState(repoPath, filePath, result.nextState)
    return result.ok
  }

  function applyPendingPatch(repoPath: string, filePath: string) {
    const editor = registry.getEditorState(repoPath, filePath)
    if (!editor) {
      return false
    }
    const result = applyWorkspaceEditorPendingPatch(editor)
    registry.setEditorState(repoPath, filePath, result.nextState)
    return result.ok
  }

  function discardPendingPatch(
    repoPath = input.getSelectedRepoPath(),
    filePath = input.getSelectedFilePath(),
  ) {
    const editor = registry.getEditorState(repoPath, filePath)
    if (!editor || !repoPath || !filePath) {
      return
    }
    registry.setEditorState(repoPath, filePath, {
      ...editor,
      pendingPatch: null,
      errorMessage: '',
    })
  }

  function formatSelectedDocument() {
    const selected = getSelectedEditorContext()
    if (!selected) {
      return false
    }
    const result = formatWorkspaceEditorDocument({
      filePath: selected.filePath,
      editor: selected.editor,
    })
    registry.setEditorState(selected.repoPath, selected.filePath, result.nextState)
    return result.ok
  }

  function formatSelectedSelection() {
    const selected = getSelectedEditorContext()
    if (!selected) {
      return false
    }
    const result = formatWorkspaceEditorSelection({
      filePath: selected.filePath,
      editor: selected.editor,
    })
    registry.setEditorState(selected.repoPath, selected.filePath, result.nextState)
    return result.ok
  }

  function renameFileState(repoPath: string, fromPath: string, toPath: string) {
    registry.renameFileState(repoPath, fromPath, toPath)
  }

  function buildWorkingSet(recentFiles: WorkspaceRecentFile[]) {
    return buildWorkspaceWorkingSet(
      recentFiles
        .map((item) => {
          const editor = registry.getEditorState(item.repoPath, item.filePath)
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
    const editor = registry.getEditorState(repoPath, filePath)
    if (!conversationId || !repoPath || !filePath || !editor) {
      return false
    }
    return saveWorkspaceFile({
      conversationId,
      repoPath,
      filePath,
      preview,
      editor,
      getEditorState: registry.getEditorState,
      setEditorState: registry.setEditorState,
      reloadSelectedFile: input.reloadSelectedFile,
      refreshWorkspaceDiff: input.refreshWorkspaceDiff,
      setPreview: input.setPreview,
    })
  }
  async function saveSelectedFile(): Promise<boolean> {
    return saveFile(input.getSelectedRepoPath(), input.getSelectedFilePath())
  }
  return createWorkspaceFileEditorStoreApi({
    getSelectedEditorState: () =>
      registry.getEditorState(input.getSelectedRepoPath(), input.getSelectedFilePath()),
    getSelectedDraftLineDiff: (): WorkspaceFileLineDiffMarkers | null => {
      const selected = getSelectedEditorContext()
      if (!selected) return null
      return computeDraftLineDiff(selected.editor.latestSavedContent, selected.editor.draftContent)
    },
    getEditorState: registry.getEditorState,
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
