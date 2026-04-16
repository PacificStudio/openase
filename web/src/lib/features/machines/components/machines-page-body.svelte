<script lang="ts">
  import { PageScaffold } from '$lib/components/layout'
  import MachinePageActions from './machine-page-actions.svelte'
  import MachineWorkspace from './machine-workspace.svelte'
  import type {
    MachineDraft,
    MachineDraftField,
    MachineEditorMode,
    MachineItem,
    MachineProbeResult,
    MachineSnapshot,
    MachineWorkspaceState,
  } from '../types'

  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    routeOrgId = '',
    loading = false,
    refreshing = false,
    workspaceState,
    listMessage = '',
    machines,
    selectedId = '',
    searchQuery = '',
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
    onRefresh,
    onCreate,
    onWizardCreated,
    onSearchChange,
    onSelectMachine,
    onDraftChange,
    onRetry,
    onRefreshHealth,
    onSave,
    onTest,
    onDelete,
    onReset,
  }: {
    routeOrgId?: string
    loading?: boolean
    refreshing?: boolean
    workspaceState: MachineWorkspaceState
    listMessage?: string
    machines: MachineItem[]
    selectedId?: string
    searchQuery?: string
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
    onRefresh?: () => void
    onCreate?: () => void
    onWizardCreated?: (machine: MachineItem) => void
    onSearchChange?: (value: string) => void
    onSelectMachine?: (machineId: string) => void
    onDraftChange?: (field: MachineDraftField, value: string) => void
    onRetry?: () => void
    onRefreshHealth?: (machineId: string) => void
    onSave?: () => void
    onTest?: (machineId: string) => void
    onDelete?: (machineId: string) => void
    onReset?: (machineId?: string) => void
  } = $props()
  const t = i18nStore.t
</script>

{#snippet actions()}
  <div data-tour="machines-actions">
    <MachinePageActions
      {refreshing}
      refreshDisabled={loading}
      createDisabled={!routeOrgId}
      {onRefresh}
      {onCreate}
    />
  </div>
{/snippet}

<PageScaffold
  title={t('machines.page.title')}
  description={t('machines.page.description')}
  helpSection="machines"
  variant="workspace"
  {actions}
>
  <MachineWorkspace
    state={workspaceState}
    {loading}
    {machines}
    {selectedId}
    {searchQuery}
    {selectedMachine}
    {mode}
    {draft}
    {snapshot}
    {probe}
    {loadingHealth}
    {refreshingHealthMachineId}
    {saving}
    {testingMachineId}
    {deletingMachineId}
    bind:editorOpen
    stateMessage={listMessage}
    organizationId={routeOrgId || null}
    {onSearchChange}
    {onSelectMachine}
    {onDraftChange}
    {onCreate}
    {onWizardCreated}
    {onRetry}
    {onRefreshHealth}
    {onSave}
    {onTest}
    {onDelete}
    {onReset}
  />
</PageScaffold>
