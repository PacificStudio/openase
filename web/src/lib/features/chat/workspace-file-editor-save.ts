import { ApiError } from '$lib/api/client'
import {
  saveProjectConversationWorkspaceFile,
  type ProjectConversationWorkspaceFilePreview,
} from '$lib/api/chat'
import type { WorkspaceFileEditorState } from './project-conversation-workspace-browser-state-helpers'
import { chatT } from './i18n'

export async function saveWorkspaceFile(input: {
  conversationId: string
  repoPath: string
  filePath: string
  preview: ProjectConversationWorkspaceFilePreview | null
  editor: WorkspaceFileEditorState
  getEditorState: (repoPath: string, filePath: string) => WorkspaceFileEditorState | null
  setEditorState: (
    repoPath: string,
    filePath: string,
    nextState: WorkspaceFileEditorState | null,
  ) => void
  reloadSelectedFile: (repoPath: string, filePath: string) => Promise<void>
  refreshWorkspaceDiff?: () => Promise<void>
  setPreview: (
    repoPath: string,
    filePath: string,
    preview: ProjectConversationWorkspaceFilePreview | null,
  ) => void
}) {
  const { conversationId, repoPath, filePath, preview, editor } = input

  if (!conversationId || !repoPath || !filePath || !preview?.writable) {
    return false
  }
  if (!editor.dirty || editor.savePhase === 'saving') {
    return !editor.dirty
  }

  input.setEditorState(repoPath, filePath, {
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
    const nextEditor = input.getEditorState(repoPath, filePath)
    if (!nextEditor) {
      return false
    }

    input.setEditorState(repoPath, filePath, {
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
      const refreshedEditor = input.getEditorState(repoPath, filePath)
      if (refreshedEditor) {
        input.setEditorState(repoPath, filePath, {
          ...refreshedEditor,
          errorMessage: 'Saved, but the workspace summary could not be refreshed.',
        })
      }
    }
    return true
  } catch (error) {
    const latestEditor = input.getEditorState(repoPath, filePath)
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
        input.setEditorState(repoPath, filePath, {
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
        input.setEditorState(repoPath, filePath, {
          ...latestEditor,
          savePhase: 'conflict',
          externalChange: true,
          errorMessage: error.message,
        })
      }
      return false
    }

    input.setEditorState(repoPath, filePath, {
      ...latestEditor,
      savePhase: 'error',
      errorMessage: error instanceof Error ? error.message : chatT('chat.workspace.errors.save'),
    })
    return false
  }
}
