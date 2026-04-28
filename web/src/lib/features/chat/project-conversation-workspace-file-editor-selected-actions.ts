import { type WorkspaceSelectionInput } from './project-conversation-workspace-editor-helpers'
import {
  computeDraftLineDiff,
  type WorkspaceFileEditorState,
  type WorkspaceFileLineDiffMarkers,
} from './project-conversation-workspace-browser-state-helpers'
import {
  formatWorkspaceEditorDocument,
  formatWorkspaceEditorSelection,
  keepWorkspaceEditorDraft,
  revertWorkspaceEditorDraft,
  updateWorkspaceEditorDraft,
  updateWorkspaceEditorSelection,
} from './project-conversation-workspace-file-editor-state-transforms'

type SelectedWorkspaceFileEditorContext = {
  repoPath: string
  filePath: string
  editor: WorkspaceFileEditorState
}

export function createWorkspaceFileEditorSelectedActions(input: {
  getSelectedRepoPath: () => string
  getSelectedFilePath: () => string
  getEditorState: (repoPath?: string, filePath?: string) => WorkspaceFileEditorState | null
  setEditorState: (
    repoPath: string,
    filePath: string,
    nextState: WorkspaceFileEditorState | null,
  ) => void
}) {
  function getSelectedEditorContext(): SelectedWorkspaceFileEditorContext | null {
    const repoPath = input.getSelectedRepoPath()
    const filePath = input.getSelectedFilePath()
    if (!repoPath || !filePath) return null
    const editor = input.getEditorState(repoPath, filePath)
    return editor ? { repoPath, filePath, editor } : null
  }

  function updateSelectedEditorState(
    update: (context: SelectedWorkspaceFileEditorContext) => WorkspaceFileEditorState | null,
  ) {
    const current = getSelectedEditorContext()
    if (!current) return null
    input.setEditorState(current.repoPath, current.filePath, update(current))
    return current
  }

  function getSelectedDraftLineDiff(): WorkspaceFileLineDiffMarkers | null {
    const current = getSelectedEditorContext()
    return current
      ? computeDraftLineDiff(current.editor.latestSavedContent, current.editor.draftContent)
      : null
  }

  function formatSelectedDocument() {
    const current = getSelectedEditorContext()
    if (!current) return false
    const result = formatWorkspaceEditorDocument({
      filePath: current.filePath,
      editor: current.editor,
    })
    input.setEditorState(current.repoPath, current.filePath, result.nextState)
    return result.ok
  }

  function formatSelectedSelection() {
    const current = getSelectedEditorContext()
    if (!current) return false
    const result = formatWorkspaceEditorSelection({
      filePath: current.filePath,
      editor: current.editor,
    })
    input.setEditorState(current.repoPath, current.filePath, result.nextState)
    return result.ok
  }

  return {
    getSelectedDraftLineDiff,
    updateSelectedDraft(nextDraftContent: string) {
      updateSelectedEditorState(({ editor }) =>
        updateWorkspaceEditorDraft(editor, nextDraftContent),
      )
    },
    updateSelectedSelection(selection: WorkspaceSelectionInput | null) {
      updateSelectedEditorState(({ editor }) => updateWorkspaceEditorSelection(editor, selection))
    },
    revertSelectedDraft() {
      updateSelectedEditorState(({ editor }) => revertWorkspaceEditorDraft(editor))
    },
    keepSelectedDraft() {
      updateSelectedEditorState(({ editor }) => keepWorkspaceEditorDraft(editor))
    },
    discardSelectedDraft: () => {
      const current = getSelectedEditorContext()
      if (!current) return
      input.setEditorState(current.repoPath, current.filePath, null)
    },
    reloadSelectedSavedVersion() {
      updateSelectedEditorState(({ editor }) => revertWorkspaceEditorDraft(editor))
    },
    formatSelectedDocument,
    formatSelectedSelection,
  }
}
