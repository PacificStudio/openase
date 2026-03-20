<script lang="ts">
  import { invalidate } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import {
    createMachine,
    deleteMachine,
    getMachineResources,
    testMachineConnection,
    updateMachine,
  } from '$lib/api/openase'
  import PageHeader from '$lib/components/layout/page-header.svelte'
  import MachinePageActions from './machine-page-actions.svelte'
  import MachineWorkspace from './machine-workspace.svelte'
  import {
    createEmptyMachineDraft,
    filterMachines,
    parseMachineDraft,
    parseMachineSnapshot,
  } from '../model'
  import {
    createEditorSelectionState,
    createEmptyState,
    createListErrorState,
    createNoOrgState,
    createStartCreateState,
    type MachinesPageViewState,
  } from '../page-state'
  import type {
    MachineDraft,
    MachineDraftField,
    MachineItem,
    MachineProbeResult,
    MachinesPageData,
    MachineSnapshot,
    MachineWorkspaceState,
  } from '../types'

  let { data }: { data: MachinesPageData } = $props()

  let loading = $state(false)
  let refreshing = $state(false)
  let loadingHealth = $state(false)
  let saving = $state(false)
  let testing = $state(false)
  let deleting = $state(false)
  let workspaceState = $state<MachineWorkspaceState>('loading')
  let routeOrgId = $state('')
  let listMessage = $state('')
  let editorError = $state('')
  let feedback = $state('')
  let machines = $state<MachineItem[]>([])
  let selectedId = $state('')
  let mode = $state<'create' | 'edit'>('edit')
  let searchQuery = $state('')
  let draft = $state<MachineDraft>(createEmptyMachineDraft())
  let snapshot = $state<MachineSnapshot | null>(null)
  let probe = $state<MachineProbeResult | null>(null)

  const selectedMachine = $derived(machines.find((machine) => machine.id === selectedId) ?? null)
  const filteredMachines = $derived(filterMachines(machines, searchQuery))

  $effect(() => {
    void syncFromRouteData(data)
  })

  async function syncFromRouteData(nextData: MachinesPageData) {
    loading = false
    refreshing = false

    if (nextData.orgContext.kind === 'no-org') return applyViewState(createNoOrgState())
    if (nextData.orgContext.kind === 'error') {
      return applyViewState(createListErrorState(nextData.orgContext.message))
    }

    const nextOrgId = nextData.orgContext.org.id
    if (nextData.initialListError) {
      return applyViewState(createListErrorState(nextData.initialListError))
    }
    if (nextData.initialMachines.length === 0) {
      return applyViewState(createEmptyState(nextOrgId))
    }

    const nextMachine =
      nextData.initialMachines.find((machine) => machine.id === selectedId) ??
      nextData.initialMachines[0]
    applyViewState(
      createEditorSelectionState(nextOrgId, nextData.initialMachines, nextMachine, feedback, true),
    )
    await loadMachineResources(nextMachine.id)
  }

  function applyViewState(nextState: MachinesPageViewState) {
    routeOrgId = nextState.routeOrgId
    machines = nextState.machines
    searchQuery = nextState.searchQuery
    workspaceState = nextState.workspaceState
    listMessage = nextState.listMessage
    selectedId = nextState.selectedId
    mode = nextState.mode
    draft = nextState.draft
    snapshot = nextState.snapshot
    probe = nextState.probe
    editorError = nextState.editorError
    feedback = nextState.feedback
  }

  async function openMachine(machine: MachineItem, options: { preserveFeedback?: boolean } = {}) {
    applyViewState(
      createEditorSelectionState(routeOrgId, machines, machine, feedback, options.preserveFeedback),
    )
    await loadMachineResources(machine.id)
  }

  async function loadMachineResources(machineId: string) {
    loadingHealth = true

    try {
      const payload = await getMachineResources(machineId)
      snapshot = parseMachineSnapshot(payload.resources)
    } catch (caughtError) {
      editorError =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to load machine resources.'
    } finally {
      loadingHealth = false
    }
  }

  function startCreate(options: { preserveFeedback?: boolean } = {}) {
    if (!routeOrgId) return applyViewState(createNoOrgState())
    applyViewState(createStartCreateState(routeOrgId, machines, feedback, options.preserveFeedback))
  }

  function resetDraft() {
    if (mode === 'create') {
      draft = createEmptyMachineDraft()
      feedback = ''
      editorError = ''
      return
    }

    if (selectedMachine) {
      draft = {
        ...draft,
        ...createEditorSelectionState(routeOrgId, machines, selectedMachine).draft,
      }
      feedback = ''
      editorError = ''
    }
  }

  async function handleRefresh() {
    if (loading || refreshing) return
    if (workspaceState === 'error' || workspaceState === 'no-org') loading = true
    else refreshing = true

    try {
      await invalidate('openase:machines-page')
    } finally {
      loading = false
      refreshing = false
    }
  }

  async function handleSave() {
    const parsed = parseMachineDraft(draft)
    if (!routeOrgId || !parsed.ok) {
      editorError = parsed.ok ? 'Organization context is unavailable.' : parsed.error
      feedback = ''
      return
    }

    saving = true
    editorError = ''
    feedback = ''

    try {
      if (mode === 'create') {
        const payload = await createMachine(routeOrgId, parsed.value)
        machines = [payload.machine, ...machines]
        await openMachine(payload.machine, { preserveFeedback: true })
        feedback = 'Machine created.'
      } else if (selectedMachine) {
        const payload = await updateMachine(selectedMachine.id, parsed.value)
        machines = machines.map((machine) =>
          machine.id === payload.machine.id ? payload.machine : machine,
        )
        await openMachine(payload.machine, { preserveFeedback: true })
        feedback = 'Machine updated.'
      }
    } catch (caughtError) {
      editorError = caughtError instanceof ApiError ? caughtError.detail : 'Failed to save machine.'
    } finally {
      saving = false
    }
  }

  async function handleTest() {
    if (!selectedMachine) return
    testing = true
    editorError = ''
    feedback = ''

    try {
      const payload = await testMachineConnection(selectedMachine.id)
      machines = machines.map((machine) =>
        machine.id === payload.machine.id ? payload.machine : machine,
      )
      await openMachine(payload.machine, { preserveFeedback: true })
      probe = payload.probe
      feedback = 'Connection test completed.'
    } catch (caughtError) {
      editorError =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to run connection test.'
    } finally {
      testing = false
    }
  }

  async function handleDelete() {
    if (!selectedMachine) return
    deleting = true
    editorError = ''
    feedback = ''

    try {
      await deleteMachine(selectedMachine.id)
      machines = machines.filter((machine) => machine.id !== selectedMachine.id)
      probe = null
      snapshot = null
      feedback = 'Machine deleted.'

      const nextMachine = machines[0] ?? null
      if (nextMachine) {
        await openMachine(nextMachine, { preserveFeedback: true })
      } else {
        applyViewState(createEmptyState(routeOrgId, feedback, true))
      }
    } catch (caughtError) {
      editorError =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete machine.'
    } finally {
      deleting = false
    }
  }
</script>

{#snippet actions()}
  <MachinePageActions
    {refreshing}
    refreshDisabled={loading}
    createDisabled={!routeOrgId}
    onRefresh={() => void handleRefresh()}
    onCreate={startCreate}
  />
{/snippet}

<PageHeader
  title="Machines"
  description="Manage SSH-backed worker machines and inspect live monitor snapshots."
  {actions}
/>

<MachineWorkspace
  state={workspaceState}
  {loading}
  machines={filteredMachines}
  {selectedId}
  {searchQuery}
  {selectedMachine}
  {mode}
  {draft}
  {snapshot}
  {probe}
  {loadingHealth}
  {saving}
  {testing}
  {deleting}
  {feedback}
  stateMessage={listMessage}
  {editorError}
  onSearchChange={(value) => {
    searchQuery = value
  }}
  onSelectMachine={(machineId) => {
    const nextMachine = machines.find((machine) => machine.id === machineId)
    if (nextMachine) void openMachine(nextMachine)
  }}
  onDraftChange={(field: MachineDraftField, value: string) => {
    draft = { ...draft, [field]: value }
  }}
  onCreate={startCreate}
  onRetry={() => void handleRefresh()}
  onSave={() => void handleSave()}
  onTest={() => void handleTest()}
  onDelete={() => void handleDelete()}
  onReset={resetDraft}
/>
