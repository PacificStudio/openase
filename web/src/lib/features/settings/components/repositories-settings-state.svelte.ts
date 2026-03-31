import { ApiError } from '$lib/api/client'
import type { ProjectRepoRecord } from '$lib/api/contracts'
import {
  createProjectRepo,
  deleteProjectRepo,
  listProjectRepos,
  updateProjectRepo,
} from '$lib/api/openase'
import { appStore } from '$lib/stores/app.svelte'
import { toastStore } from '$lib/stores/toast.svelte'
import {
  createEmptyRepositoryDraft,
  parseRepositoryDraft,
  projectRepoToDraft,
  sortProjectRepos,
  type RepositoryDraft,
} from '../repositories-model'
import {
  reloadReposAfterMutation,
  type RepositoryReloadAction,
} from './repositories-settings-feedback'
import {
  createRepositoriesSettingsUI,
  type RepositoriesSettingsUI,
} from './repositories-settings-ui'

export function createRepositoriesSettingsState() {
  const ui = $state<RepositoriesSettingsUI>(createRepositoriesSettingsUI())
  const selectedRepo = $derived(ui.repos.find((repo) => repo.id === ui.selectedId) ?? null)

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      reset()
      return
    }

    let cancelled = false
    const load = async () => {
      ui.loading = true
      try {
        const repoPayload = await listProjectRepos(projectId)
        if (!cancelled) {
          syncLoadedRepos(repoPayload.repos)
        }
      } finally {
        if (!cancelled) {
          ui.loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  function reset() {
    Object.assign(ui, createRepositoriesSettingsUI())
  }

  function syncLoadedRepos(nextRepos: ProjectRepoRecord[]) {
    const sortedRepos = sortProjectRepos(nextRepos)
    ui.repos = sortedRepos

    if (sortedRepos.length === 0) {
      ui.selectedId = ''
      ui.editorOpen = false
      ui.mode = 'create'
      ui.draft = createEmptyRepositoryDraft()
      return
    }

    if (ui.selectedId && !sortedRepos.some((repo) => repo.id === ui.selectedId)) {
      ui.selectedId = ''
      ui.editorOpen = false
      ui.mode = 'create'
      ui.draft = createEmptyRepositoryDraft()
    }
  }

  async function reloadRepos(projectId: string) {
    const payload = await listProjectRepos(projectId)
    syncLoadedRepos(payload.repos)
  }

  async function save() {
    const projectId = appStore.currentProject?.id
    const parsed = parseRepositoryDraft(ui.draft)
    if (!projectId || !parsed.ok) {
      toastStore.error(parsed.ok ? 'Project context is unavailable.' : parsed.error)
      return
    }

    ui.saving = true
    try {
      let successAction: Exclude<RepositoryReloadAction, 'deleted' | 'mirror_updated'>

      if (ui.mode === 'create') {
        const payload = await createProjectRepo(projectId, parsed.value)
        ui.selectedId = payload.repo.id
        successAction = 'created'
      } else if (selectedRepo) {
        const payload = await updateProjectRepo(projectId, selectedRepo.id, parsed.value)
        ui.selectedId = payload.repo.id
        successAction = 'updated'
      } else {
        return
      }

      await reloadReposAfterMutation(
        () => reloadRepos(projectId),
        successAction,
        `Repository ${successAction}.`,
      )
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to save repository.',
      )
    } finally {
      ui.saving = false
    }
  }

  async function deleteRepo(targetRepo: ProjectRepoRecord | null = selectedRepo) {
    const projectId = appStore.currentProject?.id
    if (!projectId || !targetRepo) {
      return
    }

    ui.deletingId = targetRepo.id
    try {
      await deleteProjectRepo(projectId, targetRepo.id)
      if (ui.selectedId === targetRepo.id) {
        ui.selectedId = ''
        ui.editorOpen = false
      }
      await reloadReposAfterMutation(() => reloadRepos(projectId), 'deleted', 'Repository deleted.')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete repository.',
      )
    } finally {
      ui.deletingId = ''
    }
  }

  return {
    ui,
    get selectedRepo() {
      return selectedRepo
    },
    openRepo(repo: ProjectRepoRecord) {
      ui.mode = 'edit'
      ui.selectedId = repo.id
      ui.draft = projectRepoToDraft(repo)
      ui.editorOpen = true
    },
    startCreate() {
      ui.mode = 'create'
      ui.selectedId = ''
      ui.draft = createEmptyRepositoryDraft()
      ui.editorOpen = true
    },
    updateField(field: keyof RepositoryDraft, value: string | boolean) {
      ui.draft = { ...ui.draft, [field]: value }
    },
    async deleteFromList(repo: ProjectRepoRecord) {
      if (repo.id !== ui.selectedId) {
        ui.selectedId = repo.id
        ui.mode = 'edit'
        ui.draft = projectRepoToDraft(repo)
      }
      await deleteRepo(repo)
    },
    save,
  }
}
