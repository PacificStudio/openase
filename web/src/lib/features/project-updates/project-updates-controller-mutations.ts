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
      toastStore.success('Update posted.')
      await input.refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to post update.',
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
      toastStore.success('Update edited.')
      await input.refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to edit update.',
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
      toastStore.success('Update deleted.')
      await input.refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete update.',
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
      toastStore.success('Comment added.')
      await input.refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to add comment.',
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
      toastStore.success('Comment edited.')
      await input.refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to edit comment.',
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
      toastStore.success('Comment deleted.')
      await input.refreshLatestThreads(projectId)
      return true
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete comment.',
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
