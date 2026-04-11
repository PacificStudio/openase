import type {
  ProjectConversationWorkspaceFilePatch,
  ProjectConversationWorkspaceFilePreview,
  ProjectConversationWorkspaceMetadata,
  ProjectConversationWorkspaceTreeEntry,
} from '$lib/api/chat'
import type {
  WorkspaceFileSavePhase,
  WorkspaceFileViewMode,
} from './project-conversation-workspace-file-drafts'

export type WorkspaceFileEditorState = {
  baseSavedContent: string
  baseSavedRevision: string
  latestSavedContent: string
  latestSavedRevision: string
  draftContent: string
  dirty: boolean
  viewMode: WorkspaceFileViewMode
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
    viewMode: 'preview',
    savePhase: 'idle',
    externalChange: false,
    errorMessage: '',
    encoding: preview.encoding,
    lineEnding: preview.lineEnding,
    lastSavedAt: '',
  }
}

export function buildWholeFileDiff(filePath: string, savedContent: string, draftContent: string) {
  if (savedContent === draftContent) {
    return ''
  }
  const oldLines = savedContent.split('\n')
  const newLines = draftContent.split('\n')
  const oldCount = savedContent === '' ? 0 : oldLines.length
  const newCount = draftContent === '' ? 0 : newLines.length

  return [
    `--- saved/${filePath}`,
    `+++ draft/${filePath}`,
    `@@ -1,${oldCount} +1,${newCount} @@`,
    ...oldLines.map((line) => `-${line}`),
    ...newLines.map((line) => `+${line}`),
  ].join('\n')
}
