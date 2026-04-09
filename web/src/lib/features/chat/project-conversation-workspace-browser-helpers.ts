import type {
  ProjectConversationWorkspaceFileStatus,
  ProjectConversationWorkspaceRepoMetadata,
} from '$lib/api/chat'

export function repoDirtyLabel(repo: ProjectConversationWorkspaceRepoMetadata) {
  return repo.dirty
    ? `${repo.filesChanged} file${repo.filesChanged === 1 ? '' : 's'} changed`
    : 'Clean'
}

export function formatTotals(added: number, removed: number) {
  return `+${added} -${removed}`
}

export function directorySegments(path: string) {
  return path.split('/').filter((segment) => segment.length > 0)
}

export function joinSegments(segments: string[]) {
  return segments.join('/')
}

export function statusLabel(status: ProjectConversationWorkspaceFileStatus) {
  switch (status) {
    case 'added':
      return 'A'
    case 'deleted':
      return 'D'
    case 'renamed':
      return 'R'
    case 'untracked':
      return 'U'
    default:
      return 'M'
  }
}

export function statusClass(status: ProjectConversationWorkspaceFileStatus) {
  switch (status) {
    case 'added':
    case 'untracked':
      return 'text-emerald-600'
    case 'deleted':
      return 'text-rose-600'
    case 'renamed':
      return 'text-amber-600'
    default:
      return 'text-sky-600'
  }
}
