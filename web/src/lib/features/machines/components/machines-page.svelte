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
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import MachinesPageBody from './machines-page-body.svelte'
  import {
    createEmptyMachineDraft,
    filterMachines,
    machineToDraft,
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
    MachineItem,
    MachineProbeResult,
    MachineSnapshot,
    MachineWorkspaceState,
  } from '../types'

  let loading = $state(false)
  let refreshing = $state(false)
  let loadingHealth = $state(false)
  let saving = $state(false)
  let testingMachineId = $state('')
  let deletingMachineId = $state('')
  let editorOpen = $state(false)
  let workspaceState = $state<MachineWorkspaceState>('loading')
  let routeOrgId = $state('')
  let listMessage = $state('')
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
    const currentOrg = appStore.currentOrg
    if (!currentOrg) {
      editorOpen = false
      loading = false
      refreshing = false
      applyViewState(createNoOrgState())
      return
    }

    let cancelled = false
    void loadMachineList(currentOrg.id, { background: false, cancelled: () => cancelled })

    return () => {
      cancelled = true
    }
  })

  async function syncFromMachineList(
    orgId: string,
    nextMachines: MachineItem[],
    nextListError: string | null,
  ) {
    if (nextListError) {
      editorOpen = false
      return applyViewState(createListErrorState(nextListError))
    }
    if (nextMachines.length === 0) {
      editorOpen = false
      return applyViewState(createEmptyState(orgId))
    }

    const nextMachine =
      nextMachines.find((machine) => machine.id === selectedId) ?? nextMachines[0]
    applyViewState({
      ...createEditorSelectionState(orgId, nextMachines, nextMachine),
      searchQuery,
    })
    await loadMachineResources(nextMachine.id)
  }

  async function loadMachineList(
    orgId: string,
    options: {
      background: boolean
      cancelled?: () => boolean
    },
  ) {
    loading = !options.background
    refreshing = options.background

    try {
      const payload = await listMachines(orgId)
      if (options.cancelled?.()) return
      await syncFromMachineList(orgId, payload.machines ?? [], null)
    } catch (caughtError) {
      if (options.cancelled?.()) return
      if (options.background && machines.length > 0) {
        toastStore.error(
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to refresh machines.',
        )
      } else {
        await syncFromMachineList(orgId, [], 'Failed to load machines.')
      }
    } finally {
      if (!options.cancelled?.()) {
        loading = false
        refreshing = false
      }
    }
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
  }

  async function openMachine(machine: MachineItem, openEditor = true) {
    applyViewState({
      ...createEditorSelectionState(routeOrgId, machines, machine),
      searchQuery,
    })
    editorOpen = openEditor
    await loadMachineResources(machine.id)
  }

  async function loadMachineResources(machineId: string) {
    loadingHealth = true

    try {
      const payload = await getMachineResources(machineId)
      snapshot = parseMachineSnapshot(payload.resources)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to load machine resources.',
      )
    } finally {
      loadingHealth = false
    }
  }

  function startCreate() {
    if (!routeOrgId) return applyViewState(createNoOrgState())
    applyViewState({ ...createStartCreateState(routeOrgId, machines), searchQuery })
    editorOpen = true
  }

  function resetDraft(machineId?: string) {
    if (machineId && machineId !== selectedId) return

    if (mode === 'create') {
      draft = createEmptyMachineDraft()
      return
    }

    if (selectedMachine) {
      draft = machineToDraft(selectedMachine)
    }
  }

  async function handleRefresh() {
    if (loading || refreshing) return
    if (!routeOrgId) return
    await loadMachineList(routeOrgId, { background: workspaceState === 'ready' })
  }

  async function handleSave() {
    const parsed = parseMachineDraft(draft)
    if (!routeOrgId || !parsed.ok) {
      toastStore.error(parsed.ok ? 'Organization context is unavailable.' : parsed.error)
      return
    }

    saving = true

    try {
      if (mode === 'create') {
        const payload = await createMachine(routeOrgId, parsed.value)
        machines = [payload.machine, ...machines]
        await openMachine(payload.machine, true)
        toastStore.success('Machine created.')
      } else if (selectedMachine) {
        const payload = await updateMachine(selectedMachine.id, parsed.value)
        machines = machines.map((machine) =>
          machine.id === payload.machine.id ? payload.machine : machine,
        )
        await openMachine(payload.machine, true)
        toastStore.success('Machine updated.')
      }
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to save machine.',
      )
    } finally {
      saving = false
    }
  }

  async function handleTest(machineId: string) {
    const machine = machines.find((item) => item.id === machineId)
    if (!machine) return
    testingMachineId = machineId

    try {
      const payload = await testMachineConnection(machineId)
      machines = machines.map((machine) =>
        machine.id === payload.machine.id ? payload.machine : machine,
      )
      if (selectedId === machineId) {
        snapshot = parseMachineSnapshot(payload.machine.resources)
        probe = payload.probe
      }
      toastStore.success('Connection test completed.')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to run connection test.',
      )
    } finally {
      testingMachineId = ''
    }
  }

  async function handleDelete(machineId: string) {
    const machine = machines.find((item) => item.id === machineId)
    if (!machine) return
    deletingMachineId = machineId

    try {
      await deleteMachine(machineId)
      const nextMachines = machines.filter((item) => item.id !== machineId)
      machines = nextMachines
      toastStore.success('Machine deleted.')

      if (selectedId === machineId) {
        probe = null
        snapshot = null
        editorOpen = false

        const nextMachine = nextMachines[0] ?? null
        if (nextMachine) {
          applyViewState({
            ...createEditorSelectionState(routeOrgId, nextMachines, nextMachine),
            searchQuery,
          })
        } else {
          applyViewState(createEmptyState(routeOrgId))
        }
      }
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete machine.',
      )
    } finally {
      deletingMachineId = ''
    }
  }
</script>

<MachinesPageBody
  {routeOrgId}
  {loading}
  {refreshing}
  {workspaceState}
  {listMessage}
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
  {testingMachineId}
  {deletingMachineId}
  bind:editorOpen
  onRefresh={() => void handleRefresh()}
  onCreate={startCreate}
  onSearchChange={(value) => {
    searchQuery = value
  }}
  onSelectMachine={(machineId) => {
    const nextMachine = machines.find((machine) => machine.id === machineId)
    if (nextMachine) void openMachine(nextMachine)
  }}
  onDraftChange={(field, value) => {
    draft = { ...draft, [field]: value }
  }}
  onRetry={() => void handleRefresh()}
  onSave={() => void handleSave()}
  onTest={(machineId) => void handleTest(machineId)}
  onDelete={(machineId) => void handleDelete(machineId)}
  onReset={resetDraft}
/>
