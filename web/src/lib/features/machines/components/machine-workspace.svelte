<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Skeleton } from '$ui/skeleton'
  import { Search, Server } from '@lucide/svelte'
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
    editorOpen = $bindable(false),
    stateMessage = '',
    onSearchChange,
    onSelectMachine,
    onDraftChange,
    onCreate,
    onRetry,
    onRefreshHealth,
    onSave,
    onTest,
    onDelete,
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
    editorOpen?: boolean
    stateMessage?: string
    onSearchChange?: (value: string) => void
    onSelectMachine?: (machineId: string) => void
    onDraftChange?: (field: MachineDraftField, value: string) => void
    onCreate?: () => void
    onRetry?: () => void
    onRefreshHealth?: (machineId: string) => void
    onSave?: () => void
    onTest?: (machineId: string) => void
    onDelete?: (machineId: string) => void
    onReset?: (machineId: string) => void
  } = $props()

  const emptyMessage = $derived(
    searchQuery.trim() ? 'No machines match the current filter.' : 'No machines registered yet.',
  )
  const selectedDraft = $derived(
    selectedMachine && mode === 'edit' ? machineToDraft(selectedMachine) : null,
  )
  const hasSelectedDraftChanges = $derived(
    selectedDraft ? JSON.stringify(selectedDraft) !== JSON.stringify(draft) : false,
  )
</script>

<div class="flex min-h-0 flex-1 flex-col px-6 pb-6" data-testid="machines-workspace">
  {#if state === 'no-org'}
    <div
      class="animate-fade-in-up border-border bg-card rounded-xl border border-dashed px-4 py-14 text-center"
    >
      <div class="bg-muted/60 mx-auto mb-4 flex size-12 items-center justify-center rounded-full">
        <Server class="text-muted-foreground size-5" />
      </div>
      <p class="text-foreground text-sm font-medium">No organization selected</p>
      <p class="text-muted-foreground mt-1 text-sm">
        Create an organization before managing machines.
      </p>
    </div>
  {:else if state === 'loading' || loading}
    <div class="space-y-3">
      {#each { length: 3 } as _}
        <div class="border-border bg-card rounded-2xl border p-4">
          <div class="grid gap-4 xl:grid-cols-[minmax(0,18rem)_minmax(0,1fr)_auto] xl:items-start">
            <div class="space-y-3">
              <div class="space-y-1.5">
                <div class="flex items-center gap-2">
                  <Skeleton class="h-5 w-32" />
                  <Skeleton class="h-4 w-10 rounded-full" />
                </div>
                <Skeleton class="h-3.5 w-28" />
              </div>
              <div class="flex items-center gap-1.5">
                <Skeleton class="size-2 rounded-full" />
                <Skeleton class="size-2 rounded-full" />
                <Skeleton class="size-2 rounded-full" />
              </div>
            </div>
            <div class="space-y-2">
              {#each { length: 3 } as __}
                <div class="flex items-center gap-3">
                  <Skeleton class="h-3 w-10" />
                  <Skeleton class="h-2 flex-1 rounded-full" />
                  <Skeleton class="h-3 w-14" />
                </div>
              {/each}
            </div>
            <div class="flex items-center gap-1">
              <Skeleton class="size-8 rounded-md" />
              <Skeleton class="size-8 rounded-md" />
            </div>
          </div>
        </div>
      {/each}
    </div>
  {:else if state === 'error'}
    <div class="border-border bg-card rounded-xl border px-4 py-10 text-center text-sm">
      <p class="text-foreground">{stateMessage || 'Failed to load machines.'}</p>
      <div class="mt-4">
        <Button variant="outline" onclick={onRetry}>Retry</Button>
      </div>
    </div>
  {:else if state === 'empty'}
    <div
      class="animate-fade-in-up border-border bg-card rounded-xl border border-dashed px-4 py-14 text-center"
    >
      <div class="bg-muted/60 mx-auto mb-4 flex size-12 items-center justify-center rounded-full">
        <Server class="text-muted-foreground size-5" />
      </div>
      <p class="text-foreground text-sm font-medium">No machines configured</p>
      <p class="text-muted-foreground mt-1 text-sm">
        Register a remote worker to make it available for orchestration and routing.
      </p>
      <div class="mt-4">
        <Button onclick={onCreate}>New machine</Button>
      </div>
    </div>
  {:else}
    <div class="space-y-4">
      <div class="relative w-full max-w-xs">
        <Search class="text-muted-foreground absolute top-2.5 left-2.5 size-3.5" />
        <Input
          value={searchQuery}
          class="h-9 pl-8 text-sm"
          placeholder="Search machines..."
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
                onOpen={() => onSelectMachine?.(machine.id)}
                onTest={() => onTest?.(machine.id)}
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
  {onDraftChange}
  onRefreshHealth={selectedMachine ? () => onRefreshHealth?.(selectedMachine.id) : undefined}
  {onSave}
/>
