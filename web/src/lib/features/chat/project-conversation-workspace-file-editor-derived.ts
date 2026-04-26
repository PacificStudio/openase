import type { ProjectConversationWorkspaceFilePreview } from '$lib/api/chat'
import {
  computeDraftLineDiff,
  type WorkspaceFileEditorState,
  type WorkspaceFileLineDiffMarkers,
  type WorkspaceRecentFile,
} from './project-conversation-workspace-browser-state-helpers'
import {
  buildWorkspaceWorkingSet,
  type WorkspaceWorkingSetEntry,
} from './project-conversation-workspace-editor-helpers'

type WorkspaceEditorGetter = (
  repoPath?: string,
  filePath?: string,
) => WorkspaceFileEditorState | null

type WorkspacePreviewGetter = (
  repoPath: string,
  filePath: string,
) => ProjectConversationWorkspaceFilePreview | null

export function buildWorkspaceEditorWorkingSet(input: {
  recentFiles: WorkspaceRecentFile[]
  getEditorState: WorkspaceEditorGetter
  getPreview: WorkspacePreviewGetter
}): WorkspaceWorkingSetEntry[] {
  return buildWorkspaceWorkingSet(
    input.recentFiles
      .map((item) => {
        const editor = input.getEditorState(item.repoPath, item.filePath)
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

export function getSelectedWorkspaceDraftLineDiff(input: {
  repoPath: string
  filePath: string
  getEditorState: WorkspaceEditorGetter
}): WorkspaceFileLineDiffMarkers | null {
  if (!input.filePath) {
    return null
  }
  const editor = input.getEditorState(input.repoPath, input.filePath)
  if (!editor) {
    return null
  }
  return computeDraftLineDiff(editor.latestSavedContent, editor.draftContent)
}
