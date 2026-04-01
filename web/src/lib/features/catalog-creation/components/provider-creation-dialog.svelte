<script lang="ts">
  import { invalidateAll } from '$app/navigation'
  import type { AgentProviderModelCatalogEntry, Machine } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { createProvider, listMachines, listProviderModelOptions } from '$lib/api/openase'
  import {
    createEmptyProviderDraft,
    parseProviderDraft,
    ProviderFormFields,
    type ProviderDraft,
  } from '$lib/features/agents/public'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'

  let {
    orgId,
    open = $bindable(false),
  }: {
    orgId: string
    open?: boolean
  } = $props()

  let draft = $state<ProviderDraft>(createEmptyProviderDraft())
  let creating = $state(false)
  let machines = $state<Machine[]>([])
  let modelCatalog = $state<AgentProviderModelCatalogEntry[]>([])

  function reset() {
    draft = createEmptyProviderDraft()
    creating = false
  }

  $effect(() => {
    if (!open) {
      return
    }

    let cancelled = false
    void Promise.all([listMachines(orgId), listProviderModelOptions()])
      .then(([machinesPayload, modelPayload]) => {
        if (cancelled) return
        machines = machinesPayload.machines
        modelCatalog = modelPayload.adapter_model_options
        if (!draft.machineId) {
          draft = { ...draft, machineId: machinesPayload.machines[0]?.id ?? '' }
        }
      })
      .catch(() => {
        if (!cancelled) {
          machines = []
          modelCatalog = []
        }
      })

    return () => {
      cancelled = true
    }
  })

  async function handleSubmit() {
    const parsed = parseProviderDraft(draft)
    if (!parsed.ok) {
      toastStore.error(parsed.error)
      return
    }

    creating = true

    try {
      await createProvider(orgId, parsed.value)
      open = false
      reset()
      await invalidateAll()
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to create provider.',
      )
    } finally {
      creating = false
    }
  }
</script>

<Dialog.Root
  bind:open
  onOpenChange={(next) => {
    if (!next) reset()
  }}
>
  <Dialog.Content class="sm:max-w-lg">
    <Dialog.Header>
      <Dialog.Title>Create provider</Dialog.Title>
      <Dialog.Description>
        Register a model adapter to use with agents in this organization.
      </Dialog.Description>
    </Dialog.Header>

    <form
      class="space-y-4"
      onsubmit={(event) => {
        event.preventDefault()
        void handleSubmit()
      }}
    >
      <ProviderFormFields
        {draft}
        {modelCatalog}
        {machines}
        onFieldChange={(field, value) => {
          draft = { ...draft, [field]: value }
        }}
      />

      <Dialog.Footer>
        <Dialog.Close>
          {#snippet child({ props })}
            <Button variant="outline" {...props}>Cancel</Button>
          {/snippet}
        </Dialog.Close>
        <Button type="submit" disabled={creating}>
          {creating ? 'Creating...' : 'Create provider'}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
