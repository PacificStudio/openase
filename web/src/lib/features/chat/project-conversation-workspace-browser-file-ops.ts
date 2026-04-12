import {
  createProjectConversationWorkspaceFile,
  deleteProjectConversationWorkspaceFile,
  renameProjectConversationWorkspaceFile,
  searchProjectConversationWorkspacePaths,
  type ChatDiffPayload,
  type ProjectConversationWorkspaceDiff,
  type ProjectConversationWorkspaceSearchResult,
} from '$lib/api/chat'
import type {
  WorkspaceFileEditorState,
  WorkspaceFocusContext,
  WorkspaceRecentFile,
  WorkspaceTab,
  WorkspaceTabFileState,
} from './project-conversation-workspace-browser-state-helpers'
import { workspaceTabKey } from './project-conversation-workspace-browser-state-helpers'

export function workspaceSelectedChangedFiles(input: {
  repoPath: string
  activeFilePath: string
  workspaceDiff: ProjectConversationWorkspaceDiff | null
}) {
  const repoDiff = input.workspaceDiff?.repos.find((repo) => repo.path === input.repoPath)
  if (!repoDiff) {
    return []
  }
  return repoDiff.files.filter((file) => file.path !== input.activeFilePath)
}

export function relativeChangedFilePath(input: {
  selectedChangedFiles: Array<{ path: string }>
  activeFilePath: string
  offset: 1 | -1
}) {
  if (input.selectedChangedFiles.length === 0) {
    return ''
  }
  const currentIndex = input.selectedChangedFiles.findIndex(
    (file) => file.path === input.activeFilePath,
  )
  const baseIndex = currentIndex >= 0 ? currentIndex : input.offset > 0 ? -1 : 0
  const nextIndex =
    (baseIndex + input.offset + input.selectedChangedFiles.length) %
    input.selectedChangedFiles.length
  return input.selectedChangedFiles[nextIndex]?.path ?? ''
}

export function workspaceActiveFilePath(input: { openTabs: WorkspaceTab[]; activeTabKey: string }) {
  return input.openTabs.find((tab) => workspaceTabKey(tab) === input.activeTabKey)?.filePath ?? ''
}

export function buildWorkspaceFocusContext(input: {
  selectedEditorState: WorkspaceFileEditorState | null
  hasActiveTab: boolean
  recentFiles: WorkspaceRecentFile[]
  buildWorkingSet: (recentFiles: WorkspaceRecentFile[]) => WorkspaceFocusContext['workingSet']
}): WorkspaceFocusContext | null {
  if (!input.selectedEditorState || !input.hasActiveTab) {
    return null
  }
  return {
    selectedArea: input.selectedEditorState.selection ? 'selection' : 'edit',
    selection: input.selectedEditorState.selection,
    workingSet: input.buildWorkingSet(input.recentFiles),
  }
}

export async function createWorkspaceFileEntry(input: {
  conversationId: string
  repoPath: string
  path: string
  refreshWorkspace: (preserveSelection: boolean) => Promise<void>
  selectFile: (path: string) => void
}) {
  if (!input.conversationId || !input.repoPath || !input.path) {
    return false
  }
  await createProjectConversationWorkspaceFile(input.conversationId, {
    repoPath: input.repoPath,
    path: input.path,
  })
  await input.refreshWorkspace(true)
  input.selectFile(input.path)
  return true
}

export async function searchWorkspacePaths(input: {
  conversationId: string
  repoPath: string
  query: string
  limit: number
}): Promise<ProjectConversationWorkspaceSearchResult[]> {
  const trimmedQuery = input.query.trim()
  if (!input.conversationId || !input.repoPath || !trimmedQuery) {
    return []
  }
  const payload = await searchProjectConversationWorkspacePaths(input.conversationId, {
    repoPath: input.repoPath,
    query: trimmedQuery,
    limit: input.limit,
  })
  return payload.workspaceSearch.results
}

export function remapWorkspaceTabPath(input: {
  repoPath: string
  fromPath: string
  toPath: string
  openTabs: WorkspaceTab[]
  tabFileStates: Map<string, WorkspaceTabFileState>
  activeTabKey: string
  recentFiles: WorkspaceRecentFile[]
  setOpenTabs: (tabs: WorkspaceTab[]) => void
  setTabFileStates: (states: Map<string, WorkspaceTabFileState>) => void
  setActiveTabKey: (key: string) => void
  setRecentFiles: (files: WorkspaceRecentFile[]) => void
  renameEditorFileState: (repoPath: string, fromPath: string, toPath: string) => void
}) {
  const fromKey = workspaceTabKey({ repoPath: input.repoPath, filePath: input.fromPath })
  const toKey = workspaceTabKey({ repoPath: input.repoPath, filePath: input.toPath })
  input.setOpenTabs(
    input.openTabs.map((tab) =>
      tab.repoPath === input.repoPath && tab.filePath === input.fromPath
        ? { repoPath: input.repoPath, filePath: input.toPath }
        : tab,
    ),
  )

  const nextTabStates = new Map(input.tabFileStates)
  const existing = nextTabStates.get(fromKey)
  if (existing) {
    nextTabStates.delete(fromKey)
    nextTabStates.set(toKey, {
      ...existing,
      preview: existing.preview ? { ...existing.preview, path: input.toPath } : existing.preview,
      patch: existing.patch ? { ...existing.patch, path: input.toPath } : existing.patch,
    })
  }
  input.setTabFileStates(nextTabStates)

  if (input.activeTabKey === fromKey) {
    input.setActiveTabKey(toKey)
  }

  input.setRecentFiles(
    input.recentFiles.map((item) =>
      item.repoPath === input.repoPath && item.filePath === input.fromPath
        ? { repoPath: input.repoPath, filePath: input.toPath }
        : item,
    ),
  )
  input.renameEditorFileState(input.repoPath, input.fromPath, input.toPath)
}

export async function renameWorkspaceFileEntry(input: {
  conversationId: string
  repoPath: string
  fromPath: string
  toPath: string
  remapTabPath: () => void
  refreshWorkspace: (preserveSelection: boolean) => Promise<void>
  activateTab: (repoPath: string, filePath: string) => void
  getActiveTabKey: () => string
  loadFile: (repoPath: string, filePath: string, options?: { silent?: boolean }) => Promise<void>
}) {
  if (!input.conversationId || !input.repoPath || !input.fromPath || !input.toPath) {
    return false
  }

  await renameProjectConversationWorkspaceFile(input.conversationId, {
    repoPath: input.repoPath,
    fromPath: input.fromPath,
    toPath: input.toPath,
  })
  input.remapTabPath()
  await input.refreshWorkspace(true)
  input.activateTab(input.repoPath, input.toPath)
  if (
    input.getActiveTabKey() ===
    workspaceTabKey({ repoPath: input.repoPath, filePath: input.toPath })
  ) {
    await input.loadFile(input.repoPath, input.toPath, { silent: true })
  }
  return true
}

export async function deleteWorkspaceFileEntry(input: {
  conversationId: string
  repoPath: string
  path: string
  discardDraft: (repoPath: string, filePath: string) => void
  closeTab: (repoPath: string, filePath: string) => void
  refreshWorkspace: (preserveSelection: boolean) => Promise<void>
}) {
  if (!input.conversationId || !input.repoPath || !input.path) {
    return false
  }
  await deleteProjectConversationWorkspaceFile(input.conversationId, {
    repoPath: input.repoPath,
    path: input.path,
  })
  input.discardDraft(input.repoPath, input.path)
  input.closeTab(input.repoPath, input.path)
  await input.refreshWorkspace(true)
  return true
}

export async function reviewWorkspacePatch(input: {
  repoPath: string
  diff: ChatDiffPayload
  autoApply: boolean
  openTab: (repoPath: string, filePath: string) => void
  loadFile: (repoPath: string, filePath: string, options?: { silent?: boolean }) => Promise<void>
  reviewPatch: (repoPath: string, filePath: string, diff: ChatDiffPayload) => boolean
  applyPendingPatch: (repoPath: string, filePath: string) => boolean
}) {
  if (!input.repoPath || !input.diff.file) {
    return false
  }
  input.openTab(input.repoPath, input.diff.file)
  await input.loadFile(input.repoPath, input.diff.file, { silent: true })
  const ok = input.reviewPatch(input.repoPath, input.diff.file, input.diff)
  if (ok && input.autoApply) {
    return input.applyPendingPatch(input.repoPath, input.diff.file)
  }
  return ok
}
