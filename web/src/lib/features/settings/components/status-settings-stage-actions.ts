import { createStage, deleteStage, updateStage } from '$lib/api/openase'
import { moveStage, type EditableStage, type ParsedStageDraft } from '$lib/features/stages/public'

type StageUI = {
  stages: EditableStage[]
  creatingStage: boolean
  busyStageId: string
}

type StageActionDeps = {
  ui: StageUI
  currentProjectId: () => string | null
  reload: (projectId: string) => Promise<void>
  touchAfterReload: () => void
  reportMutationError: (error: unknown, fallback: string) => void
  toastSuccess: (message: string) => void
}

function stageUpdateBody(current: EditableStage, draft: ParsedStageDraft) {
  const body: Parameters<typeof updateStage>[1] = {}
  if (draft.name !== current.name) body.name = draft.name
  if (draft.maxActiveRuns !== current.maxActiveRuns) body.max_active_runs = draft.maxActiveRuns
  return body
}

export function createStageActionHandlers(deps: StageActionDeps) {
  return {
    async createStage(draft: ParsedStageDraft) {
      const projectId = deps.currentProjectId()
      if (!projectId) return

      deps.ui.creatingStage = true
      try {
        const payload = await createStage(projectId, {
          key: draft.key,
          name: draft.name,
          max_active_runs: draft.maxActiveRuns,
        })
        await deps.reload(projectId)
        deps.touchAfterReload()
        deps.toastSuccess(`Created stage "${payload.stage.name}".`)
      } catch (caughtError) {
        deps.reportMutationError(caughtError, 'Failed to create stage.')
      } finally {
        deps.ui.creatingStage = false
      }
    },
    async saveStage(stageId: string, draft: ParsedStageDraft) {
      const projectId = deps.currentProjectId()
      const current = deps.ui.stages.find((stage) => stage.id === stageId)
      if (!projectId || !current) return

      const body = stageUpdateBody(current, draft)
      if (Object.keys(body).length === 0) return

      deps.ui.busyStageId = stageId
      try {
        await updateStage(stageId, body)
        await deps.reload(projectId)
        deps.touchAfterReload()
        deps.toastSuccess(`Updated stage "${draft.name}".`)
      } catch (caughtError) {
        deps.reportMutationError(caughtError, 'Failed to update stage.')
      } finally {
        deps.ui.busyStageId = ''
      }
    },
    async moveStage(stageId: string, direction: 'up' | 'down') {
      const projectId = deps.currentProjectId()
      if (!projectId) return

      const nextStages = moveStage(deps.ui.stages, stageId, direction)
      if (nextStages === deps.ui.stages) return

      deps.ui.stages = nextStages
      deps.ui.busyStageId = stageId
      try {
        await Promise.all(
          nextStages.map((stage) => updateStage(stage.id, { position: stage.position })),
        )
        await deps.reload(projectId)
        deps.touchAfterReload()
        deps.toastSuccess('Stage order updated.')
      } catch (caughtError) {
        deps.reportMutationError(caughtError, 'Failed to reorder stages.')
        await deps.reload(projectId)
      } finally {
        deps.ui.busyStageId = ''
      }
    },
    async deleteStage(stage: EditableStage) {
      const projectId = deps.currentProjectId()
      if (!projectId) return

      if (
        typeof window !== 'undefined' &&
        !window.confirm(
          `Delete "${stage.name}"? Statuses in this stage will become ungrouped until you reassign them.`,
        )
      ) {
        return
      }

      deps.ui.busyStageId = stage.id
      try {
        await deleteStage(stage.id)
        await deps.reload(projectId)
        deps.touchAfterReload()
        deps.toastSuccess(`Deleted stage "${stage.name}".`)
      } catch (caughtError) {
        deps.reportMutationError(caughtError, 'Failed to delete stage.')
      } finally {
        deps.ui.busyStageId = ''
      }
    },
  }
}
