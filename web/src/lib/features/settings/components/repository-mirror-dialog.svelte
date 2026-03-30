<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import type { Machine, ProjectRepoRecord } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import {
    suggestRepositoryMirrorLocalPath,
    type RepositoryMirrorDraft,
    type RepositoryMirrorMode,
  } from '../repository-mirror-model'

  let {
    open = $bindable(false),
    repo = null,
    draft,
    machines = [],
    saving = false,
    errorMessage = '',
    submitLabel = 'Set up mirror',
    title = 'Set up mirror',
    onDraftChange,
    onSubmit,
  }: {
    open?: boolean
    repo?: ProjectRepoRecord | null
    draft: RepositoryMirrorDraft
    machines?: Machine[]
    saving?: boolean
    errorMessage?: string
    submitLabel?: string
    title?: string
    onDraftChange?: (field: keyof RepositoryMirrorDraft, value: string) => void
    onSubmit?: () => void
  } = $props()

  const modeDescription = $derived.by(() => {
    if (draft.mode === 'register_existing') {
      return 'Use an existing local checkout and register it as the mirror for this machine.'
    }

    return 'Let OpenASE derive the machine-level default mirror path, or override it for this one repository.'
  })

  const machineLabel = (machine: Machine) =>
    machine.status === 'online' ? machine.name : `${machine.name} (${machine.status})`
  const selectedMachine = $derived(
    machines.find((machine) => machine.id === draft.machineId) ?? null,
  )
  const suggestedLocalPath = $derived(
    suggestRepositoryMirrorLocalPath(
      selectedMachine,
      repo,
      appStore.currentOrg?.slug ?? null,
      appStore.currentProject?.slug ?? null,
    ),
  )
  const localPathLabel = $derived(
    draft.mode === 'register_existing' ? 'Local path' : 'Local path override',
  )
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="sm:max-w-lg">
    <Dialog.Header>
      <Dialog.Title>{title}</Dialog.Title>
      <Dialog.Description>
        {#if repo}
          Configure a ready mirror for <span class="font-medium">{repo.name}</span>. Workflows can
          only pick up tickets after at least one target machine has a ready mirror.
        {:else}
          Configure a ready mirror for this repository.
        {/if}
      </Dialog.Description>
    </Dialog.Header>

    <div class="space-y-4 py-2">
      <div class="space-y-2">
        <Label>Mirror mode</Label>
        <Select.Root
          type="single"
          value={draft.mode}
          onValueChange={(value) =>
            onDraftChange?.('mode', (value || 'register_existing') as RepositoryMirrorMode)}
        >
          <Select.Trigger class="w-full">
            {draft.mode === 'register_existing'
              ? 'Register existing checkout'
              : 'Prepare new mirror'}
          </Select.Trigger>
          <Select.Content>
            <Select.Item value="register_existing">Register existing checkout</Select.Item>
            <Select.Item value="prepare">Prepare new mirror</Select.Item>
          </Select.Content>
        </Select.Root>
        <p class="text-muted-foreground text-xs">{modeDescription}</p>
      </div>

      <div class="space-y-2">
        <Label>Target machine</Label>
        <Select.Root
          type="single"
          value={draft.machineId}
          onValueChange={(value) => onDraftChange?.('machineId', value || '')}
        >
          <Select.Trigger class="w-full">
            {machines.find((machine) => machine.id === draft.machineId)?.name ?? 'Select machine'}
          </Select.Trigger>
          <Select.Content>
            {#each machines as machine (machine.id)}
              <Select.Item value={machine.id}>{machineLabel(machine)}</Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
      </div>

      <div class="space-y-2">
        <Label for="repository-mirror-local-path">{localPathLabel}</Label>
        <Input
          id="repository-mirror-local-path"
          value={draft.localPath}
          placeholder={draft.mode === 'register_existing'
            ? '/absolute/path/to/existing/repo'
            : suggestedLocalPath || '/absolute/path/for/mirror/clone'}
          oninput={(event) =>
            onDraftChange?.('localPath', (event.currentTarget as HTMLInputElement).value)}
        />
        <p class="text-muted-foreground text-xs">
          {#if draft.mode === 'register_existing'}
            This existing checkout path is required, must be absolute, and must live on the selected
            machine.
          {:else if suggestedLocalPath}
            Leave this blank to let OpenASE prepare the mirror at <span class="font-mono"
              >{suggestedLocalPath}</span
            >. Enter an absolute path only to override the machine-level default.
          {:else}
            Leave this blank to let OpenASE derive the default path on the backend. If the selected
            machine has no configured mirror root, remote machines need `workspace_root` or an
            explicit `mirror_root`.
          {/if}
        </p>
      </div>

      {#if errorMessage}
        <div class="border-destructive/30 bg-destructive/5 rounded-lg border px-3 py-2 text-sm">
          <p class="text-destructive">{errorMessage}</p>
        </div>
      {/if}
    </div>

    <Dialog.Footer class="mt-4">
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button onclick={onSubmit} disabled={saving || machines.length === 0}>
        {saving ? 'Submitting…' : submitLabel}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
