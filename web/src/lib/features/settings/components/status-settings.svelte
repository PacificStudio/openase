<script lang="ts">
  import {
    getSettingsSectionCapability,
    capabilityStateClasses,
    capabilityStateLabel,
  } from '$lib/features/capabilities'
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
  import { Separator } from '$ui/separator'
  import StatusSettingsCreate from './status-settings-create.svelte'
  import StatusSettingsRow from './status-settings-row.svelte'
  let statuses = $state<EditableStatus[]>([])
  let createName = $state('')
  let createColor = $state('#94a3b8')
  let createDefault = $state(false)
  let loading = $state(false)
  let creating = $state(false)
  let resetting = $state(false)
  let busyStatusId = $state('')
  let error = $state(''),
    feedback = $state('')
  const statusCapability = getSettingsSectionCapability('statuses')

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      statuses = []
      createName = ''
      createColor = createEmptyStatusDraft().color
      createDefault = false
      busyStatusId = ''
      creating = false
      resetting = false
      feedback = ''
      error = ''
      return
    }
    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const payload = await listStatuses(projectId)
        if (cancelled) return
        statuses = normalizeStatuses(payload.statuses)
      } catch (caughtError) {
        if (cancelled) return
        error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load statuses.'
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
    const payload = await listStatuses(projectId)
    statuses = normalizeStatuses(payload.statuses)
  }

  async function handleCreate() {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    const parsed = parseStatusDraft({
      name: createName,
      color: createColor,
      isDefault: createDefault,
    })
    if (!parsed.ok) {
      error = parsed.error
      feedback = ''
      return
    }

    creating = true
    error = ''
    feedback = ''

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
      feedback = `Created status "${payload.status.name}".`
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to create status.'
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
    error = ''
    feedback = ''

    try {
      await updateStatus(statusId, body)
      await reloadStatuses(projectId)
      statusSync.touch()
      feedback = `Updated status "${draft.name}".`
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to update status.'
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
    error = ''
    feedback = ''

    try {
      await Promise.all(
        nextStatuses.map((status) => updateStatus(status.id, { position: status.position })),
      )
      await reloadStatuses(projectId)
      statusSync.touch()
      feedback = 'Status order updated.'
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to reorder statuses.'
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
    error = ''
    feedback = ''

    try {
      await deleteStatus(status.id)
      await reloadStatuses(projectId)
      statusSync.touch()
      feedback = `Deleted status "${status.name}".`
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete status.'
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
    error = ''
    feedback = ''

    try {
      await resetStatuses(projectId)
      await reloadStatuses(projectId)
      statusSync.touch()
      feedback = 'Statuses reset to the default template.'
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to reset statuses.'
    } finally {
      resetting = false
    }
  }
</script>

<div class="max-w-lg space-y-6">
  <div>
    <div class="flex items-center gap-2">
      <h2 class="text-foreground text-base font-semibold">Statuses</h2>
      <span
        class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${capabilityStateClasses(statusCapability.state)}`}
      >
        {capabilityStateLabel(statusCapability.state)}
      </span>
    </div>
    <p class="text-muted-foreground mt-1 text-sm">{statusCapability.summary}</p>
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

  {#if loading}
    <div class="text-muted-foreground text-sm">Loading statuses…</div>
  {:else}
    <div class="space-y-2">
      {#if statuses.length === 0}
        <div class="text-muted-foreground rounded-md border border-dashed px-4 py-6 text-sm">
          No statuses yet. Add one above or use reset to seed the default workflow template.
        </div>
      {:else}
        {#each statuses as status, index (status.id)}
          <StatusSettingsRow
            {status}
            order={index}
            busy={busyStatusId === status.id || resetting || loading}
            canMoveUp={index > 0}
            canMoveDown={index < statuses.length - 1}
            onSave={handleSave}
            onDelete={handleDelete}
            onMoveUp={(statusId) => handleMove(statusId, 'up')}
            onMoveDown={(statusId) => handleMove(statusId, 'down')}
          />
        {/each}
      {/if}
    </div>
  {/if}

  {#if feedback}
    <p class="text-sm text-emerald-400">{feedback}</p>
  {/if}

  {#if error}
    <p class="text-destructive text-sm">{error}</p>
  {/if}
</div>
