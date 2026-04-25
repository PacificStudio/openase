import type { ProjectConversationWorkspaceFilePreview } from '$lib/api/chat'
import { buildWorkspaceWorkingSet } from './project-conversation-workspace-editor-helpers'
import type {
  WorkspaceFileEditorState,
  WorkspaceRecentFile,
} from './project-conversation-workspace-browser-state-helpers'

export function shouldAutosaveWorkspaceFileEditor(args: {
  autosaveEnabled: boolean | undefined
  editorState: WorkspaceFileEditorState
  preview: ProjectConversationWorkspaceFilePreview | null
}) {
  return Boolean(
    args.autosaveEnabled &&
    args.editorState.dirty &&
    args.editorState.savePhase !== 'saving' &&
    args.editorState.savePhase !== 'conflict' &&
    !args.editorState.externalChange &&
    args.preview?.writable === true,
  )
}

export function buildWorkspaceFileEditorWorkingSet(args: {
  recentFiles: WorkspaceRecentFile[]
  getEditorState: (repoPath: string, filePath: string) => WorkspaceFileEditorState | null
  getPreview: (repoPath: string, filePath: string) => ProjectConversationWorkspaceFilePreview | null
}) {
  return buildWorkspaceWorkingSet(
    args.recentFiles.flatMap((item) => {
      const editor = args.getEditorState(item.repoPath, item.filePath)
      const preview = args.getPreview(item.repoPath, item.filePath)
      const content = editor?.draftContent ?? preview?.content ?? ''
      if (!content) {
        return []
      }
      return [
        {
          filePath: item.filePath,
          content,
          dirty: editor?.dirty ?? false,
        },
      ]
    }),
  )
}
