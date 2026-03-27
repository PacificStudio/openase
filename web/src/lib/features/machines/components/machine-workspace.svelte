<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Search } from '@lucide/svelte'
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

<div class="flex min-h-0 flex-1 flex-col px-6 pb-6">
  {#if state === 'no-org'}
    <div
      class="border-border bg-card text-muted-foreground rounded-xl border border-dashed px-4 py-10 text-center text-sm"
    >
      Create an organization before managing machines.
    </div>
  {:else if state === 'loading' || loading}
    <div
      class="border-border bg-card text-muted-foreground rounded-xl border px-4 py-10 text-center text-sm"
    >
      Loading machines…
    </div>
  {:else if state === 'error'}
    <div class="border-border bg-card rounded-xl border px-4 py-10 text-center text-sm">
      <p class="text-foreground">{stateMessage || 'Failed to load machines.'}</p>
      <div class="mt-4">
        <Button variant="outline" onclick={onRetry}>Retry</Button>
      </div>
    </div>
  {:else if state === 'empty'}
    <div class="border-border bg-card rounded-xl border border-dashed px-4 py-10 text-center">
      <p class="text-foreground text-sm font-medium">
        No machines configured for this organization.
      </p>
      <p class="text-muted-foreground mt-2 text-sm">
        Register a remote worker to make it available for orchestration and routing.
      </p>
      <div class="mt-4">
        <Button onclick={onCreate}>New machine</Button>
      </div>
    </div>
  {:else}
    <div class="space-y-4">
      <div
        class="flex flex-col gap-3 rounded-2xl border border-dashed px-4 py-4 md:flex-row md:items-center md:justify-between"
      >
        <div class="max-w-2xl">
          <h2 class="text-foreground text-sm font-semibold">Fleet overview</h2>
          <p class="text-muted-foreground mt-1 text-sm">
            Click any machine card to open the editor drawer. Runtime health stays read-only in the
            list; edits happen in the drawer only.
          </p>
        </div>

        <div class="relative w-full md:max-w-xs">
          <Search class="text-muted-foreground absolute top-2.5 left-2.5 size-3.5" />
          <Input
            value={searchQuery}
            class="h-9 pl-8 text-sm"
            placeholder="Search machines..."
            oninput={(event) => onSearchChange?.((event.currentTarget as HTMLInputElement).value)}
          />
        </div>
      </div>

      {#if machines.length === 0}
        <div
          class="border-border bg-card text-muted-foreground rounded-xl border border-dashed px-4 py-8 text-center text-sm"
        >
          {emptyMessage}
        </div>
      {:else}
        <div class="space-y-3">
          {#each machines as machine (machine.id)}
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
  {saving}
  {onDraftChange}
  {onSave}
/>
