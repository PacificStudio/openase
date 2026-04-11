import type {
  ProjectConversationWorkspaceFilePatch,
  ProjectConversationWorkspaceFilePreview,
  ProjectConversationWorkspaceMetadata,
  ProjectConversationWorkspaceTreeEntry,
} from '$lib/api/chat'
import type { WorkspaceFileSavePhase } from './project-conversation-workspace-file-drafts'

export type WorkspaceTab = {
  repoPath: string
  filePath: string
}

export type WorkspaceTabFileState = {
  preview: ProjectConversationWorkspaceFilePreview | null
  patch: ProjectConversationWorkspaceFilePatch | null
  loading: boolean
  error: string
}

export const EMPTY_TAB_FILE_STATE: WorkspaceTabFileState = {
  preview: null,
  patch: null,
  loading: false,
  error: '',
}

export function workspaceTabKey(tab: { repoPath: string; filePath: string }): string {
  return `${tab.repoPath}::${tab.filePath}`
}

/**
 * Pure transform: insert (or focus) a tab and return the next state slice.
 * Caller is responsible for assigning the returned values to $state holders.
 */
export function applyOpenTab(
  openTabs: WorkspaceTab[],
  repoPath: string,
  filePath: string,
): { openTabs: WorkspaceTab[]; activeTabKey: string; treeRepoPath: string } {
  const key = workspaceTabKey({ repoPath, filePath })
  const exists = openTabs.some((t) => workspaceTabKey(t) === key)
  return {
    openTabs: exists ? openTabs : [...openTabs, { repoPath, filePath }],
    activeTabKey: key,
    treeRepoPath: repoPath,
  }
}

/**
 * Pure transform: close a tab and figure out the new active tab. Returns null
 * if no such tab existed.
 */
export function applyCloseTab(
  openTabs: WorkspaceTab[],
  activeTabKey: string,
  repoPath: string,
  filePath: string,
): {
  openTabs: WorkspaceTab[]
  activeTabKey: string
  nextTreeRepo: string | null
} | null {
  const key = workspaceTabKey({ repoPath, filePath })
  const idx = openTabs.findIndex((t) => workspaceTabKey(t) === key)
  if (idx === -1) return null
  const nextTabs = openTabs.slice(0, idx).concat(openTabs.slice(idx + 1))
  let nextActive = activeTabKey
  let nextTreeRepo: string | null = null
  if (activeTabKey === key) {
    const fallback = nextTabs[idx] ?? nextTabs[idx - 1] ?? null
    nextActive = fallback ? workspaceTabKey(fallback) : ''
    if (fallback) nextTreeRepo = fallback.repoPath
  }
  return { openTabs: nextTabs, activeTabKey: nextActive, nextTreeRepo }
}

/**
 * Pure transform: merge a partial tab file-state patch into the map. Identity
 * for `preview` / `patch` is preserved when their values are deep-equal so
 * downstream identity-based effects don't refire on no-op refreshes.
 */
export function patchTabFileStateMap(
  current: Map<string, WorkspaceTabFileState>,
  key: string,
  patch: Partial<WorkspaceTabFileState>,
): Map<string, WorkspaceTabFileState> {
  const next = new Map(current)
  const existing = next.get(key) ?? EMPTY_TAB_FILE_STATE
  const merged: WorkspaceTabFileState = { ...existing, ...patch }
  if ('preview' in patch && patch.preview && areFilePreviewEqual(existing.preview, patch.preview)) {
    merged.preview = existing.preview
  }
  if ('patch' in patch && patch.patch && areFilePatchEqual(existing.patch, patch.patch)) {
    merged.patch = existing.patch
  }
  next.set(key, merged)
  return next
}

export function deleteTabFileStateMap(
  current: Map<string, WorkspaceTabFileState>,
  key: string,
): Map<string, WorkspaceTabFileState> {
  if (!current.has(key)) return current
  const next = new Map(current)
  next.delete(key)
  return next
}

export type WorkspaceFileEditorState = {
  baseSavedContent: string
  baseSavedRevision: string
  latestSavedContent: string
  latestSavedRevision: string
  draftContent: string
  dirty: boolean
  savePhase: WorkspaceFileSavePhase
  externalChange: boolean
  errorMessage: string
  encoding: 'utf-8'
  lineEnding: 'lf' | 'crlf'
  lastSavedAt: string
}

export function areTreeEntriesEqual(
  left: ProjectConversationWorkspaceTreeEntry[] | undefined,
  right: ProjectConversationWorkspaceTreeEntry[],
) {
  if (!left || left.length !== right.length) {
    return false
  }

  return left.every((entry, index) => {
    const next = right[index]
    return (
      next != null &&
      entry.path === next.path &&
      entry.name === next.name &&
      entry.kind === next.kind &&
      entry.sizeBytes === next.sizeBytes
    )
  })
}

export function areWorkspaceMetadataEqual(
  left: ProjectConversationWorkspaceMetadata | null,
  right: ProjectConversationWorkspaceMetadata,
) {
  if (
    !left ||
    left.conversationId !== right.conversationId ||
    left.available !== right.available ||
    left.workspacePath !== right.workspacePath ||
    left.repos.length !== right.repos.length
  ) {
    return false
  }

  return left.repos.every((repo, index) => {
    const next = right.repos[index]
    return (
      next != null &&
      repo.name === next.name &&
      repo.path === next.path &&
      repo.branch === next.branch &&
      repo.headCommit === next.headCommit &&
      repo.headSummary === next.headSummary &&
      repo.dirty === next.dirty &&
      repo.filesChanged === next.filesChanged &&
      repo.added === next.added &&
      repo.removed === next.removed
    )
  })
}

export function areFilePreviewEqual(
  left: ProjectConversationWorkspaceFilePreview | null,
  right: ProjectConversationWorkspaceFilePreview,
) {
  return (
    !!left &&
    left.conversationId === right.conversationId &&
    left.repoPath === right.repoPath &&
    left.path === right.path &&
    left.sizeBytes === right.sizeBytes &&
    left.mediaType === right.mediaType &&
    left.previewKind === right.previewKind &&
    left.truncated === right.truncated &&
    left.content === right.content &&
    left.revision === right.revision &&
    left.writable === right.writable &&
    left.readOnlyReason === right.readOnlyReason &&
    left.encoding === right.encoding &&
    left.lineEnding === right.lineEnding
  )
}

export function areFilePatchEqual(
  left: ProjectConversationWorkspaceFilePatch | null,
  right: ProjectConversationWorkspaceFilePatch,
) {
  return (
    !!left &&
    left.conversationId === right.conversationId &&
    left.repoPath === right.repoPath &&
    left.path === right.path &&
    left.status === right.status &&
    left.diffKind === right.diffKind &&
    left.truncated === right.truncated &&
    left.diff === right.diff
  )
}

export function createInitialEditorState(
  preview: ProjectConversationWorkspaceFilePreview,
): WorkspaceFileEditorState {
  return {
    baseSavedContent: preview.content,
    baseSavedRevision: preview.revision,
    latestSavedContent: preview.content,
    latestSavedRevision: preview.revision,
    draftContent: preview.content,
    dirty: false,
    savePhase: 'idle',
    externalChange: false,
    errorMessage: '',
    encoding: preview.encoding,
    lineEnding: preview.lineEnding,
    lastSavedAt: '',
  }
}

export {
  computeDraftLineDiff,
  type WorkspaceFileLineDiffMarkers,
} from './project-conversation-workspace-line-diff'
