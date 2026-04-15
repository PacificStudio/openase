import type { ProjectConversationWorkspaceTreeEntry } from '$lib/api/chat'
import { areTreeEntriesEqual } from './project-conversation-workspace-browser-state-helpers'

export function createWorkspaceBrowserTreeState() {
  let treeNodes = $state<Map<string, ProjectConversationWorkspaceTreeEntry[]>>(new Map())
  let expandedDirs = $state<Set<string>>(new Set())
  let loadingDirs = $state<Set<string>>(new Set())

  function setTreeEntries(dirPath: string, entries: ProjectConversationWorkspaceTreeEntry[]) {
    if (areTreeEntriesEqual(treeNodes.get(dirPath), entries)) return
    const nextTreeNodes = new Map(treeNodes)
    nextTreeNodes.set(dirPath, entries)
    treeNodes = nextTreeNodes
  }

  function setDirLoading(dirPath: string, loading: boolean) {
    if (loadingDirs.has(dirPath) === loading) return
    const nextLoadingDirs = new Set(loadingDirs)
    if (loading) nextLoadingDirs.add(dirPath)
    else nextLoadingDirs.delete(dirPath)
    loadingDirs = nextLoadingDirs
  }

  function setDirExpanded(dirPath: string, expanded: boolean) {
    if (expandedDirs.has(dirPath) === expanded) return
    const nextExpandedDirs = new Set(expandedDirs)
    if (expanded) nextExpandedDirs.add(dirPath)
    else nextExpandedDirs.delete(dirPath)
    expandedDirs = nextExpandedDirs
  }

  function reset() {
    treeNodes = new Map()
    expandedDirs = new Set()
    loadingDirs = new Set()
  }

  return {
    get treeNodes() {
      return treeNodes
    },
    get expandedDirs() {
      return expandedDirs
    },
    get loadingDirs() {
      return loadingDirs
    },
    setTreeEntries,
    setDirLoading,
    setDirExpanded,
    reset,
  }
}
