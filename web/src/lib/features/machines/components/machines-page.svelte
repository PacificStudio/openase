<!-- eslint-disable max-lines -->
<script lang="ts">
  import { untrack } from 'svelte'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { subscribeOrganizationMachineEvents } from '$lib/features/org-events'
  import { syncMachinesPageProjectAIFocus } from './machines-page-focus'
  import MachinesPageBody from './machines-page-body.svelte'
  import { syncMachineListState } from './machines-page-state-sync'
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
  import {
    markMachinesPageCacheDirty,
    readMachineSnapshotCache,
    readMachinesPageCache,
    writeMachineSnapshotCache,
    writeMachinesPageCache,
  } from '../machines-page-cache'
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
  let activeOrgId = $state(''),
    listRequestVersion = $state(0),
    snapshotRequestVersion = $state(0),
    queuedReload = $state(false),
    reloadInFlight = $state(false)
  const projectAIFocusOwner = 'machines-page'
  const currentOrgId = $derived(appStore.currentOrg?.id ?? '')
  const currentProjectId = $derived(appStore.currentProject?.id ?? '')
  const selectedMachine = $derived(machines.find((machine) => machine.id === selectedId) ?? null),
    filteredMachines = $derived(filterMachines(machines, searchQuery))
  $effect(() => {
    if (!currentOrgId) {
      editorOpen = false
      loading = false
      refreshing = false
      applyViewState(createNoOrgState())
      return
    }
    const orgId = currentOrgId
    activeOrgId = orgId
    queuedReload = false
    reloadInFlight = false
    let cancelled = false
    const cachedPage = readMachinesPageCache(orgId)
    if (cachedPage) {
      const nextState = syncMachineListState({
        orgId,
        nextMachines: cachedPage.snapshot.machines,
        nextListError: null,
        selectedId: cachedPage.snapshot.selectedId,
        searchQuery: cachedPage.snapshot.searchQuery,
      })
      editorOpen = nextState.selectedMachineId !== null
      applyViewState(nextState.viewState)
      if (nextState.selectedMachineId) {
        const cachedSnapshot = readMachineSnapshotCache(orgId, nextState.selectedMachineId)
        if (cachedSnapshot) {
          snapshot = cachedSnapshot.snapshot
        }
        if (cachedPage.dirty || cachedSnapshot?.dirty) {
          untrack(() => {
            void loadMachineList(orgId, { background: true, cancelled: () => cancelled })
          })
        }
      } else if (cachedPage.dirty) {
        untrack(() => {
          void loadMachineList(orgId, { background: true, cancelled: () => cancelled })
        })
      }
    } else {
      untrack(() => {
        void loadMachineList(orgId, { background: false, cancelled: () => cancelled })
      })
    }
    const disconnect = subscribeOrganizationMachineEvents(orgId, () => {
      markMachinesPageCacheDirty(orgId)
      requestReload(orgId, () => cancelled)
    })
    return () => {
      cancelled = true
      if (activeOrgId === orgId) {
        activeOrgId = ''
      }
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
    const requestVersion = ++listRequestVersion
    loading = !options.background
    refreshing = options.background
    try {
      const nextMachines = await loadMachines(orgId)
      if (options.cancelled?.() || activeOrgId !== orgId || requestVersion !== listRequestVersion)
        return
      const nextState = syncMachineListState({
        orgId,
        nextMachines,
        nextListError: null,
        selectedId,
        searchQuery,
      })
      editorOpen = nextState.selectedMachineId !== null
      applyViewState(nextState.viewState)
      persistMachinesPageCache(orgId)
      if (nextState.selectedMachineId) await loadMachineResources(nextState.selectedMachineId)
    } catch (caughtError) {
      if (options.cancelled?.() || activeOrgId !== orgId || requestVersion !== listRequestVersion)
        return
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
      if (!options.cancelled?.() && activeOrgId === orgId && requestVersion === listRequestVersion)
        loading = false
      if (!options.cancelled?.() && activeOrgId === orgId && requestVersion === listRequestVersion)
        refreshing = false
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
    persistMachinesPageCache(routeOrgId)
    await loadMachineResources(machine.id)
  }
  async function loadMachineResources(machineId: string) {
    const orgId = routeOrgId
    if (!orgId) return
    const cachedSnapshot = readMachineSnapshotCache(orgId, machineId)
    if (cachedSnapshot) {
      snapshot = cachedSnapshot.snapshot
      if (!cachedSnapshot.dirty) {
        loadingHealth = false
        return
      }
    }

    const requestVersion = ++snapshotRequestVersion
    loadingHealth = true
    try {
      const nextSnapshot = await loadMachineSnapshot(machineId)
      if (
        activeOrgId !== orgId ||
        requestVersion !== snapshotRequestVersion ||
        selectedId !== machineId
      ) {
        return
      }
      snapshot = nextSnapshot
      writeMachineSnapshotCache(orgId, machineId, nextSnapshot)
    } catch (caughtError) {
      toastStore.error(machineErrorMessage(caughtError, 'Failed to load machine resources.'))
    } finally {
      if (
        activeOrgId === orgId &&
        requestVersion === snapshotRequestVersion &&
        selectedId === machineId
      )
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
      persistMachinesPageCache(routeOrgId)
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
        writeMachineSnapshotCache(routeOrgId, machineId, payload.snapshot)
      }
      persistMachinesPageCache(routeOrgId)
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
        writeMachineSnapshotCache(routeOrgId, machineId, payload.snapshot)
      }
      persistMachinesPageCache(routeOrgId)
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
      persistMachinesPageCache(routeOrgId)
    } catch (caughtError) {
      toastStore.error(machineErrorMessage(caughtError, 'Failed to delete machine.'))
    } finally {
      deletingMachineId = ''
    }
  }
  $effect(() => {
    if (routeOrgId) {
      persistMachinesPageCache(routeOrgId)
    }
  })
  $effect(() => {
    return syncMachinesPageProjectAIFocus({
      clearFocus: (owner) => appStore.clearProjectAssistantFocus(owner),
      setFocus: (owner, focus, priority) =>
        appStore.setProjectAssistantFocus(owner, focus, priority),
      owner: projectAIFocusOwner,
      projectId: currentProjectId,
      editorOpen,
      mode,
      selectedMachine,
      snapshot,
    })
  })

  function persistMachinesPageCache(orgId: string) {
    if (!orgId) return
    writeMachinesPageCache(orgId, {
      machines,
      selectedId,
      searchQuery,
    })
  }

  const requestReload = (orgId: string, cancelled?: () => boolean) => {
    queuedReload = true
    void drainReloadQueue(orgId, cancelled)
  }

  async function drainReloadQueue(orgId: string, cancelled?: () => boolean) {
    if (!queuedReload || reloadInFlight || activeOrgId !== orgId || cancelled?.()) {
      return
    }

    reloadInFlight = true
    queuedReload = false
    try {
      await loadMachineList(orgId, { background: true, cancelled })
    } finally {
      reloadInFlight = false
      if (queuedReload && activeOrgId === orgId && !cancelled?.()) {
        void drainReloadQueue(orgId, cancelled)
      }
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
