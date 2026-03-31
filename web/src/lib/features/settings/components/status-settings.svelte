<script lang="ts">
  /* eslint-disable max-lines */
  import { ApiError } from '$lib/api/client'
  import { connectEventStream } from '$lib/api/sse'
  import {
    createStage,
    createStatus,
    deleteStage,
    deleteStatus,
    listStatuses,
    resetStatuses,
    updateStage,
    updateStatus,
  } from '$lib/api/openase'
  import {
    moveStage,
    normalizeStages,
    type EditableStage,
    type ParsedStageDraft,
  } from '$lib/features/stages/public'
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
  import { Separator } from '$ui/separator'
  import { startStageRuntimeSync } from './stage-runtime-sync'
  import StageSettingsPanel from './stage-settings-panel.svelte'
  import StatusSettingsCreate from './status-settings-create.svelte'
  import StatusSettingsList from './status-settings-list.svelte'

  let statuses = $state<EditableStatus[]>([])
  let stages = $state<EditableStage[]>([])
  let createName = $state('')
  let createColor = $state('#94a3b8')
  let createDefault = $state(false)
  let createStageId = $state('')
  let loading = $state(false)
  let creating = $state(false)
  let creatingStage = $state(false)
  let resetting = $state(false)
  let busyStatusId = $state('')
  let busyStageId = $state('')

  function assignStatuses(payload: Awaited<ReturnType<typeof listStatuses>>) {
    statuses = normalizeStatuses(payload.statuses)
    stages = normalizeStages(payload.stages)
  }

  function resetEditorState() {
    statuses = []
    stages = []
    createName = ''
    createColor = createEmptyStatusDraft().color
    createDefault = false
    createStageId = ''
    busyStatusId = ''
    busyStageId = ''
    creating = false
    creatingStage = false
    resetting = false
    loading = false
  }

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      resetEditorState()
      return
    }

    return startStageRuntimeSync({
      projectId,
      loadStatuses: listStatuses,
      connectEventStream,
      applySnapshot: assignStatuses,
      setLoading: (nextLoading) => (loading = nextLoading),
      onInitialError: (message) => toastStore.error(message),
      onRefreshError: (error) => {
        console.error('Failed to refresh status settings:', error)
      },
    })
  })

  async function reloadStatuses(projectId: string) {
    assignStatuses(await listStatuses(projectId))
  }

  async function handleCreate() {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    const parsed = parseStatusDraft({
      name: createName,
      color: createColor,
      isDefault: createDefault,
      stageId: createStageId,
    })
    if (!parsed.ok) return void toastStore.error(parsed.error)

    creating = true

    try {
      const payload = await createStatus(projectId, {
        name: parsed.value.name,
        color: parsed.value.color,
        is_default: parsed.value.isDefault,
        stage_id: parsed.value.stageId,
      })
      await reloadStatuses(projectId)
      statusSync.touch()
      createName = ''
      createColor = createEmptyStatusDraft().color
      createDefault = false
      createStageId = ''
      toastStore.success(`Created status "${payload.status.name}".`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to create status.',
      )
    } finally {
      creating = false
    }
  }

  async function handleSave(statusId: string, draft: ParsedStatusDraft) {
    const projectId = appStore.currentProject?.id
    const current = statuses.find((status) => status.id === statusId)
    if (!projectId || !current) return

    const body: Parameters<typeof updateStatus>[1] = {}
    if (draft.name !== current.name) body.name = draft.name
    if (draft.color !== current.color) body.color = draft.color
    if (draft.isDefault !== current.isDefault) body.is_default = draft.isDefault
    if (draft.stageId !== current.stageId) body.stage_id = draft.stageId
    if (Object.keys(body).length === 0) return

    busyStatusId = statusId

    try {
      await updateStatus(statusId, body)
      await reloadStatuses(projectId)
      statusSync.touch()
      toastStore.success(`Updated status "${draft.name}".`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update status.',
      )
    } finally {
      busyStatusId = ''
    }
  }

  async function handleSetDefault(statusId: string) {
    const projectId = appStore.currentProject?.id
    const current = statuses.find((status) => status.id === statusId)
    if (!projectId || !current || current.isDefault) return

    busyStatusId = statusId

    try {
      await updateStatus(statusId, { is_default: true })
      await reloadStatuses(projectId)
      statusSync.touch()
      toastStore.success(`"${current.name}" is now the default status.`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to set default status.',
      )
    } finally {
      busyStatusId = ''
    }
  }

  async function handleMove(statusId: string, direction: 'up' | 'down') {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    const nextStatuses = moveStatus(statuses, statusId, direction)
    if (nextStatuses === statuses) return

    statuses = nextStatuses
    busyStatusId = statusId

    try {
      await Promise.all(
        nextStatuses.map((status) => updateStatus(status.id, { position: status.position })),
      )
      await reloadStatuses(projectId)
      statusSync.touch()
      toastStore.success('Status order updated.')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to reorder statuses.',
      )
      await reloadStatuses(projectId)
    } finally {
      busyStatusId = ''
    }
  }

  async function handleDelete(status: EditableStatus) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    if (typeof window !== 'undefined') {
      const confirmed = window.confirm(
        `Delete "${status.name}"? Tickets assigned to it will be moved to a replacement status.`,
      )
      if (!confirmed) return
    }

    busyStatusId = status.id

    try {
      await deleteStatus(status.id)
      await reloadStatuses(projectId)
      statusSync.touch()
      toastStore.success(`Deleted status "${status.name}".`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete status.',
      )
    } finally {
      busyStatusId = ''
    }
  }

  async function handleReset() {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    if (typeof window !== 'undefined') {
      const confirmed = window.confirm(
        'Reset statuses to the default template? Custom statuses will be removed.',
      )
      if (!confirmed) return
    }

    resetting = true

    try {
      await resetStatuses(projectId)
      await reloadStatuses(projectId)
      statusSync.touch()
      toastStore.success('Statuses reset to the default template.')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to reset statuses.',
      )
    } finally {
      resetting = false
    }
  }

  async function handleCreateStage(draft: ParsedStageDraft) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    creatingStage = true

    try {
      const payload = await createStage(projectId, {
        key: draft.key,
        name: draft.name,
        max_active_runs: draft.maxActiveRuns,
      })
      await reloadStatuses(projectId)
      statusSync.touch()
      toastStore.success(`Created stage "${payload.stage.name}".`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to create stage.',
      )
    } finally {
      creatingStage = false
    }
  }

  async function handleSaveStage(stageId: string, draft: ParsedStageDraft) {
    const projectId = appStore.currentProject?.id
    const current = stages.find((stage) => stage.id === stageId)
    if (!projectId || !current) return

    const body: Parameters<typeof updateStage>[1] = {}
    if (draft.name !== current.name) body.name = draft.name
    if (draft.maxActiveRuns !== current.maxActiveRuns) body.max_active_runs = draft.maxActiveRuns
    if (Object.keys(body).length === 0) return

    busyStageId = stageId

    try {
      await updateStage(stageId, body)
      await reloadStatuses(projectId)
      statusSync.touch()
      toastStore.success(`Updated stage "${draft.name}".`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update stage.',
      )
    } finally {
      busyStageId = ''
    }
  }

  async function handleMoveStage(stageId: string, direction: 'up' | 'down') {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    const nextStages = moveStage(stages, stageId, direction)
    if (nextStages === stages) return

    stages = nextStages
    busyStageId = stageId

    try {
      await Promise.all(
        nextStages.map((stage) => updateStage(stage.id, { position: stage.position })),
      )
      await reloadStatuses(projectId)
      statusSync.touch()
      toastStore.success('Stage order updated.')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to reorder stages.',
      )
      await reloadStatuses(projectId)
    } finally {
      busyStageId = ''
    }
  }

  async function handleDeleteStage(stage: EditableStage) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    if (typeof window !== 'undefined') {
      const confirmed = window.confirm(
        `Delete "${stage.name}"? Statuses in this stage will become ungrouped until you reassign them.`,
      )
      if (!confirmed) return
    }

    busyStageId = stage.id

    try {
      await deleteStage(stage.id)
      await reloadStatuses(projectId)
      statusSync.touch()
      toastStore.success(`Deleted stage "${stage.name}".`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete stage.',
      )
    } finally {
      busyStageId = ''
    }
  }
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Statuses</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Configure the stage groups that workflows share, then assign statuses into those stages.
    </p>
  </div>

  <Separator />

  <div class="grid gap-6 xl:grid-cols-[minmax(0,1.05fr)_minmax(0,1fr)]">
    <StageSettingsPanel
      {stages}
      {statuses}
      {loading}
      creating={creatingStage}
      {busyStageId}
      onCreate={handleCreateStage}
      onSave={handleSaveStage}
      onDelete={handleDeleteStage}
      onMove={handleMoveStage}
    />

    <div class="space-y-6">
      <StatusSettingsCreate
        bind:name={createName}
        bind:color={createColor}
        bind:isDefault={createDefault}
        bind:stageId={createStageId}
        {stages}
        {creating}
        {loading}
        {resetting}
        onCreate={handleCreate}
        onReset={handleReset}
      />

      <StatusSettingsList
        {statuses}
        {stages}
        {loading}
        {resetting}
        {busyStatusId}
        onSave={handleSave}
        onDelete={handleDelete}
        onMove={handleMove}
        onSetDefault={handleSetDefault}
      />
    </div>
  </div>
</div>
