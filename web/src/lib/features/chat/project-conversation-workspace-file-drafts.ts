export type WorkspaceFileSavePhase = 'idle' | 'saving' | 'conflict' | 'error'

export type PersistedWorkspaceFileDraft = {
  draftContent: string
  baseSavedContent: string
  baseSavedRevision: string
  encoding: 'utf-8'
  lineEnding: 'lf' | 'crlf'
  updatedAt: string
}

const STORAGE_KEY = 'openase.project-conversation.workspace-file-drafts'

type DraftStorageShape = Record<string, PersistedWorkspaceFileDraft>

export function workspaceFileDraftStorageKey(input: {
  conversationId: string
  repoPath: string
  filePath: string
  refCacheKey?: string
}) {
  return [
    input.conversationId.trim(),
    input.repoPath.trim(),
    input.refCacheKey?.trim() ?? '',
    input.filePath.trim(),
  ].join('::')
}

function readDraftStorage(): DraftStorageShape {
  if (typeof window === 'undefined') {
    return {}
  }
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)?.trim() ?? ''
    if (!raw) {
      return {}
    }
    const parsed = JSON.parse(raw) as DraftStorageShape
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function writeDraftStorage(value: DraftStorageShape) {
  if (typeof window === 'undefined') {
    return
  }
  try {
    if (Object.keys(value).length === 0) {
      window.localStorage.removeItem(STORAGE_KEY)
      return
    }
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(value))
  } catch {
    // Ignore localStorage failures.
  }
}

export function loadPersistedWorkspaceFileDraft(key: string): PersistedWorkspaceFileDraft | null {
  const storage = readDraftStorage()
  return storage[key] ?? null
}

export function savePersistedWorkspaceFileDraft(key: string, value: PersistedWorkspaceFileDraft) {
  const storage = readDraftStorage()
  storage[key] = value
  writeDraftStorage(storage)
}

export function deletePersistedWorkspaceFileDraft(key: string) {
  const storage = readDraftStorage()
  if (!(key in storage)) {
    return
  }
  delete storage[key]
  writeDraftStorage(storage)
}

import { chatT } from './i18n'

export function workspaceFileReadOnlyMessage(reason: string) {
  switch (reason) {
    case 'binary_file':
      return chatT('chat.binaryFileReadOnly')
    case 'file_too_large':
      return chatT('chat.fileTooLargeReadOnly')
    case 'unsupported_encoding':
      return chatT('chat.unsupportedEncodingReadOnly')
    default:
      return chatT('chat.genericReadOnlyFile')
  }
}
