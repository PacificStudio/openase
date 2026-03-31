import { ApiError } from '$lib/api/client'
import { connectEventStream } from '$lib/api/sse'
import {
  createStatus,
  deleteStatus,
  listStatuses,
  resetStatuses,
  updateStatus,
} from '$lib/api/openase'
import { normalizeStages, type EditableStage } from '$lib/features/stages/public'
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
import { startStageRuntimeSync } from './stage-runtime-sync'
import { createStageActionHandlers } from './status-settings-stage-actions'

type StatusSettingsUI = {
  statuses: EditableStatus[]
  stages: EditableStage[]
  createName: string
  createColor: string
  createDefault: boolean
  createStageId: string
  loading: boolean
  creating: boolean
  creatingStage: boolean
  resetting: boolean
  busyStatusId: string
  busyStageId: string
}

function createStatusSettingsUI(): StatusSettingsUI {
  return {
    statuses: [],
    stages: [],
    createName: '',
    createColor: createEmptyStatusDraft().color,
    createDefault: false,
    createStageId: '',
    loading: false,
    creating: false,
    creatingStage: false,
    resetting: false,
    busyStatusId: '',
    busyStageId: '',
  }
}

export function createStatusSettingsState() {
  const ui = $state<StatusSettingsUI>(createStatusSettingsUI())

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      Object.assign(ui, createStatusSettingsUI())
      return
    }

    return startStageRuntimeSync({
      projectId,
      loadStatuses: listStatuses,
      connectEventStream,
      applySnapshot: assignPayload,
      setLoading: (loading) => (ui.loading = loading),
      onInitialError: (message) => toastStore.error(message),
      onRefreshError: (error) => {
        console.error('Failed to refresh status settings:', error)
      },
    })
  })

  function assignPayload(payload: Awaited<ReturnType<typeof listStatuses>>) {
    ui.statuses = normalizeStatuses(payload.statuses)
    ui.stages = normalizeStages(payload.stages)
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
    ui.createStageId = ''
  }

  function statusUpdateBody(current: EditableStatus, draft: ParsedStatusDraft) {
    const body: Parameters<typeof updateStatus>[1] = {}
    if (draft.name !== current.name) body.name = draft.name
    if (draft.color !== current.color) body.color = draft.color
    if (draft.isDefault !== current.isDefault) body.is_default = draft.isDefault
    if (draft.stageId !== current.stageId) body.stage_id = draft.stageId
    return body
  }

  function reportMutationError(error: unknown, fallback: string) {
    toastStore.error(error instanceof ApiError ? error.detail : fallback)
  }

  function touchAfterReload() {
    statusSync.touch()
  }

  const stageActions = createStageActionHandlers({
    ui,
    currentProjectId,
    reload,
    touchAfterReload,
    reportMutationError,
    toastSuccess: (message) => toastStore.success(message),
  })

  return {
    ui,
    async createStatus() {
      const projectId = currentProjectId()
      if (!projectId) return

      const parsed = parseStatusDraft({
        name: ui.createName,
        color: ui.createColor,
        isDefault: ui.createDefault,
        stageId: ui.createStageId,
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
          stage_id: parsed.value.stageId,
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

      if (
        typeof window !== 'undefined' &&
        !window.confirm('Reset statuses to the default template? Custom statuses will be removed.')
      ) {
        return
      }

      ui.resetting = true
      try {
        await resetStatuses(projectId)
        await reload(projectId)
        touchAfterReload()
        toastStore.success('Statuses reset to the default template.')
      } catch (caughtError) {
        reportMutationError(caughtError, 'Failed to reset statuses.')
      } finally {
        ui.resetting = false
      }
    },
    ...stageActions,
  }
}
