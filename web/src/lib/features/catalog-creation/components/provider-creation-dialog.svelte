<script lang="ts">
  import { invalidateAll } from '$app/navigation'
  import type { AgentProviderModelCatalogEntry, Machine } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { createProvider, listMachines, listProviderModelOptions } from '$lib/api/openase'
  import {
    createEmptyProviderDraft,
    parseProviderDraft,
    ProviderModelPicker,
    providerAdapterOptions,
  } from '$lib/features/agents/public'
  import type { ProviderDraft } from '$lib/features/agents/public'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'

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

  function updateField(field: keyof ProviderDraft, value: string) {
    draft = { ...draft, [field]: value }
  }

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
      <div class="space-y-2">
        <Label for="provider-name">Provider name</Label>
        <Input
          id="provider-name"
          value={draft.name}
          placeholder="Codex primary"
          oninput={(event) => updateField('name', (event.currentTarget as HTMLInputElement).value)}
        />
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-2">
          <Label>Execution machine</Label>
          <Select.Root
            type="single"
            value={draft.machineId}
            onValueChange={(value) => updateField('machineId', value || '')}
          >
            <Select.Trigger class="w-full">
              {machines.find((machine) => machine.id === draft.machineId)?.name ?? 'Select machine'}
            </Select.Trigger>
            <Select.Content>
              {#each machines as machine (machine.id)}
                <Select.Item value={machine.id}
                  >{machine.name} · {machine.status} · {machine.host}</Select.Item
                >
              {/each}
            </Select.Content>
          </Select.Root>
        </div>

        <div class="space-y-2">
          <Label>Adapter</Label>
          <Select.Root
            type="single"
            value={draft.adapterType}
            onValueChange={(value) => {
              draft = { ...draft, adapterType: value || 'custom' }
            }}
          >
            <Select.Trigger class="w-full">
              {providerAdapterOptions.find((o) => o.value === draft.adapterType)?.label ??
                'Select adapter'}
            </Select.Trigger>
            <Select.Content>
              {#each providerAdapterOptions as option (option.value)}
                <Select.Item value={option.value}>{option.label}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>

        <div class="space-y-2">
          <ProviderModelPicker
            adapterType={draft.adapterType}
            modelName={draft.modelName}
            {modelCatalog}
            inputId="provider-model"
            onModelNameChange={(value) => updateField('modelName', value)}
          />
        </div>
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-2">
          <Label for="provider-cli-command">CLI command</Label>
          <Input
            id="provider-cli-command"
            value={draft.cliCommand}
            placeholder="codex"
            oninput={(event) =>
              updateField('cliCommand', (event.currentTarget as HTMLInputElement).value)}
          />
        </div>

        <div class="space-y-2">
          <Label for="provider-cli-args">CLI args</Label>
          <Textarea
            id="provider-cli-args"
            rows={2}
            value={draft.cliArgs}
            placeholder={`app-server\n--listen`}
            oninput={(event) =>
              updateField('cliArgs', (event.currentTarget as HTMLTextAreaElement).value)}
          />
        </div>
      </div>

      <div class="space-y-2">
        <Label for="provider-auth-config">Auth config (JSON)</Label>
        <Textarea
          id="provider-auth-config"
          rows={3}
          value={draft.authConfig}
          placeholder={`{ "token": "secret" }`}
          oninput={(event) =>
            updateField('authConfig', (event.currentTarget as HTMLTextAreaElement).value)}
        />
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-2">
          <Label for="provider-temperature">Temperature</Label>
          <Input
            id="provider-temperature"
            type="number"
            min="0"
            step="0.01"
            value={draft.modelTemperature}
            oninput={(event) =>
              updateField('modelTemperature', (event.currentTarget as HTMLInputElement).value)}
          />
        </div>

        <div class="space-y-2">
          <Label for="provider-max-tokens">Max tokens</Label>
          <Input
            id="provider-max-tokens"
            type="number"
            min="1"
            step="1"
            value={draft.modelMaxTokens}
            oninput={(event) =>
              updateField('modelMaxTokens', (event.currentTarget as HTMLInputElement).value)}
          />
        </div>
      </div>

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
