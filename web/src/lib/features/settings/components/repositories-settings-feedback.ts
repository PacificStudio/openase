import { ApiError } from '$lib/api/client'
import { toastStore } from '$lib/stores/toast.svelte'

export type RepositoryReloadAction = 'created' | 'updated' | 'deleted'

export async function reloadReposAfterMutation(
  reloadRepos: () => Promise<void>,
  action: RepositoryReloadAction,
  successMessage: string | null,
) {
  try {
    await reloadRepos()
    if (successMessage) {
      toastStore.success(successMessage)
    }
  } catch (caughtError) {
    toastStore.error(
      caughtError instanceof ApiError
        ? reloadFailureMessage(action, caughtError.detail)
        : reloadFailureMessage(action),
    )
  }
}

function reloadFailureMessage(action: RepositoryReloadAction, detail?: string) {
  const message = `Repository ${action}, but reloading the repository list failed.`
  return detail ? `${message} ${detail}` : message
}
