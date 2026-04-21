import { type ChatDiffPayload, type ProjectConversationWorkspaceFilePreview } from '$lib/api/chat'
import type {
  WorkspaceFileEditorState,
  WorkspaceFileLineDiffMarkers,
  WorkspaceRecentFile,
} from './project-conversation-workspace-browser-state-helpers'
import type { WorkspaceWorkingSetEntry } from './project-conversation-workspace-editor-helpers'

export function createWorkspaceFileEditorStoreApi<TSelection>(input: {
  getSelectedEditorState: () => WorkspaceFileEditorState | null
  getSelectedDraftLineDiff: () => WorkspaceFileLineDiffMarkers | null
  getEditorState: (repoPath?: string, filePath?: string) => WorkspaceFileEditorState | null
  reset: () => void
  syncFromPreview: (
    repoPath: string,
    filePath: string,
    nextPreview: ProjectConversationWorkspaceFilePreview,
  ) => void
  updateSelectedDraft: (nextDraftContent: string) => void
  updateSelectedSelection: (selection: TSelection) => void
  revertSelectedDraft: () => void
  keepSelectedDraft: () => void
  reloadSelectedSavedVersion: () => void
  reviewPatch: (repoPath: string, filePath: string, diff: ChatDiffPayload) => boolean
  applyPendingPatch: (repoPath: string, filePath: string) => boolean
  discardPendingPatch: (repoPath?: string, filePath?: string) => void
  formatSelectedDocument: () => boolean
  formatSelectedSelection: () => boolean
  renameFileState: (repoPath: string, fromPath: string, toPath: string) => void
  buildWorkingSet: (recentFiles: WorkspaceRecentFile[]) => WorkspaceWorkingSetEntry[]
  saveSelectedFile: () => Promise<boolean>
  saveFile: (repoPath: string, filePath: string) => Promise<boolean>
  discardSelectedDraft: () => void
  discardDraft: (repoPath: string, filePath: string) => void
}) {
  return {
    get selectedEditorState() {
      return input.getSelectedEditorState()
    },
    get selectedDraftLineDiff() {
      return input.getSelectedDraftLineDiff()
    },
    getEditorState: input.getEditorState,
    reset: input.reset,
    syncFromPreview: input.syncFromPreview,
    updateSelectedDraft: input.updateSelectedDraft,
    updateSelectedSelection: input.updateSelectedSelection,
    revertSelectedDraft: input.revertSelectedDraft,
    keepSelectedDraft: input.keepSelectedDraft,
    reloadSelectedSavedVersion: input.reloadSelectedSavedVersion,
    reviewPatch: input.reviewPatch,
    applyPendingPatch: input.applyPendingPatch,
    discardPendingPatch: input.discardPendingPatch,
    formatSelectedDocument: input.formatSelectedDocument,
    formatSelectedSelection: input.formatSelectedSelection,
    renameFileState: input.renameFileState,
    buildWorkingSet: input.buildWorkingSet,
    saveSelectedFile: input.saveSelectedFile,
    saveFile: input.saveFile,
    discardSelectedDraft: input.discardSelectedDraft,
    discardDraft: input.discardDraft,
  }
}
