import { ApiError } from '$lib/api/client'
import {
  createStatus,
  deleteStatus,
  listStatuses,
  resetStatuses,
  updateStatus,
} from '$lib/api/openase'
import {
  type EditableStatus,
  moveStatus,
  normalizeStatuses,
  parseStatusDraft,
  statusSync,
  ticketStatusStageLabel,
  type ParsedStatusDraft,
  type TicketStatusStage,
} from '$lib/features/statuses/public'
import { subscribeProjectEvents } from '$lib/features/project-events'
import { appStore } from '$lib/stores/app.svelte'
import { toastStore } from '$lib/stores/toast.svelte'
import { startStatusRuntimeSync } from './status-runtime-sync'
import {
  createStatusSettingsUI,
  resetCreateStatusDraft,
  statusCreateBody,
  statusUpdateBody,
  type StatusSettingsUI,
} from './status-settings-state-helpers'

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
      subscribeProjectEvents,
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

  const currentProjectId = () => appStore.currentProject?.id ?? null

  function reportMutationError(error: unknown, fallback: string) {
    toastStore.error(error instanceof ApiError ? error.detail : fallback)
  }

  async function createStatusFromDraft(
    projectId: string,
    draft: ParsedStatusDraft,
    isDefault: boolean,
  ) {
    const payload = await createStatus(projectId, statusCreateBody(draft, isDefault))
    await reload(projectId)
    touchAfterReload()
    return payload
  }

  const touchAfterReload = () => statusSync.touch()

  return {
    ui,
    async createStatus() {
      const projectId = currentProjectId()
      if (!projectId) return

      const parsed = parseStatusDraft({
        name: ui.createName,
        stage: ui.createStage,
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
        const payload = await createStatusFromDraft(projectId, parsed.value, parsed.value.isDefault)
        resetCreateStatusDraft(ui)
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
    async moveToStage(statusId: string, newStage: TicketStatusStage) {
      const projectId = currentProjectId()
      const current = ui.statuses.find((s) => s.id === statusId)
      if (!projectId || !current || current.stage === newStage) return

      ui.busyStatusId = statusId
      try {
        await updateStatus(statusId, { stage: newStage })
        await reload(projectId)
        touchAfterReload()
        toastStore.success(`Moved "${current.name}" to ${ticketStatusStageLabel(newStage)}.`)
      } catch (caughtError) {
        reportMutationError(caughtError, 'Failed to move status.')
      } finally {
        ui.busyStatusId = ''
      }
    },
    async createStatusInStage(
      stage: TicketStatusStage,
      name: string,
      color: string,
      maxActiveRuns: string,
    ): Promise<boolean> {
      const projectId = currentProjectId()
      if (!projectId) return false

      const parsed = parseStatusDraft({ name, stage, color, isDefault: false, maxActiveRuns })
      if (!parsed.ok) {
        toastStore.error(parsed.error)
        return false
      }

      ui.creating = true
      try {
        const payload = await createStatusFromDraft(projectId, parsed.value, false)
        toastStore.success(`Created status "${payload.status.name}".`)
        return true
      } catch (caughtError) {
        reportMutationError(caughtError, 'Failed to create status.')
        return false
      } finally {
        ui.creating = false
      }
    },
    async moveStatusInStage(statusId: string, direction: 'up' | 'down') {
      const projectId = currentProjectId()
      if (!projectId) return

      const target = ui.statuses.find((s) => s.id === statusId)
      if (!target) return

      const sameStage = ui.statuses
        .filter((s) => s.stage === target.stage)
        .sort((a, b) => a.position - b.position)

      const stageIdx = sameStage.findIndex((s) => s.id === statusId)
      const swapIdx = direction === 'up' ? stageIdx - 1 : stageIdx + 1
      if (swapIdx < 0 || swapIdx >= sameStage.length) return

      const swapWith = sameStage[swapIdx]
      const targetPos = target.position
      const swapPos = swapWith.position

      ui.statuses = ui.statuses.map((s) => {
        if (s.id === statusId) return { ...s, position: swapPos }
        if (s.id === swapWith.id) return { ...s, position: targetPos }
        return s
      })

      ui.busyStatusId = statusId
      try {
        await Promise.all([
          updateStatus(statusId, { position: swapPos }),
          updateStatus(swapWith.id, { position: targetPos }),
        ])
        await reload(projectId)
        touchAfterReload()
      } catch (caughtError) {
        reportMutationError(caughtError, 'Failed to reorder statuses.')
        await reload(projectId)
      } finally {
        ui.busyStatusId = ''
      }
    },
  }
}
