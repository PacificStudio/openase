import type { ProjectRepoRecord } from '$lib/api/contracts'
import {
  createEmptyRepositoryDraft,
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
  }
}
