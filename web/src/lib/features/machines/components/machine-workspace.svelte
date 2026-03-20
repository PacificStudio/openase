<script lang="ts">
  import { Button } from '$ui/button'
  import MachineBrowser from './machine-browser.svelte'
  import MachineEditor from './machine-editor.svelte'
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
    testing = false,
    deleting = false,
    feedback = '',
    stateMessage = '',
    editorError = '',
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
    testing?: boolean
    deleting?: boolean
    feedback?: string
    stateMessage?: string
    editorError?: string
    onSearchChange?: (value: string) => void
    onSelectMachine?: (machineId: string) => void
    onDraftChange?: (field: MachineDraftField, value: string) => void
    onCreate?: () => void
    onRetry?: () => void
    onSave?: () => void
    onTest?: () => void
    onDelete?: () => void
    onReset?: () => void
  } = $props()

  const emptyMessage = $derived(
    searchQuery.trim() ? 'No machines match the current filter.' : 'No machines registered yet.',
  )
</script>

<div class="px-6 pb-6">
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
      {#if feedback}
        <p class="mt-4 text-sm text-emerald-400">{feedback}</p>
      {/if}
      {#if stateMessage}
        <p class="text-destructive mt-3 text-sm">{stateMessage}</p>
      {/if}
    </div>
  {:else}
    <div class="grid gap-4 xl:grid-cols-[22rem_minmax(0,1fr)]">
      <MachineBrowser
        {machines}
        {selectedId}
        {searchQuery}
        {emptyMessage}
        {onSearchChange}
        onSelect={onSelectMachine}
      />

      <MachineEditor
        {mode}
        machine={selectedMachine}
        {draft}
        {snapshot}
        {probe}
        {loadingHealth}
        {saving}
        {testing}
        {deleting}
        {feedback}
        error={editorError}
        {onDraftChange}
        {onSave}
        {onTest}
        {onDelete}
        {onReset}
      />
    </div>
  {/if}
</div>
