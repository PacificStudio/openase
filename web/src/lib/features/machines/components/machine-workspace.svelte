<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Skeleton } from '$ui/skeleton'
  import { Search, Server } from '@lucide/svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import MachineCreateWizard from './machine-create-wizard.svelte'
  import MachineEditorSheet from './machine-editor-sheet.svelte'
  import MachineRowCard from './machine-row-card.svelte'
  import { machineToDraft } from '../model'
  import type {
    MachineDraft,
    MachineDraftField,
    MachineEditorMode,
    MachineItem,
    MachineProbeResult,
    MachineSnapshot,
    MachineWorkspaceState,
  } from '../types'

  let {
    state,
    loading = false,
    selectedId = '',
    searchQuery = '',
    machines,
    selectedMachine,
    mode,
    draft,
    snapshot,
    probe,
    loadingHealth = false,
    refreshingHealthMachineId = '',
    saving = false,
    testingMachineId = '',
    deletingMachineId = '',
    statusUpdatingMachineId = '',
    editorOpen = $bindable(false),
    stateMessage = '',
    organizationId = null,
    onSearchChange,
    onSelectMachine,
    onDraftChange,
    onCreate,
    onWizardCreated,
    onRetry,
    onRefreshHealth,
    onSave,
    onTest,
    onDelete,
    onToggleMaintenance,
    onReset,
  }: {
    state: MachineWorkspaceState
    loading?: boolean
    selectedId?: string
    searchQuery?: string
    machines: MachineItem[]
    selectedMachine: MachineItem | null
    mode: MachineEditorMode
    draft: MachineDraft
    snapshot: MachineSnapshot | null
    probe: MachineProbeResult | null
    loadingHealth?: boolean
    refreshingHealthMachineId?: string
    saving?: boolean
    testingMachineId?: string
    deletingMachineId?: string
    statusUpdatingMachineId?: string
    editorOpen?: boolean
    stateMessage?: string
    organizationId?: string | null
    onSearchChange?: (value: string) => void
    onSelectMachine?: (machineId: string) => void
    onDraftChange?: (field: MachineDraftField, value: string) => void
    onCreate?: () => void
    onWizardCreated?: (machine: MachineItem) => void
    onRetry?: () => void
    onRefreshHealth?: (machineId: string) => void
    onSave?: () => void
    onTest?: (machineId: string) => void
    onDelete?: (machineId: string) => void
    onToggleMaintenance?: (machineId: string, enabled: boolean) => void
    onReset?: (machineId: string) => void
  } = $props()

  const emptyMessage = $derived(
    searchQuery.trim()
      ? i18nStore.t('machines.machineWorkspace.emptyState.filtered')
      : i18nStore.t('machines.machineWorkspace.emptyState.noneRegistered'),
  )
  const selectedDraft = $derived(
    selectedMachine && mode === 'edit' ? machineToDraft(selectedMachine) : null,
  )
  const hasSelectedDraftChanges = $derived(
    selectedDraft ? JSON.stringify(selectedDraft) !== JSON.stringify(draft) : false,
  )
</script>

<div
  class="flex min-h-0 flex-1 flex-col px-4 pb-4 sm:px-6 sm:pb-6"
  data-testid="machines-workspace"
>
  {#if state === 'no-org'}
    <div
      class="animate-fade-in-up border-border bg-card rounded-xl border border-dashed px-4 py-14 text-center"
    >
      <div class="bg-muted/60 mx-auto mb-4 flex size-12 items-center justify-center rounded-full">
        <Server class="text-muted-foreground size-5" />
      </div>
      <p class="text-foreground text-sm font-medium">
        {i18nStore.t('machines.machineWorkspace.noOrg.title')}
      </p>
      <p class="text-muted-foreground mt-1 text-sm">
        {i18nStore.t('machines.machineWorkspace.noOrg.description')}
      </p>
    </div>
  {:else if state === 'loading' || loading}
    <div class="space-y-3">
      {#each { length: 3 } as _}
        <div class="border-border bg-card rounded-lg border p-3 sm:p-4">
          <div class="flex items-start gap-3">
            <div class="min-w-0 flex-1 space-y-1">
              <div class="flex items-center gap-2">
                <Skeleton class="h-4 w-32" />
                <Skeleton class="hidden h-3 w-28 sm:block" />
              </div>
              <div class="flex items-center gap-1">
                <Skeleton class="h-4 w-14 rounded-full" />
                <Skeleton class="h-4 w-16 rounded-full" />
                <Skeleton class="h-4 w-20 rounded-full" />
              </div>
            </div>
            <div class="flex items-center gap-1">
              <Skeleton class="size-7 rounded-md" />
              <Skeleton class="size-7 rounded-md" />
            </div>
          </div>
          <div class="mt-3 grid grid-cols-2 gap-x-4 gap-y-2 sm:grid-cols-4">
            {#each { length: 4 } as __}
              <div class="space-y-1">
                <div class="flex items-baseline justify-between">
                  <Skeleton class="h-2.5 w-8" />
                  <Skeleton class="h-3 w-10" />
                </div>
                <Skeleton class="h-1.5 w-full rounded-full" />
              </div>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  {:else if state === 'error'}
    <div class="border-border bg-card rounded-xl border px-4 py-10 text-center text-sm">
      <p class="text-foreground">
        {stateMessage || i18nStore.t('machines.machineWorkspace.error.loadFailed')}
      </p>
      <div class="mt-4">
        <Button variant="outline" onclick={onRetry}>
          {i18nStore.t('machines.machineWorkspace.actions.retry')}
        </Button>
      </div>
    </div>
  {:else if state === 'empty'}
    <div
      class="animate-fade-in-up border-border bg-card rounded-xl border border-dashed px-4 py-14 text-center"
    >
      <div class="bg-muted/60 mx-auto mb-4 flex size-12 items-center justify-center rounded-full">
        <Server class="text-muted-foreground size-5" />
      </div>
      <p class="text-foreground text-sm font-medium">
        {i18nStore.t('machines.machineWorkspace.empty.title')}
      </p>
      <p class="text-muted-foreground mx-auto mt-1 max-w-sm text-sm">
        {i18nStore.t('machines.machineWorkspace.empty.description')}
      </p>
      <div class="mt-4">
        <Button onclick={onCreate}>
          {i18nStore.t('machines.machineWorkspace.actions.newMachine')}
        </Button>
      </div>
    </div>
  {:else}
    <div class="space-y-4" data-tour="machines-list-panel">
      <div class="relative w-full max-w-xs" data-tour="machines-search">
        <Search class="text-muted-foreground absolute top-2.5 left-2.5 size-3.5" />
        <Input
          value={searchQuery}
          class="h-9 pl-8 text-sm"
          placeholder={i18nStore.t('machines.machineWorkspace.searchPlaceholder')}
          oninput={(event) => onSearchChange?.((event.currentTarget as HTMLInputElement).value)}
        />
      </div>

      {#if machines.length === 0}
        <div class="border-border bg-card rounded-xl border border-dashed px-4 py-12 text-center">
          <div
            class="bg-muted/60 mx-auto mb-3 flex size-10 items-center justify-center rounded-full"
          >
            <Search class="text-muted-foreground size-4" />
          </div>
          <p class="text-muted-foreground text-sm">{emptyMessage}</p>
        </div>
      {:else}
        <div class="space-y-3">
          {#each machines as machine, idx (machine.id)}
            <div class="animate-stagger" style="--stagger-index: {idx}">
              <MachineRowCard
                {machine}
                selected={machine.id === selectedId && editorOpen}
                resetEnabled={machine.id === selectedId &&
                  editorOpen &&
                  mode === 'edit' &&
                  hasSelectedDraftChanges}
                testing={testingMachineId === machine.id}
                deleting={deletingMachineId === machine.id}
                maintenanceUpdating={statusUpdatingMachineId === machine.id}
                onOpen={() => onSelectMachine?.(machine.id)}
                onTest={() => onTest?.(machine.id)}
                onToggleMaintenance={(enabled) => onToggleMaintenance?.(machine.id, enabled)}
                onReset={() => onReset?.(machine.id)}
                onDelete={() => onDelete?.(machine.id)}
              />
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>

{#if mode === 'create'}
  <MachineCreateWizard bind:open={editorOpen} {organizationId} onCreated={onWizardCreated} />
{:else}
  <MachineEditorSheet
    bind:open={editorOpen}
    {mode}
    machine={selectedMachine}
    {draft}
    {snapshot}
    {probe}
    {loadingHealth}
    refreshingHealth={selectedMachine ? refreshingHealthMachineId === selectedMachine.id : false}
    {saving}
    maintenanceUpdating={selectedMachine ? statusUpdatingMachineId === selectedMachine.id : false}
    {onDraftChange}
    onRefreshHealth={selectedMachine ? () => onRefreshHealth?.(selectedMachine.id) : undefined}
    onToggleMaintenance={selectedMachine
      ? (enabled) => onToggleMaintenance?.(selectedMachine.id, enabled)
      : undefined}
    {onSave}
  />
{/if}
