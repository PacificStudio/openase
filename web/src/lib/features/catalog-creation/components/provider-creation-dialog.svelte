<script lang="ts">
  import { invalidateAll } from '$app/navigation'
  import type { AgentProvider, AgentProviderModelCatalogEntry, Machine } from '$lib/api/contracts'
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
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    orgId,
    open = $bindable(false),
    title,
    description,
    onCreated,
  }: {
    orgId: string
    open?: boolean
    title?: string
    description?: string
    onCreated?: (provider: AgentProvider) => void | Promise<void>
  } = $props()

  let draft = $state<ProviderDraft>(createEmptyProviderDraft())
  let creating = $state(false)
  let machines = $state<Machine[]>([])
  let modelCatalog = $state<AgentProviderModelCatalogEntry[]>([])
  const canSubmit = $derived(machines.length > 0)

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
      const payload = await createProvider(orgId, parsed.value)
      if (payload.provider && onCreated) {
        await onCreated(payload.provider)
      } else {
        await invalidateAll()
      }
      open = false
      reset()
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : i18nStore.t('catalog.provider.dialog.errors.create'),
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
      <Dialog.Title>
        {title ?? i18nStore.t('catalog.provider.dialog.title')}
      </Dialog.Title>
      <Dialog.Description>
        {description ?? i18nStore.t('catalog.provider.dialog.description')}
      </Dialog.Description>
    </Dialog.Header>

    <form
      class="flex min-h-0 flex-1 flex-col gap-6"
      onsubmit={(event) => {
        event.preventDefault()
        void handleSubmit()
      }}
    >
      <Dialog.Body class="space-y-4">
        <ProviderFormFields
          {draft}
          {modelCatalog}
          {machines}
          onFieldChange={(field, value) => {
            draft = { ...draft, [field]: value }
          }}
        />

        {#if !canSubmit}
          <p class="text-muted-foreground text-sm">
            {i18nStore.t('catalog.provider.dialog.hint.registerMachine')}
          </p>
        {/if}
      </Dialog.Body>

      <Dialog.Footer>
        <Dialog.Close>
          {#snippet child({ props })}
            <Button variant="outline" {...props}>
              {i18nStore.t('catalog.provider.dialog.actions.cancel')}
            </Button>
          {/snippet}
        </Dialog.Close>
        <Button type="submit" disabled={creating || !canSubmit}>
          {creating
            ? i18nStore.t('catalog.provider.dialog.actions.creating')
            : i18nStore.t('catalog.provider.dialog.actions.create')}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
