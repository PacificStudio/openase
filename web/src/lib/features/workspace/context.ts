import { getContext, setContext } from 'svelte'
import type { createWorkspaceController } from './controller.svelte'

const workspaceContextKey = Symbol('workspace-controller')

export type WorkspaceController = ReturnType<typeof createWorkspaceController>

export function setWorkspaceContext(workspace: WorkspaceController) {
  setContext(workspaceContextKey, workspace)
  return workspace
}

export function getWorkspaceContext() {
  const workspace = getContext<WorkspaceController | undefined>(workspaceContextKey)
  if (!workspace) {
    throw new Error('Workspace controller context is not available in this route.')
  }

  return workspace
}
