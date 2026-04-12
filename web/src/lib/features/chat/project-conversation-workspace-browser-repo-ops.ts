import {
  checkoutProjectConversationWorkspaceBranch,
  getProjectConversationWorkspaceGitGraph,
  getProjectConversationWorkspaceRepoRefs,
  listProjectConversationWorkspaceTree,
  type ProjectConversationWorkspaceDiff,
  type ProjectConversationWorkspaceGitGraph,
  type ProjectConversationWorkspaceMetadata,
  type ProjectConversationWorkspaceRepoRefs,
  type ProjectConversationWorkspaceTreeEntry,
} from '$lib/api/chat'
import type { WorkspaceTab } from './project-conversation-workspace-browser-state-helpers'

export async function refreshWorkspaceRepoGitContext(input: {
  conversationId: string
  repoPath: string
  treeRepoPath: string
  selectedGitCommitID: string
  setRepoRefsLoading: (loading: boolean) => void
  setRepoRefsError: (error: string) => void
  setRepoRefs: (value: ProjectConversationWorkspaceRepoRefs | null) => void
  setGitGraphLoading: (loading: boolean) => void
  setGitGraphError: (error: string) => void
  setGitGraph: (value: ProjectConversationWorkspaceGitGraph | null) => void
  setSelectedGitCommitID: (commitId: string) => void
  isCurrentConversation: () => boolean
}) {
  const { conversationId, repoPath } = input
  if (!conversationId || !repoPath) {
    input.setRepoRefs(null)
    input.setRepoRefsError('')
    input.setGitGraph(null)
    input.setGitGraphError('')
    input.setSelectedGitCommitID('')
    return
  }

  input.setRepoRefsLoading(true)
  input.setRepoRefsError('')
  input.setGitGraphLoading(true)
  input.setGitGraphError('')

  const [refsResult, graphResult] = await Promise.allSettled([
    getProjectConversationWorkspaceRepoRefs(conversationId, { repoPath }),
    getProjectConversationWorkspaceGitGraph(conversationId, { repoPath }),
  ])

  if (!input.isCurrentConversation() || repoPath !== input.treeRepoPath) {
    return
  }

  if (refsResult.status === 'fulfilled') {
    input.setRepoRefs(refsResult.value.repoRefs)
    input.setRepoRefsError('')
  } else {
    input.setRepoRefs(null)
    input.setRepoRefsError(
      refsResult.reason instanceof Error
        ? refsResult.reason.message
        : 'Failed to load workspace branches.',
    )
  }
  input.setRepoRefsLoading(false)

  if (graphResult.status === 'fulfilled') {
    input.setGitGraph(graphResult.value.gitGraph)
    input.setGitGraphError('')
    const nextSelectedCommit = graphResult.value.gitGraph.commits.find(
      (commit) => commit.commitId === input.selectedGitCommitID,
    )
    input.setSelectedGitCommitID(
      nextSelectedCommit?.commitId ??
        graphResult.value.gitGraph.commits.find((commit) => commit.head)?.commitId ??
        graphResult.value.gitGraph.commits[0]?.commitId ??
        '',
    )
  } else {
    input.setGitGraph(null)
    input.setGitGraphError(
      graphResult.reason instanceof Error
        ? graphResult.reason.message
        : 'Failed to load the workspace git graph.',
    )
    input.setSelectedGitCommitID('')
  }
  input.setGitGraphLoading(false)
}

export async function loadWorkspaceDirEntries(input: {
  conversationId: string
  repoPath: string
  dirPath: string
  requestID: number
  currentRequestID: number
  silent: boolean
  treeRepoPath: string
  setDirLoading: (dirPath: string, loading: boolean) => void
  setTreeEntries: (dirPath: string, entries: ProjectConversationWorkspaceTreeEntry[]) => void
}) {
  if (!input.repoPath || !input.conversationId) {
    return
  }

  if (!input.silent) {
    input.setDirLoading(input.dirPath, true)
  }

  try {
    const payload = await listProjectConversationWorkspaceTree(input.conversationId, {
      repoPath: input.repoPath,
      path: input.dirPath,
    })
    if (input.requestID !== input.currentRequestID || input.repoPath !== input.treeRepoPath) {
      return
    }
    input.setTreeEntries(input.dirPath, payload.workspaceTree.entries)
  } catch {
    input.setTreeEntries(input.dirPath, [])
  } finally {
    if (!input.silent) {
      input.setDirLoading(input.dirPath, false)
    }
  }
}

export async function revealWorkspaceFileInTree(input: {
  path: string
  requestID: number
  currentRequestID: () => number
  hasTreeEntries: (dirPath: string) => boolean
  setDirExpanded: (dirPath: string, expanded: boolean) => void
  loadDirEntries: (
    dirPath: string,
    requestID?: number,
    options?: { silent?: boolean },
  ) => Promise<void>
  options?: { silent?: boolean }
}) {
  if (!input.path) {
    return
  }

  const ancestorDirs = input.path
    .split('/')
    .slice(0, -1)
    .reduce<string[]>((dirs, segment) => {
      const nextPath = dirs.length > 0 ? `${dirs[dirs.length - 1]}/${segment}` : segment
      dirs.push(nextPath)
      return dirs
    }, [])

  if (ancestorDirs.length === 0) {
    return
  }
  if (!input.hasTreeEntries('')) {
    await input.loadDirEntries('', input.requestID, input.options)
  }

  for (const dirPath of ancestorDirs) {
    if (input.requestID !== input.currentRequestID()) {
      return
    }
    input.setDirExpanded(dirPath, true)
    if (!input.hasTreeEntries(dirPath)) {
      await input.loadDirEntries(dirPath, input.requestID, input.options)
    }
  }
}

export function computeWorkspaceCheckoutBlockers(input: {
  repoPath: string
  metadata: ProjectConversationWorkspaceMetadata | null
  workspaceDiff: ProjectConversationWorkspaceDiff | null
  openTabs: WorkspaceTab[]
  getEditorState: (
    repoPath: string,
    filePath: string,
  ) => { dirty: boolean; savePhase: string } | null
}) {
  const blockers: string[] = []
  const repoMetadata = input.metadata?.repos.find((repo) => repo.path === input.repoPath) ?? null
  if (
    repoMetadata?.dirty ||
    input.workspaceDiff?.repos.some((repo) => repo.path === input.repoPath && repo.dirty)
  ) {
    blockers.push('Workspace has uncommitted changes.')
  }

  const repoTabs = input.openTabs.filter((tab) => tab.repoPath === input.repoPath)
  if (
    repoTabs.some((tab) => {
      const state = input.getEditorState(tab.repoPath, tab.filePath)
      return state?.dirty === true
    })
  ) {
    blockers.push('Unsaved drafts must be saved or discarded first.')
  }
  if (
    repoTabs.some((tab) => {
      const state = input.getEditorState(tab.repoPath, tab.filePath)
      const phase = state?.savePhase
      return phase === 'saving' || phase === 'conflict'
    })
  ) {
    blockers.push('Resolve saving/conflict state before switching branches.')
  }
  return blockers
}

export async function checkoutWorkspaceBranch(input: {
  conversationId: string
  repoPath: string
  targetKind: 'local_branch' | 'remote_tracking_branch'
  targetName: string
  createTrackingBranch: boolean
  localBranchName?: string
  repoRefs?: ProjectConversationWorkspaceRepoRefs | null
  metadata: ProjectConversationWorkspaceMetadata | null
  workspaceDiff: ProjectConversationWorkspaceDiff | null
  openTabs: WorkspaceTab[]
  getEditorState: (
    repoPath: string,
    filePath: string,
  ) => { dirty: boolean; savePhase: string } | null
  setMetadata: (nextMetadata: ProjectConversationWorkspaceMetadata) => void
  refreshWorkspace: (preserveSelection: boolean) => Promise<void>
  refreshRepoGitContext: (repoPath?: string) => Promise<void>
}) {
  const blockers = computeWorkspaceCheckoutBlockers(input)
  if (blockers.length > 0) {
    return { ok: false as const, blockers }
  }

  const checkoutRequest = resolveWorkspaceCheckoutRequest(input)
  const response = await checkoutProjectConversationWorkspaceBranch(input.conversationId, {
    repoPath: input.repoPath,
    targetKind: checkoutRequest.targetKind,
    targetName: checkoutRequest.targetName,
    createTrackingBranch: checkoutRequest.createTrackingBranch,
    localBranchName: checkoutRequest.localBranchName,
    expectedCleanWorkspace: true,
  })

  if (input.metadata) {
    input.setMetadata({
      ...input.metadata,
      repos: input.metadata.repos.map((repo) =>
        repo.path === input.repoPath
          ? {
              ...repo,
              branch: response.checkout.currentRef.displayName,
              currentRef: response.checkout.currentRef,
            }
          : repo,
      ),
    })
  }

  await input.refreshWorkspace(true)
  await input.refreshRepoGitContext(input.repoPath)
  return { ok: true as const, blockers: [] as string[] }
}

export function resolveWorkspaceCheckoutRequest(input: {
  targetKind: 'local_branch' | 'remote_tracking_branch'
  targetName: string
  createTrackingBranch: boolean
  localBranchName?: string
  repoRefs?: ProjectConversationWorkspaceRepoRefs | null
}) {
  if (input.targetKind !== 'remote_tracking_branch') {
    return {
      targetKind: input.targetKind,
      targetName: input.targetName,
      createTrackingBranch: input.createTrackingBranch,
      localBranchName: input.localBranchName,
    }
  }

  const localBranchName =
    input.localBranchName?.trim() || deriveTrackingLocalBranchName(input.targetName)
  if (
    localBranchName &&
    input.repoRefs?.localBranches.some((branch) => branch.name === localBranchName)
  ) {
    return {
      targetKind: 'local_branch' as const,
      targetName: localBranchName,
      createTrackingBranch: false,
      localBranchName: undefined,
    }
  }

  return {
    targetKind: input.targetKind,
    targetName: input.targetName,
    createTrackingBranch: input.createTrackingBranch,
    localBranchName: input.localBranchName,
  }
}

function deriveTrackingLocalBranchName(targetName: string): string {
  const trimmed = targetName.trim()
  if (!trimmed) return ''
  const separator = trimmed.indexOf('/')
  if (separator < 0 || separator === trimmed.length - 1) {
    return trimmed
  }
  return trimmed.slice(separator + 1)
}
