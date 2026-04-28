import type {
  ProjectConversationWorkspaceGitGraph,
  ProjectConversationWorkspaceMetadata,
  ProjectConversationWorkspaceRepoRefs,
} from '$lib/api/chat'
import { areWorkspaceMetadataEqual } from './project-conversation-workspace-browser-state-helpers'
import {
  readWorkspaceAutosavePreference,
  storeWorkspaceAutosavePreference,
} from './workspace-browser-autosave'

export function createWorkspaceBrowserSessionState() {
  let metadata = $state<ProjectConversationWorkspaceMetadata | null>(null)
  let metadataLoading = $state(false)
  let metadataError = $state('')
  let autosaveEnabled = $state(readWorkspaceAutosavePreference())
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

  function clearGitContext() {
    repoRefs = null
    repoRefsError = ''
    gitGraph = null
    gitGraphError = ''
    selectedGitCommitID = ''
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

  function setAutosaveEnabled(enabled: boolean) {
    autosaveEnabled = enabled
    storeWorkspaceAutosavePreference(enabled)
  }

  return {
    get metadata() {
      return metadata
    },
    get metadataLoading() {
      return metadataLoading
    },
    get metadataError() {
      return metadataError
    },
    get autosaveEnabled() {
      return autosaveEnabled
    },
    get repoRefs() {
      return repoRefs
    },
    get repoRefsLoading() {
      return repoRefsLoading
    },
    get repoRefsError() {
      return repoRefsError
    },
    get gitGraph() {
      return gitGraph
    },
    get gitGraphLoading() {
      return gitGraphLoading
    },
    get gitGraphError() {
      return gitGraphError
    },
    get selectedGitCommitID() {
      return selectedGitCommitID
    },
    get detailMode() {
      return detailMode
    },
    setMetadataLoading: (loading: boolean) => {
      metadataLoading = loading
    },
    setMetadataError: (error: string) => {
      metadataError = error
    },
    setMetadata,
    clearMetadata: () => {
      metadata = null
    },
    setRepoRefsLoading: (loading: boolean) => {
      repoRefsLoading = loading
    },
    setRepoRefsError: (error: string) => {
      repoRefsError = error
    },
    setRepoRefs: (value: ProjectConversationWorkspaceRepoRefs | null) => {
      repoRefs = value
    },
    setGitGraphLoading: (loading: boolean) => {
      gitGraphLoading = loading
    },
    setGitGraphError: (error: string) => {
      gitGraphError = error
    },
    setGitGraph: (value: ProjectConversationWorkspaceGitGraph | null) => {
      gitGraph = value
    },
    setSelectedGitCommitID: (commitId: string) => {
      selectedGitCommitID = commitId
    },
    setDetailMode: (mode: 'file' | 'git_graph') => {
      detailMode = mode
    },
    setAutosaveEnabled,
    clearGitContext,
    reset,
  }
}
