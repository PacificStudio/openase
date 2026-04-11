export type EditorWrapMode = 'wrap' | 'nowrap'

export const EDITOR_WRAP_MODE_STORAGE_KEY = 'openase.editor.wrap-mode'
export const DEFAULT_EDITOR_WRAP_MODE: EditorWrapMode = 'wrap'

export function parseEditorWrapMode(raw: unknown): EditorWrapMode {
  return raw === 'nowrap' ? 'nowrap' : DEFAULT_EDITOR_WRAP_MODE
}

export function readEditorWrapMode(): EditorWrapMode {
  if (typeof window === 'undefined') {
    return DEFAULT_EDITOR_WRAP_MODE
  }

  try {
    return parseEditorWrapMode(window.localStorage.getItem(EDITOR_WRAP_MODE_STORAGE_KEY)?.trim())
  } catch {
    return DEFAULT_EDITOR_WRAP_MODE
  }
}

export function storeEditorWrapMode(mode: EditorWrapMode): void {
  if (typeof window === 'undefined') {
    return
  }

  try {
    window.localStorage.setItem(EDITOR_WRAP_MODE_STORAGE_KEY, mode)
  } catch {
    // Ignore localStorage failures.
  }
}
