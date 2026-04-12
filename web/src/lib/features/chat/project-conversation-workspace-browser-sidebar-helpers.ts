import type {
  ProjectConversationWorkspaceFileStatus,
  ProjectConversationWorkspaceTreeEntry,
} from '$lib/api/chat'
import type { TreeMenuItem } from './project-conversation-workspace-browser-tree-menu.svelte'
import type { TreeMenuTarget } from './project-conversation-workspace-browser-tree.svelte'

export function dirtyFileColorClass(status: ProjectConversationWorkspaceFileStatus): string {
  switch (status) {
    case 'added':
    case 'untracked':
      return 'text-emerald-600 dark:text-emerald-400'
    case 'deleted':
      return 'text-rose-600 dark:text-rose-400'
    default:
      return 'text-amber-600 dark:text-amber-400'
  }
}

export function parentOf(path: string): string {
  const idx = path.lastIndexOf('/')
  return idx === -1 ? '' : path.slice(0, idx)
}

export function filenameFromPath(path: string): string {
  return path.split('/').pop() ?? ''
}

type TreeMenuActions = {
  onStartCreate: (kind: 'file' | 'folder', parentPath?: string) => void
  onStartRename: (path: string) => void
  onDeleteEntry?: (path: string) => void
  onCopyAbsolutePath?: (path: string) => void
  onCopyRelativePath?: (path: string) => void
}

export function buildTreeMenuItems(
  entry: TreeMenuTarget,
  actions: TreeMenuActions,
): TreeMenuItem[] {
  const items: TreeMenuItem[] = []
  if (entry.kind === 'directory') {
    items.push({
      kind: 'item',
      label: 'New File',
      onSelect: () => actions.onStartCreate('file', entry.path),
    })
    items.push({
      kind: 'item',
      label: 'New Folder',
      onSelect: () => actions.onStartCreate('folder', entry.path),
    })
    items.push({ kind: 'separator' })
  }
  items.push({
    kind: 'item',
    label: 'Rename…',
    onSelect: () => actions.onStartRename(entry.path),
  })
  items.push({
    kind: 'item',
    label: 'Delete',
    danger: true,
    onSelect: () => actions.onDeleteEntry?.(entry.path),
  })
  items.push({ kind: 'separator' })
  items.push({
    kind: 'item',
    label: 'Copy Path',
    onSelect: () => actions.onCopyAbsolutePath?.(entry.path),
  })
  items.push({
    kind: 'item',
    label: 'Copy Relative Path',
    onSelect: () => actions.onCopyRelativePath?.(entry.path),
  })
  return items
}

export function buildDirtyFileStatusMap(
  dirtyFiles: Array<{ path: string; status: ProjectConversationWorkspaceFileStatus }>,
) {
  return new Map<string, ProjectConversationWorkspaceFileStatus>(
    dirtyFiles.map((file) => [file.path, file.status]),
  )
}

export function buildDirtyParentDirs(
  dirtyFiles: Array<Pick<ProjectConversationWorkspaceTreeEntry, 'path'>>,
) {
  const dirs = new Set<string>()
  for (const file of dirtyFiles) {
    const parts = file.path.split('/')
    for (let i = 1; i < parts.length; i++) {
      dirs.add(parts.slice(0, i).join('/'))
    }
  }
  return dirs
}
