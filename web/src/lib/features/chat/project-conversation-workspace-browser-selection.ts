import type { ProjectConversationWorkspaceDiff } from '$lib/api/chat'
import {
  buildWorkspaceFocusContext,
  workspaceSelectedChangedFiles,
} from './project-conversation-workspace-browser-file-ops'
import type {
  WorkspaceFileEditorState,
  WorkspaceRecentFile,
  WorkspaceTab,
  WorkspaceTabFileState,
} from './project-conversation-workspace-browser-state-helpers'
import type { WorkspaceWorkingSetEntry } from './project-conversation-workspace-editor-helpers'

export function createWorkspaceBrowserSelection(input: {
  getActiveTab: () => WorkspaceTab | null
  getActiveTabFileState: () => WorkspaceTabFileState
  getSelectedEditorState: () => WorkspaceFileEditorState | null
  getRecentFiles: () => WorkspaceRecentFile[]
  getTreeRepoPath: () => string
  getWorkspaceDiff: () => ProjectConversationWorkspaceDiff | null
  buildWorkingSet: (recentFiles: WorkspaceRecentFile[]) => WorkspaceWorkingSetEntry[]
}) {
  const getSelectedRepoPath = () => input.getTreeRepoPath()
  const getSelectedFilePath = () => input.getActiveTab()?.filePath ?? ''

  return {
    getSelectedRepoPath,
    getSelectedFilePath,
    getPreview: () => input.getActiveTabFileState().preview,
    getPatch: () => input.getActiveTabFileState().patch,
    getFileLoading: () => input.getActiveTabFileState().loading,
    getFileError: () => input.getActiveTabFileState().error,
    getSelectedChangedFiles: () =>
      workspaceSelectedChangedFiles({
        repoPath: getSelectedRepoPath(),
        activeFilePath: getSelectedFilePath(),
        workspaceDiff: input.getWorkspaceDiff(),
      }),
    getSelectedFocusContext: () =>
      buildWorkspaceFocusContext({
        selectedEditorState: input.getSelectedEditorState(),
        hasActiveTab: input.getActiveTab() != null,
        recentFiles: input.getRecentFiles(),
        buildWorkingSet: input.buildWorkingSet,
      }),
  }
}
