<script lang="ts">
  import MachineBrowser from './machine-browser.svelte'
  import MachineEditor from './machine-editor.svelte'
  import type {
    MachineDraft,
    MachineDraftField,
    MachineEditorMode,
    MachineItem,
    MachineProbeResult,
    MachineSnapshot,
  } from '../types'

  let {
    orgReady,
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
    error = '',
    onSearchChange,
    onSelectMachine,
    onDraftChange,
    onSave,
    onTest,
    onDelete,
    onReset,
  }: {
    orgReady: boolean
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
    error?: string
    onSearchChange?: (value: string) => void
    onSelectMachine?: (machineId: string) => void
    onDraftChange?: (field: MachineDraftField, value: string) => void
    onSave?: () => void
    onTest?: () => void
    onDelete?: () => void
    onReset?: () => void
  } = $props()
</script>

<div class="px-6 pb-6">
  {#if !orgReady}
    <div
      class="border-border bg-card text-muted-foreground rounded-xl border border-dashed px-4 py-10 text-center text-sm"
    >
      Create an organization before managing machines.
    </div>
  {:else if loading}
    <div
      class="border-border bg-card text-muted-foreground rounded-xl border px-4 py-10 text-center text-sm"
    >
      Loading machines…
    </div>
  {:else}
    <div class="grid gap-4 xl:grid-cols-[22rem_minmax(0,1fr)]">
      <MachineBrowser
        {machines}
        {selectedId}
        {searchQuery}
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
        {error}
        {onDraftChange}
        {onSave}
        {onTest}
        {onDelete}
        {onReset}
      />
    </div>
  {/if}
</div>
