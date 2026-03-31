import { ApiError } from '$lib/api/client'
import { connectEventStream } from '$lib/api/sse'
import {
  createStatus,
  deleteStatus,
  listStatuses,
  resetStatuses,
  updateStatus,
} from '$lib/api/openase'
import {
  createEmptyStatusDraft,
  moveStatus,
  normalizeStatuses,
  parseStatusDraft,
  statusSync,
  type EditableStatus,
  type ParsedStatusDraft,
} from '$lib/features/statuses/public'
import { appStore } from '$lib/stores/app.svelte'
import { toastStore } from '$lib/stores/toast.svelte'
import { startStatusRuntimeSync } from './status-runtime-sync'

type StatusSettingsUI = {
  statuses: EditableStatus[]
  createName: string
  createColor: string
  createDefault: boolean
  createMaxActiveRuns: string
  loading: boolean
  creating: boolean
  resetting: boolean
  busyStatusId: string
}

function createStatusSettingsUI(): StatusSettingsUI {
  return {
    statuses: [],
    createName: '',
    createColor: createEmptyStatusDraft().color,
    createDefault: false,
    createMaxActiveRuns: '',
    loading: false,
    creating: false,
    resetting: false,
    busyStatusId: '',
  }
}

type StatusSettingsSnapshot = Awaited<ReturnType<typeof listStatuses>>

export function createStatusSettingsState() {
  const ui = $state<StatusSettingsUI>(createStatusSettingsUI())

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      Object.assign(ui, createStatusSettingsUI())
      return
    }

    return startStatusRuntimeSync({
      projectId,
      loadSnapshot: listStatuses,
      connectEventStream,
      applySnapshot: assignPayload,
      setLoading: (loading) => (ui.loading = loading),
      onInitialError: (message) => toastStore.error(message),
      onRefreshError: (error) => {
        console.error('Failed to refresh status settings:', error)
      },
    })
  })

  function assignPayload(payload: StatusSettingsSnapshot) {
    ui.statuses = normalizeStatuses(payload.statuses)
  }

  async function reload(projectId: string) {
    assignPayload(await listStatuses(projectId))
  }

  function currentProjectId() {
    return appStore.currentProject?.id ?? null
  }

  function resetCreateStatusDraft() {
    ui.createName = ''
    ui.createColor = createEmptyStatusDraft().color
    ui.createDefault = false
    ui.createMaxActiveRuns = ''
  }

  function statusUpdateBody(current: EditableStatus, draft: ParsedStatusDraft) {
    const body: Parameters<typeof updateStatus>[1] = {}
    if (draft.name !== current.name) body.name = draft.name
    if (draft.color !== current.color) body.color = draft.color
    if (draft.isDefault !== current.isDefault) body.is_default = draft.isDefault
    if (draft.maxActiveRuns !== current.maxActiveRuns) body.max_active_runs = draft.maxActiveRuns
    return body
  }

  function reportMutationError(error: unknown, fallback: string) {
    toastStore.error(error instanceof ApiError ? error.detail : fallback)
  }

  function touchAfterReload() {
    statusSync.touch()
  }

  return {
    ui,
    async createStatus() {
      const projectId = currentProjectId()
      if (!projectId) return

      const parsed = parseStatusDraft({
        name: ui.createName,
        color: ui.createColor,
        isDefault: ui.createDefault,
        maxActiveRuns: ui.createMaxActiveRuns,
      })
      if (!parsed.ok) {
        toastStore.error(parsed.error)
        return
      }

      ui.creating = true
      try {
        const payload = await createStatus(projectId, {
          name: parsed.value.name,
          color: parsed.value.color,
          is_default: parsed.value.isDefault,
          max_active_runs: parsed.value.maxActiveRuns,
        })
        await reload(projectId)
        touchAfterReload()
        resetCreateStatusDraft()
        toastStore.success(`Created status "${payload.status.name}".`)
      } catch (caughtError) {
        reportMutationError(caughtError, 'Failed to create status.')
      } finally {
        ui.creating = false
      }
    },
    async saveStatus(statusId: string, draft: ParsedStatusDraft) {
      const projectId = currentProjectId()
      const current = ui.statuses.find((status) => status.id === statusId)
      if (!projectId || !current) return

      const body = statusUpdateBody(current, draft)
      if (Object.keys(body).length === 0) return

      ui.busyStatusId = statusId
      try {
        await updateStatus(statusId, body)
        await reload(projectId)
        touchAfterReload()
        toastStore.success(`Updated status "${draft.name}".`)
      } catch (caughtError) {
        reportMutationError(caughtError, 'Failed to update status.')
      } finally {
        ui.busyStatusId = ''
      }
    },
    async setDefaultStatus(statusId: string) {
      const projectId = currentProjectId()
      const current = ui.statuses.find((status) => status.id === statusId)
      if (!projectId || !current || current.isDefault) return

      ui.busyStatusId = statusId
      try {
        await updateStatus(statusId, { is_default: true })
        await reload(projectId)
        touchAfterReload()
        toastStore.success(`"${current.name}" is now the default status.`)
      } catch (caughtError) {
        reportMutationError(caughtError, 'Failed to set default status.')
      } finally {
        ui.busyStatusId = ''
      }
    },
    async moveStatus(statusId: string, direction: 'up' | 'down') {
      const projectId = currentProjectId()
      if (!projectId) return

      const nextStatuses = moveStatus(ui.statuses, statusId, direction)
      if (nextStatuses === ui.statuses) return

      ui.statuses = nextStatuses
      ui.busyStatusId = statusId
      try {
        await Promise.all(
          nextStatuses.map((status) => updateStatus(status.id, { position: status.position })),
        )
        await reload(projectId)
        touchAfterReload()
        toastStore.success('Status order updated.')
      } catch (caughtError) {
        reportMutationError(caughtError, 'Failed to reorder statuses.')
        await reload(projectId)
      } finally {
        ui.busyStatusId = ''
      }
    },
    async deleteStatus(status: EditableStatus) {
      const projectId = currentProjectId()
      if (!projectId) return

      if (
        typeof window !== 'undefined' &&
        !window.confirm(
          `Delete "${status.name}"? Tickets assigned to it will be moved to a replacement status.`,
        )
      ) {
        return
      }

      ui.busyStatusId = status.id
      try {
        await deleteStatus(status.id)
        await reload(projectId)
        touchAfterReload()
        toastStore.success(`Deleted status "${status.name}".`)
      } catch (caughtError) {
        reportMutationError(caughtError, 'Failed to delete status.')
      } finally {
        ui.busyStatusId = ''
      }
    },
    async resetStatuses() {
      const projectId = currentProjectId()
      if (!projectId) return

      ui.resetting = true
      try {
        await resetStatuses(projectId)
        await reload(projectId)
        touchAfterReload()
        toastStore.success('Reset statuses to the default template.')
      } catch (caughtError) {
        reportMutationError(caughtError, 'Failed to reset statuses.')
      } finally {
        ui.resetting = false
      }
    },
  }
}
