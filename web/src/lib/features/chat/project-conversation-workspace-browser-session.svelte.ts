import type {
  ProjectConversationWorkspaceGitGraph,
  ProjectConversationWorkspaceMetadata,
  ProjectConversationWorkspaceRepoRefs,
} from '$lib/api/chat'
import { areWorkspaceMetadataEqual } from './project-conversation-workspace-browser-state-helpers'

export function createWorkspaceBrowserSessionState() {
  let metadata = $state<ProjectConversationWorkspaceMetadata | null>(null)
  let metadataLoading = $state(false)
  let metadataError = $state('')
  let repoRefs = $state<ProjectConversationWorkspaceRepoRefs | null>(null)
  let repoRefsLoading = $state(false)
  let repoRefsError = $state('')
  let gitGraph = $state<ProjectConversationWorkspaceGitGraph | null>(null)
  let gitGraphLoading = $state(false)
  let gitGraphError = $state('')
  let selectedGitCommitID = $state('')
  let detailMode = $state<'file' | 'git_graph'>('file')

  function setMetadata(nextMetadata: ProjectConversationWorkspaceMetadata) {
    if (!areWorkspaceMetadataEqual(metadata, nextMetadata)) {
      metadata = nextMetadata
    }
  }

  function reset() {
    metadata = null
    metadataLoading = false
    metadataError = ''
    repoRefs = null
    repoRefsLoading = false
    repoRefsError = ''
    gitGraph = null
    gitGraphLoading = false
    gitGraphError = ''
    selectedGitCommitID = ''
    detailMode = 'file'
  }

  return {
    get metadata() {
      return metadata
    },
    setMetadata,
    setMetadataLoading: (loading: boolean) => {
      metadataLoading = loading
    },
    get metadataLoading() {
      return metadataLoading
    },
    setMetadataError: (error: string) => {
      metadataError = error
    },
    get metadataError() {
      return metadataError
    },
    setRepoRefs: (value: ProjectConversationWorkspaceRepoRefs | null) => {
      repoRefs = value
    },
    get repoRefs() {
      return repoRefs
    },
    setRepoRefsLoading: (loading: boolean) => {
      repoRefsLoading = loading
    },
    get repoRefsLoading() {
      return repoRefsLoading
    },
    setRepoRefsError: (error: string) => {
      repoRefsError = error
    },
    get repoRefsError() {
      return repoRefsError
    },
    setGitGraph: (value: ProjectConversationWorkspaceGitGraph | null) => {
      gitGraph = value
    },
    get gitGraph() {
      return gitGraph
    },
    setGitGraphLoading: (loading: boolean) => {
      gitGraphLoading = loading
    },
    get gitGraphLoading() {
      return gitGraphLoading
    },
    setGitGraphError: (error: string) => {
      gitGraphError = error
    },
    get gitGraphError() {
      return gitGraphError
    },
    get selectedGitCommitID() {
      return selectedGitCommitID
    },
    setSelectedGitCommitID: (commitId: string) => {
      selectedGitCommitID = commitId
    },
    get detailMode() {
      return detailMode
    },
    setDetailMode: (mode: 'file' | 'git_graph') => {
      detailMode = mode
    },
    clearMetadata: () => {
      metadata = null
    },
    reset,
  }
}
