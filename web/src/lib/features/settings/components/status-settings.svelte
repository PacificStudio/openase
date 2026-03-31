<script lang="ts">
  import { ApiError } from '$lib/api/client'
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
  } from '$lib/features/statuses/public'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Separator } from '$ui/separator'
  import StatusSettingsCreate from './status-settings-create.svelte'
  import StatusSettingsList from './status-settings-list.svelte'
  let statuses = $state<EditableStatus[]>([])
  let createName = $state('')
  let createColor = $state('#94a3b8')
  let createDefault = $state(false)
  let loading = $state(false)
  let creating = $state(false)
  let resetting = $state(false)
  let busyStatusId = $state('')

  function assignStatuses(payload: Awaited<ReturnType<typeof listStatuses>>) {
    statuses = normalizeStatuses(payload.statuses)
  }

  function resetEditorState() {
    statuses = []
    createName = ''
    createColor = createEmptyStatusDraft().color
    createDefault = false
    busyStatusId = ''
    creating = false
    resetting = false
    loading = false
  }

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      resetEditorState()
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true

      try {
        const payload = await listStatuses(projectId)
        if (cancelled) return
        assignStatuses(payload)
      } catch (error) {
        if (!cancelled) {
          toastStore.error(error instanceof ApiError ? error.detail : 'Failed to load statuses.')
        }
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
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
    })
    if (!parsed.ok) return void toastStore.error(parsed.error)

    creating = true

    try {
      const payload = await createStatus(projectId, {
        name: parsed.value.name,
        color: parsed.value.color,
        is_default: parsed.value.isDefault,
      })
      await reloadStatuses(projectId)
      statusSync.touch()
      createName = ''
      createColor = createEmptyStatusDraft().color
      createDefault = false
      toastStore.success(`Created status "${payload.status.name}".`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to create status.',
      )
    } finally {
      creating = false
    }
  }

  async function handleSave(
    statusId: string,
    draft: { name: string; color: string; isDefault: boolean },
  ) {
    const projectId = appStore.currentProject?.id
    const current = statuses.find((status) => status.id === statusId)
    if (!projectId || !current) return

    const body: Parameters<typeof updateStatus>[1] = {}
    if (draft.name !== current.name) body.name = draft.name
    if (draft.color !== current.color) body.color = draft.color
    if (draft.isDefault !== current.isDefault) body.is_default = draft.isDefault
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
</script>

<div class="max-w-lg space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Statuses</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Create, edit, reorder, and manage board statuses.
    </p>
  </div>

  <Separator />

  <StatusSettingsCreate
    bind:name={createName}
    bind:color={createColor}
    bind:isDefault={createDefault}
    {creating}
    {loading}
    {resetting}
    onCreate={handleCreate}
    onReset={handleReset}
  />

  <StatusSettingsList
    {statuses}
    {loading}
    {resetting}
    {busyStatusId}
    onSave={handleSave}
    onDelete={handleDelete}
    onMove={handleMove}
    onSetDefault={handleSetDefault}
  />
</div>
