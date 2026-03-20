import type { WorkspaceStartOptions } from './controller.svelte'

export function readWorkspaceRouteSelection(searchParams: URLSearchParams): WorkspaceStartOptions {
  return {
    preferredOrgId: readNonEmptyQuery(searchParams, 'org'),
    preferredProjectId: readNonEmptyQuery(searchParams, 'project'),
    preferredWorkflowId: readNonEmptyQuery(searchParams, 'workflow'),
  }
}

function readNonEmptyQuery(searchParams: URLSearchParams, key: string) {
  const value = searchParams.get(key)?.trim()
  return value ? value : undefined
}
