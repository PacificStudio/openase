import { ApiError } from '$lib/api/client'
import type { ProjectRepoRecord } from '$lib/api/contracts'
import { syncProjectRepoMirror } from '$lib/api/openase'
import { toastStore } from '$lib/stores/toast.svelte'
import { createRepositoryMirrorDraft } from '../repository-mirror-model'
import { projectRepoMirrorProjection } from '../repositories-readiness'
import { reloadReposAfterMutation } from './repositories-settings-feedback'
import type { RepositoriesSettingsUI } from './repositories-settings-ui'

export function openRepositoryMirrorDialog(ui: RepositoriesSettingsUI, repo: ProjectRepoRecord) {
  ui.mirrorRepoId = repo.id
  ui.mirrorDraft = createRepositoryMirrorDraft(ui.machines, repo)
  ui.mirrorErrorMessage = ''
  ui.mirrorDialogOpen = true
}

export async function runRepositoryMirrorAction(
  ui: RepositoriesSettingsUI,
  projectId: string | undefined,
  repo: ProjectRepoRecord,
  reloadRepos: () => Promise<void>,
) {
  if (!projectId) {
    toastStore.error('Project context is unavailable.')
    return
  }

  const mirror = projectRepoMirrorProjection(repo)
  if (mirror.action !== 'sync_mirror' || !mirror.mirrorMachineId) {
    openRepositoryMirrorDialog(ui, repo)
    return
  }

  ui.materializingId = repo.id
  try {
    await syncProjectRepoMirror(projectId, repo.id, { machine_id: mirror.mirrorMachineId })
    await reloadReposAfterMutation(reloadRepos, 'mirror_updated', 'Repository mirror updated.')
  } catch (caughtError) {
    toastStore.error(
      caughtError instanceof ApiError ? caughtError.detail : 'Failed to repair repository mirror.',
    )
  } finally {
    ui.materializingId = ''
  }
}
