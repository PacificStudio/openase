<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { syncMachinesPageProjectAIFocus } from './machines-page-focus'
  import MachinesPageBody from './machines-page-body.svelte'
  import { syncMachineListState } from './machines-page-state-sync'
  import { connectMachinesPageStream } from './machines-page-streams'
  import {
    loadMachineSnapshot,
    loadMachines,
    machineErrorMessage,
    removeMachine,
    runMachineHealthRefresh,
    runMachineConnectionTest,
    saveMachine,
  } from './machines-page-api'
  import {
    createEmptyMachineDraft,
    filterMachines,
    machineToDraft,
    parseMachineDraft,
  } from '../model'
  import {
    createEditorSelectionState,
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
  let loading = $state(false),
    refreshing = $state(false),
    loadingHealth = $state(false),
    saving = $state(false),
    editorOpen = $state(false)
  let refreshingHealthMachineId = $state(''),
    testingMachineId = $state(''),
    deletingMachineId = $state('')
  let workspaceState = $state<MachineWorkspaceState>('loading'),
    routeOrgId = $state(''),
    listMessage = $state(''),
    machines = $state<MachineItem[]>([]),
    selectedId = $state(''),
    mode = $state<'create' | 'edit'>('edit'),
    searchQuery = $state('')
  let draft = $state<MachineDraft>(createEmptyMachineDraft()),
    snapshot = $state<MachineSnapshot | null>(null),
    probe = $state<MachineProbeResult | null>(null)
  const projectAIFocusOwner = 'machines-page'
  const selectedMachine = $derived(machines.find((machine) => machine.id === selectedId) ?? null),
    filteredMachines = $derived(filterMachines(machines, searchQuery))
  $effect(() => {
    const currentOrg = appStore.currentOrg
    if (!currentOrg) {
      editorOpen = false
      loading = false
      refreshing = false
      applyViewState(createNoOrgState())
      return
    }
    const orgId = currentOrg.id
    let cancelled = false
    void loadMachineList(orgId, { background: false, cancelled: () => cancelled })
    const disconnect = connectMachinesPageStream(orgId, () => {
      void loadMachineList(orgId, { background: true, cancelled: () => cancelled })
    })
    return () => {
      cancelled = true
      disconnect()
    }
  })
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
      const nextMachines = await loadMachines(orgId)
      if (options.cancelled?.()) return
      const nextState = syncMachineListState({
        orgId,
        nextMachines,
        nextListError: null,
        selectedId,
        searchQuery,
      })
      editorOpen = nextState.selectedMachineId !== null
      applyViewState(nextState.viewState)
      if (nextState.selectedMachineId) await loadMachineResources(nextState.selectedMachineId)
    } catch (caughtError) {
      if (options.cancelled?.()) return
      if (options.background && machines.length > 0) {
        toastStore.error(machineErrorMessage(caughtError, 'Failed to refresh machines.'))
      } else {
        const nextState = syncMachineListState({
          orgId,
          nextMachines: [],
          nextListError: 'Failed to load machines.',
          selectedId,
          searchQuery,
        })
        editorOpen = false
        applyViewState(nextState.viewState)
      }
    } finally {
      if (!options.cancelled?.()) loading = false
      if (!options.cancelled?.()) refreshing = false
    }
  }
  function applyViewState(nextState: MachinesPageViewState) {
    ;({
      routeOrgId,
      machines,
      searchQuery,
      workspaceState,
      listMessage,
      selectedId,
      mode,
      draft,
      snapshot,
      probe,
    } = nextState)
  }
  async function openMachine(machine: MachineItem, openEditor = true) {
    applyViewState({ ...createEditorSelectionState(routeOrgId, machines, machine), searchQuery })
    editorOpen = openEditor
    await loadMachineResources(machine.id)
  }
  async function loadMachineResources(machineId: string) {
    loadingHealth = true
    try {
      snapshot = await loadMachineSnapshot(machineId)
    } catch (caughtError) {
      toastStore.error(machineErrorMessage(caughtError, 'Failed to load machine resources.'))
    } finally {
      loadingHealth = false
    }
  }
  function startCreate() {
    if (!routeOrgId) return void applyViewState(createNoOrgState())
    applyViewState({ ...createStartCreateState(routeOrgId, machines), searchQuery })
    editorOpen = true
  }
  function resetDraft(machineId?: string) {
    if (machineId && machineId !== selectedId) return
    if (mode === 'create') return void (draft = createEmptyMachineDraft())
    if (selectedMachine) draft = machineToDraft(selectedMachine)
  }
  async function handleRefresh() {
    if (loading || refreshing || !routeOrgId) return
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
      const result = await saveMachine(routeOrgId, selectedMachine, mode, parsed.value)
      machines =
        mode === 'create'
          ? [result.machine, ...machines]
          : machines.map((machine) => (machine.id === result.machine.id ? result.machine : machine))
      await openMachine(result.machine, true)
      toastStore.success(result.feedback)
    } catch (caughtError) {
      toastStore.error(machineErrorMessage(caughtError, 'Failed to save machine.'))
    } finally {
      saving = false
    }
  }
  async function handleTest(machineId: string) {
    const machine = machines.find((item) => item.id === machineId)
    if (!machine) return
    testingMachineId = machineId
    try {
      const payload = await runMachineConnectionTest(machineId)
      machines = machines.map((machine) =>
        machine.id === payload.machine.id ? payload.machine : machine,
      )
      if (selectedId === machineId) {
        snapshot = payload.snapshot
        probe = payload.probe
      }
      toastStore.success('Connection test completed.')
    } catch (caughtError) {
      toastStore.error(machineErrorMessage(caughtError, 'Failed to run connection test.'))
    } finally {
      testingMachineId = ''
    }
  }
  async function handleRefreshHealth(machineId: string) {
    const machine = machines.find((item) => item.id === machineId)
    if (!machine) return
    refreshingHealthMachineId = machineId
    try {
      const payload = await runMachineHealthRefresh(machineId)
      machines = machines.map((item) => (item.id === payload.machine.id ? payload.machine : item))
      if (selectedId === machineId) {
        snapshot = payload.snapshot
      }
      toastStore.success('Machine health checks refreshed.')
    } catch (caughtError) {
      toastStore.error(machineErrorMessage(caughtError, 'Failed to refresh machine health.'))
    } finally {
      refreshingHealthMachineId = ''
    }
  }
  async function handleDelete(machineId: string) {
    const machine = machines.find((item) => item.id === machineId)
    if (!machine) return
    deletingMachineId = machineId
    try {
      await removeMachine(machineId)
      const nextMachines = machines.filter((item) => item.id !== machineId)
      machines = nextMachines
      toastStore.success('Machine deleted.')

      if (selectedId === machineId) {
        probe = null
        snapshot = null
        editorOpen = false
        applyViewState(
          syncMachineListState({
            orgId: routeOrgId,
            nextMachines,
            nextListError: null,
            selectedId: '',
            searchQuery,
          }).viewState,
        )
      }
    } catch (caughtError) {
      toastStore.error(machineErrorMessage(caughtError, 'Failed to delete machine.'))
    } finally {
      deletingMachineId = ''
    }
  }
  $effect(() => {
    return syncMachinesPageProjectAIFocus({
      clearFocus: (owner) => appStore.clearProjectAssistantFocus(owner),
      setFocus: (owner, focus, priority) =>
        appStore.setProjectAssistantFocus(owner, focus, priority),
      owner: projectAIFocusOwner,
      projectId: appStore.currentProject?.id ?? '',
      editorOpen,
      mode,
      selectedMachine,
      snapshot,
    })
  })
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
  {refreshingHealthMachineId}
  {saving}
  {testingMachineId}
  {deletingMachineId}
  bind:editorOpen
  onRefresh={() => void handleRefresh()}
  onCreate={startCreate}
  onSearchChange={(value) => (searchQuery = value)}
  onSelectMachine={(machineId) => {
    const nextMachine = machines.find((machine) => machine.id === machineId)
    if (nextMachine) void openMachine(nextMachine)
  }}
  onDraftChange={(field, value) => (draft = { ...draft, [field]: value })}
  onRetry={() => void handleRefresh()}
  onRefreshHealth={(machineId) => void handleRefreshHealth(machineId)}
  onSave={() => void handleSave()}
  onTest={(machineId) => void handleTest(machineId)}
  onDelete={(machineId) => void handleDelete(machineId)}
  onReset={resetDraft}
/>
