import type { Machine, ProjectRepoRecord } from '$lib/api/contracts'
import {
  createEmptyRepositoryDraft,
  type RepositoryDraft,
  type RepositoryEditorMode,
} from '../repositories-model'
import { createRepositoryMirrorDraft, type RepositoryMirrorDraft } from '../repository-mirror-model'

export type RepositoriesSettingsUI = {
  repos: ProjectRepoRecord[]
  machines: Machine[]
  loading: boolean
  saving: boolean
  deletingId: string
  materializingId: string
  editorOpen: boolean
  mirrorDialogOpen: boolean
  selectedId: string
  mirrorRepoId: string
  mode: RepositoryEditorMode
  draft: RepositoryDraft
  mirrorDraft: RepositoryMirrorDraft
  mirrorErrorMessage: string
}

export const emptyMirrorContext = {
  buttonLabel: 'Set up mirror',
  dialogTitle: 'Set up mirror',
  submitLabel: 'Set up mirror',
}

export function createRepositoriesSettingsUI(): RepositoriesSettingsUI {
  return {
    repos: [],
    machines: [],
    loading: false,
    saving: false,
    deletingId: '',
    materializingId: '',
    editorOpen: false,
    mirrorDialogOpen: false,
    selectedId: '',
    mirrorRepoId: '',
    mode: 'create',
    draft: createEmptyRepositoryDraft(),
    mirrorDraft: createRepositoryMirrorDraft([], null),
    mirrorErrorMessage: '',
  }
}
