import type { ProjectRepoRecord } from '$lib/api/contracts'
import {
  createEmptyRepositoryDraft,
  createEmptyGitHubRepositoryCreateDraft,
  type GitHubRepositoryCreateDraft,
  type GitHubRepositoryNamespace,
  type GitHubRepositoryRecord,
  type RepositoryDraft,
  type RepositoryEditorMode,
} from '../repositories-model'

export type RepositoriesSettingsUI = {
  repos: ProjectRepoRecord[]
  loading: boolean
  saving: boolean
  deletingId: string
  editorOpen: boolean
  selectedId: string
  mode: RepositoryEditorMode
  draft: RepositoryDraft
  githubRepoQuery: string
  githubRepos: GitHubRepositoryRecord[]
  githubReposLoading: boolean
  githubReposLoadingMore: boolean
  githubReposNextCursor: string
  githubRepoError: string
  githubBindingRepoFullName: string
  githubNamespaces: GitHubRepositoryNamespace[]
  githubNamespacesLoading: boolean
  githubCreateDraft: GitHubRepositoryCreateDraft
  githubCreating: boolean
}

export function createRepositoriesSettingsUI(): RepositoriesSettingsUI {
  return {
    repos: [],
    loading: false,
    saving: false,
    deletingId: '',
    editorOpen: false,
    selectedId: '',
    mode: 'create',
    draft: createEmptyRepositoryDraft(),
    githubRepoQuery: '',
    githubRepos: [],
    githubReposLoading: false,
    githubReposLoadingMore: false,
    githubReposNextCursor: '',
    githubRepoError: '',
    githubBindingRepoFullName: '',
    githubNamespaces: [],
    githubNamespacesLoading: false,
    githubCreateDraft: createEmptyGitHubRepositoryCreateDraft(),
    githubCreating: false,
  }
}
