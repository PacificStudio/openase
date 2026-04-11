import type { WorkspaceFileEditorState } from './project-conversation-workspace-browser-state-helpers'

export function workspaceFileStateLabel(
  editorState: WorkspaceFileEditorState | null | undefined,
): string {
  if (!editorState) {
    return ''
  }
  if (editorState.savePhase === 'saving') {
    return 'Saving...'
  }
  if (editorState.savePhase === 'conflict') {
    return 'Conflict'
  }
  if (editorState.externalChange) {
    return 'Changed in workspace'
  }
  if (editorState.dirty) {
    return 'Unsaved'
  }
  return 'Saved'
}

export function workspaceFileStateClass(
  editorState: WorkspaceFileEditorState | null | undefined,
): string {
  if (!editorState) {
    return 'bg-muted text-muted-foreground'
  }
  if (editorState.savePhase === 'saving') {
    return 'bg-sky-500/10 text-sky-700 dark:text-sky-300'
  }
  if (editorState.savePhase === 'conflict' || editorState.externalChange) {
    return 'bg-amber-500/10 text-amber-700 dark:text-amber-300'
  }
  if (editorState.dirty) {
    return 'bg-orange-500/10 text-orange-700 dark:text-orange-300'
  }
  return 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300'
}
