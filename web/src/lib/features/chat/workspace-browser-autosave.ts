const WORKSPACE_AUTOSAVE_STORAGE_KEY = 'openase.project-conversation.workspace-autosave'

export function readWorkspaceAutosavePreference() {
  if (typeof window === 'undefined') return false
  return window.localStorage.getItem(WORKSPACE_AUTOSAVE_STORAGE_KEY) === 'true'
}

export function storeWorkspaceAutosavePreference(enabled: boolean) {
  if (typeof window === 'undefined') return
  window.localStorage.setItem(WORKSPACE_AUTOSAVE_STORAGE_KEY, enabled ? 'true' : 'false')
}
