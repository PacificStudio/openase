import type { WorkspaceSelectionInput } from './project-conversation-workspace-editor-helpers'
import {
  type WorkspaceFileEditorState,
  type WorkspaceFileLineDiffMarkers,
  computeDraftLineDiff,
} from './project-conversation-workspace-browser-state-helpers'

function readSelectedEditor(input: {
  getSelectedRepoPath: () => string
  getSelectedFilePath: () => string
  getEditorState: (repoPath?: string, filePath?: string) => WorkspaceFileEditorState | null
}) {
  const repoPath = input.getSelectedRepoPath()
  const filePath = input.getSelectedFilePath()
  const editor = input.getEditorState(repoPath, filePath)
  if (!repoPath || !filePath || !editor) {
    return null
  }
  return { repoPath, filePath, editor }
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
  updateDraft: (editor: WorkspaceFileEditorState, nextDraftContent: string) => WorkspaceFileEditorState
  updateSelection: (
    editor: WorkspaceFileEditorState,
    selection: WorkspaceSelectionInput | null,
  ) => WorkspaceFileEditorState
  revertDraft: (editor: WorkspaceFileEditorState) => WorkspaceFileEditorState
  keepDraft: (editor: WorkspaceFileEditorState) => WorkspaceFileEditorState
  formatDocument: (args: {
    filePath: string
    editor: WorkspaceFileEditorState
  }) => { ok: boolean; nextState: WorkspaceFileEditorState }
  formatSelection: (args: {
    filePath: string
    editor: WorkspaceFileEditorState
  }) => { ok: boolean; nextState: WorkspaceFileEditorState }
  saveFile: (repoPath: string, filePath: string) => Promise<boolean>
}) {
  return {
    getSelectedEditorState: () => readSelectedEditor(input)?.editor ?? null,
    getSelectedDraftLineDiff: (): WorkspaceFileLineDiffMarkers | null => {
      const selected = readSelectedEditor(input)
      if (!selected) {
        return null
      }
      return computeDraftLineDiff(selected.editor.latestSavedContent, selected.editor.draftContent)
    },
    updateSelectedDraft: (nextDraftContent: string) => {
      const selected = readSelectedEditor(input)
      if (!selected) {
        return
      }
      input.setEditorState(
        selected.repoPath,
        selected.filePath,
        input.updateDraft(selected.editor, nextDraftContent),
      )
    },
    updateSelectedSelection: (selection: WorkspaceSelectionInput | null) => {
      const selected = readSelectedEditor(input)
      if (!selected) {
        return
      }
      input.setEditorState(
        selected.repoPath,
        selected.filePath,
        input.updateSelection(selected.editor, selection),
      )
    },
    revertSelectedDraft: () => {
      const selected = readSelectedEditor(input)
      if (!selected) {
        return
      }
      input.setEditorState(selected.repoPath, selected.filePath, input.revertDraft(selected.editor))
    },
    keepSelectedDraft: () => {
      const selected = readSelectedEditor(input)
      if (!selected) {
        return
      }
      input.setEditorState(selected.repoPath, selected.filePath, input.keepDraft(selected.editor))
    },
    discardSelectedDraft: () => {
      const selected = readSelectedEditor(input)
      if (!selected) {
        return
      }
      input.setEditorState(selected.repoPath, selected.filePath, null)
    },
    formatSelectedDocument: () => {
      const selected = readSelectedEditor(input)
      if (!selected) {
        return false
      }
      const result = input.formatDocument({
        filePath: selected.filePath,
        editor: selected.editor,
      })
      input.setEditorState(selected.repoPath, selected.filePath, result.nextState)
      return result.ok
    },
    formatSelectedSelection: () => {
      const selected = readSelectedEditor(input)
      if (!selected) {
        return false
      }
      const result = input.formatSelection({
        filePath: selected.filePath,
        editor: selected.editor,
      })
      input.setEditorState(selected.repoPath, selected.filePath, result.nextState)
      return result.ok
    },
    reloadSelectedSavedVersion: () => {
      const selected = readSelectedEditor(input)
      if (!selected) {
        return
      }
      input.setEditorState(selected.repoPath, selected.filePath, input.revertDraft(selected.editor))
    },
    saveSelectedFile: () => {
      const selected = readSelectedEditor(input)
      if (!selected) {
        return Promise.resolve(false)
      }
      return input.saveFile(selected.repoPath, selected.filePath)
    },
  }
}
