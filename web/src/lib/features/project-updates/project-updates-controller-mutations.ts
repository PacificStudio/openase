import { ApiError } from '$lib/api/client'
import {
  createProjectUpdateComment,
  createProjectUpdateThread,
  deleteProjectUpdateComment,
  deleteProjectUpdateThread,
  updateProjectUpdateComment,
  updateProjectUpdateThread,
} from '$lib/api/openase'
import { toastStore } from '$lib/stores/toast.svelte'
import type { ProjectUpdateStatus } from './types'
import { projectUpdatesT } from './i18n'

type ProjectUpdatesMutationDeps = {
  getProjectId: () => string
  isCreatingThread: () => boolean
  setCreatingThread: (value: boolean) => void
  refreshLatestThreads: (projectId: string) => Promise<void>
}

export function createProjectUpdateMutationHandlers(input: ProjectUpdatesMutationDeps) {
  async function handleCreateThread(draft: { status: ProjectUpdateStatus; body: string }) {
    const projectId = input.getProjectId()
    if (!projectId || input.isCreatingThread()) {
      return false
    }

    input.setCreatingThread(true)
    try {
      await createProjectUpdateThread(projectId, draft)
      toastStore.success(projectUpdatesT('projectUpdates.postSuccess'))
      await input.refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : projectUpdatesT('projectUpdates.postFailed'),
      )
      return false
    } finally {
      input.setCreatingThread(false)
    }
  }

  async function handleSaveThread(
    threadId: string,
    draft: { status: ProjectUpdateStatus; body: string },
  ) {
    const projectId = input.getProjectId()
    if (!projectId) {
      return false
    }

    try {
      await updateProjectUpdateThread(projectId, threadId, draft)
      toastStore.success(projectUpdatesT('projectUpdates.editSuccess'))
      await input.refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : projectUpdatesT('projectUpdates.editFailed'),
      )
      return false
    }
  }

  async function handleDeleteThread(threadId: string) {
    const projectId = input.getProjectId()
    if (!projectId) {
      return false
    }

    try {
      await deleteProjectUpdateThread(projectId, threadId)
      toastStore.success(projectUpdatesT('projectUpdates.deleteSuccess'))
      await input.refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : projectUpdatesT('projectUpdates.deleteFailed'),
      )
      await input.refreshLatestThreads(projectId)
      return false
    }
  }

  async function handleCreateComment(threadId: string, body: string) {
    const projectId = input.getProjectId()
    if (!projectId) {
      return false
    }

    try {
      await createProjectUpdateComment(projectId, threadId, { body })
      toastStore.success(projectUpdatesT('projectUpdates.commentAddSuccess'))
      await input.refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : projectUpdatesT('projectUpdates.commentAddFailed'),
      )
      return false
    }
  }

  async function handleSaveComment(threadId: string, commentId: string, body: string) {
    const projectId = input.getProjectId()
    if (!projectId || !body) {
      return false
    }

    try {
      await updateProjectUpdateComment(projectId, threadId, commentId, { body })
      toastStore.success(projectUpdatesT('projectUpdates.commentEditSuccess'))
      await input.refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : projectUpdatesT('projectUpdates.commentEditFailed'),
      )
      return false
    }
  }

  async function handleDeleteComment(threadId: string, commentId: string) {
    const projectId = input.getProjectId()
    if (!projectId) {
      return false
    }

    try {
      await deleteProjectUpdateComment(projectId, threadId, commentId)
      toastStore.success(projectUpdatesT('projectUpdates.commentDeleteSuccess'))
      await input.refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : projectUpdatesT('projectUpdates.commentDeleteFailed'),
      )
      await input.refreshLatestThreads(projectId)
      return false
    }
  }

  return {
    handleCreateThread,
    handleSaveThread,
    handleDeleteThread,
    handleCreateComment,
    handleSaveComment,
    handleDeleteComment,
  }
}
