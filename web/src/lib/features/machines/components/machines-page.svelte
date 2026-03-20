<script lang="ts">
  import { invalidate } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import { createMachine, deleteMachine, getMachineResources, testMachineConnection, updateMachine } from '$lib/api/openase'
  import PageHeader from '$lib/components/layout/page-header.svelte'
  import MachinePageActions from './machine-page-actions.svelte'
  import MachineWorkspace from './machine-workspace.svelte'
  import { createEmptyMachineDraft, filterMachines, machineToDraft, parseMachineDraft, parseMachineSnapshot } from '../model'
  import type { MachineDraft, MachineDraftField, MachineItem, MachineProbeResult, MachinesPageData, MachineSnapshot, MachineWorkspaceState } from '../types'

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
    routeOrgId = nextData.orgContext.kind === 'ready' ? nextData.orgContext.org.id : ''
    loading = false
    refreshing = false

    if (nextData.orgContext.kind === 'no-org') return applyNoOrgState()
    if (nextData.orgContext.kind === 'error') return applyListErrorState(nextData.orgContext.message)

    machines = nextData.initialMachines
    listMessage = nextData.initialListError ?? ''
    editorError = ''

    if (nextData.initialListError) return applyListErrorState(nextData.initialListError)
    if (nextData.initialMachines.length === 0) return applyEmptyState()

    workspaceState = 'ready'
    const nextMachine =
      nextData.initialMachines.find((machine) => machine.id === selectedId) ??
      nextData.initialMachines[0]
    await openMachine(nextMachine, { preserveFeedback: true })
  }

  function resetEditorState(options: { preserveFeedback?: boolean } = {}) {
    selectedId = ''
    mode = 'edit'
    draft = createEmptyMachineDraft()
    snapshot = null
    probe = null
    editorError = ''
    if (!options.preserveFeedback) feedback = ''
  }

  function applyNoOrgState() {
    routeOrgId = ''
    machines = []
    listMessage = ''
    searchQuery = ''
    workspaceState = 'no-org'
    resetEditorState()
  }

  function applyListErrorState(message: string, options: { preserveFeedback?: boolean } = {}) {
    machines = []
    listMessage = message
    searchQuery = ''
    workspaceState = 'error'
    resetEditorState(options)
  }

  function applyEmptyState(options: { preserveFeedback?: boolean } = {}) {
    listMessage = ''
    searchQuery = ''
    machines = []
    workspaceState = 'empty'
    resetEditorState(options)
  }

  async function openMachine(machine: MachineItem, options: { preserveFeedback?: boolean } = {}) {
    workspaceState = 'ready'
    mode = 'edit'
    selectedId = machine.id
    draft = machineToDraft(machine)
    editorError = ''
    if (!options.preserveFeedback) feedback = ''
    probe = null
    snapshot = parseMachineSnapshot(machine.resources)
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
    if (!routeOrgId) return applyNoOrgState()
    workspaceState = 'ready'
    mode = 'create'
    selectedId = ''
    draft = createEmptyMachineDraft()
    probe = null
    snapshot = null
    editorError = ''
    if (!options.preserveFeedback) feedback = ''
  }

  function resetDraft() {
    if (mode === 'create') {
      draft = createEmptyMachineDraft()
      feedback = ''
      editorError = ''
      return
    }

    if (selectedMachine) {
      draft = machineToDraft(selectedMachine)
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
        workspaceState = 'ready'
        await openMachine(payload.machine, { preserveFeedback: true })
        feedback = 'Machine created.'
      } else if (selectedMachine) {
        const payload = await updateMachine(selectedMachine.id, parsed.value)
        machines = machines.map((machine) =>
          machine.id === payload.machine.id ? payload.machine : machine,
        )
        workspaceState = 'ready'
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
        applyEmptyState({ preserveFeedback: true })
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

<PageHeader title="Machines" description="Manage SSH-backed worker machines and inspect live monitor snapshots." {actions} />

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
    if (nextMachine) {
      void openMachine(nextMachine)
    }
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
