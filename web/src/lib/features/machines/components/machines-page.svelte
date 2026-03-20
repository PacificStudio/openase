<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import {
    createMachine,
    deleteMachine,
    getMachineResources,
    listMachines,
    testMachineConnection,
    updateMachine,
  } from '$lib/api/openase'
  import PageHeader from '$lib/components/layout/page-header.svelte'
  import { appStore } from '$lib/stores/app.svelte'
  import MachinePageActions from './machine-page-actions.svelte'
  import MachineWorkspace from './machine-workspace.svelte'
  import {
    createEmptyMachineDraft,
    filterMachines,
    machineToDraft,
    parseMachineDraft,
    parseMachineSnapshot,
  } from '../model'
  import type {
    MachineDraft,
    MachineDraftField,
    MachineItem,
    MachineProbeResult,
    MachineSnapshot,
  } from '../types'

  let loading = $state(false)
  let refreshing = $state(false)
  let loadingHealth = $state(false)
  let saving = $state(false)
  let testing = $state(false)
  let deleting = $state(false)
  let error = $state('')
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
    const orgId = appStore.currentOrg?.id
    if (!orgId) {
      machines = []
      selectedId = ''
      mode = 'edit'
      snapshot = null
      probe = null
      draft = createEmptyMachineDraft()
      return
    }

    void loadMachines(orgId, { initial: true })
  })

  async function loadMachines(orgId: string, options: { initial?: boolean } = {}) {
    if (options.initial) {
      loading = true
    } else {
      refreshing = true
    }
    error = ''

    try {
      const payload = await listMachines(orgId)
      machines = payload.machines

      if (mode === 'create') {
        return
      }

      const nextMachine =
        payload.machines.find((machine) => machine.id === selectedId) ?? payload.machines[0] ?? null
      if (!nextMachine) {
        selectedId = ''
        draft = createEmptyMachineDraft()
        snapshot = null
        probe = null
        return
      }

      await openMachine(nextMachine)
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load machines.'
    } finally {
      loading = false
      refreshing = false
    }
  }

  async function openMachine(machine: MachineItem, options: { preserveFeedback?: boolean } = {}) {
    mode = 'edit'
    selectedId = machine.id
    draft = machineToDraft(machine)
    if (!options.preserveFeedback) {
      feedback = ''
    }
    error = ''
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
      error =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to load machine resources.'
    } finally {
      loadingHealth = false
    }
  }

  function startCreate(options: { preserveFeedback?: boolean } = {}) {
    mode = 'create'
    selectedId = ''
    draft = createEmptyMachineDraft()
    probe = null
    snapshot = null
    if (!options.preserveFeedback) {
      feedback = ''
    }
    error = ''
  }

  function resetDraft() {
    if (mode === 'create') {
      draft = createEmptyMachineDraft()
      feedback = ''
      error = ''
      return
    }

    if (selectedMachine) {
      draft = machineToDraft(selectedMachine)
      feedback = ''
      error = ''
    }
  }

  async function handleSave() {
    const orgId = appStore.currentOrg?.id
    const parsed = parseMachineDraft(draft)
    if (!orgId || !parsed.ok) {
      error = parsed.ok ? 'Organization context is unavailable.' : parsed.error
      feedback = ''
      return
    }

    saving = true
    error = ''
    feedback = ''

    try {
      if (mode === 'create') {
        const payload = await createMachine(orgId, parsed.value)
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
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to save machine.'
    } finally {
      saving = false
    }
  }

  async function handleTest() {
    if (!selectedMachine) {
      return
    }

    testing = true
    error = ''
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
      error =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to run connection test.'
    } finally {
      testing = false
    }
  }

  async function handleDelete() {
    if (!selectedMachine) {
      return
    }

    deleting = true
    error = ''
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
        startCreate({ preserveFeedback: true })
      }
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete machine.'
    } finally {
      deleting = false
    }
  }
</script>

{#snippet actions()}
  <MachinePageActions
    {refreshing}
    onRefresh={() => {
      const orgId = appStore.currentOrg?.id
      if (orgId) {
        void loadMachines(orgId)
      }
    }}
    onCreate={startCreate}
  />
{/snippet}

<PageHeader
  title="Machines"
  description="Manage SSH-backed worker machines and inspect live monitor snapshots."
  {actions}
/>

<MachineWorkspace
  orgReady={Boolean(appStore.currentOrg)}
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
  {error}
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
  onSave={() => void handleSave()}
  onTest={() => void handleTest()}
  onDelete={() => void handleDelete()}
  onReset={resetDraft}
/>
