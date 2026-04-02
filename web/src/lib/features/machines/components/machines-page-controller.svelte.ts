import { untrack } from 'svelte'
import { appStore } from '$lib/stores/app.svelte'
import { toastStore } from '$lib/stores/toast.svelte'
import { subscribeOrganizationMachineEvents } from '$lib/features/org-events'
import { syncMachinesPageProjectAIFocus } from './machines-page-focus'
import { syncMachineListState } from './machines-page-state-sync'
import { saveMachine, machineErrorMessage } from './machines-page-api'
import { createEmptyMachineDraft, filterMachines, parseMachineDraft } from '../model'
import {
  createEditorSelectionState,
  createNoOrgState,
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
  writeMachinesPageCache,
} from '../machines-page-cache'
import {
  type MachinesPageControllerOpsState,
  type MachinesPageControllerView,
  createMachinesPageControllerOpsState,
  handleMachineDelete,
  handleMachineHealthRefresh,
  handleMachineTest,
  loadMachineResources,
  resetMachineDraft,
  startMachineCreate,
} from './machines-page-controller-ops'
import {
  loadMachineList,
  requestMachineReload,
  type MachinesPageControllerLoadState,
} from './machines-page-controller-load'
export function createMachinesPageController() {
  let loading = $state(false)
  let refreshing = $state(false)
  let loadingHealth = $state(false)
  let saving = $state(false)
  let editorOpen = $state(false)
  let refreshingHealthMachineId = $state('')
  let testingMachineId = $state('')
  let deletingMachineId = $state('')
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
  let activeOrgId = $state('')
  let listRequestVersion = $state(0)
  let snapshotRequestVersion = $state(0)
  let queuedReload = $state(false)
  let reloadInFlight = $state(false)

  const currentOrgId = $derived(appStore.currentOrg?.id ?? '')
  const currentProjectId = $derived(appStore.currentProject?.id ?? '')
  const selectedMachine = $derived(machines.find((machine) => machine.id === selectedId) ?? null)
  const filteredMachines = $derived(filterMachines(machines, searchQuery))
  const controllerState: MachinesPageControllerOpsState = createMachinesPageControllerOpsState({
    getRouteOrgId: () => routeOrgId,
    getSelectedId: () => selectedId,
    getSearchQuery: () => searchQuery,
    getMachines: () => machines,
    setMachines: (value) => (machines = value),
    getSelectedMachine: () => selectedMachine,
    getMode: () => mode,
    getDraft: () => draft,
    setDraft: (value) => (draft = value),
    getSnapshot: () => snapshot,
    setSnapshot: (value) => (snapshot = value),
    getProbe: () => probe,
    setProbe: (value) => (probe = value),
    getEditorOpen: () => editorOpen,
    setEditorOpen: (value) => (editorOpen = value),
    getActiveOrgId: () => activeOrgId,
    getSnapshotRequestVersion: () => snapshotRequestVersion,
    setSnapshotRequestVersion: (value) => (snapshotRequestVersion = value),
    getLoadingHealth: () => loadingHealth,
    setLoadingHealth: (value) => (loadingHealth = value),
    getTestingMachineId: () => testingMachineId,
    setTestingMachineId: (value) => (testingMachineId = value),
    getRefreshingHealthMachineId: () => refreshingHealthMachineId,
    setRefreshingHealthMachineId: (value) => (refreshingHealthMachineId = value),
    getDeletingMachineId: () => deletingMachineId,
    setDeletingMachineId: (value) => (deletingMachineId = value),
    applyViewState,
    persistMachinesPageCache,
  })
  const loadState: MachinesPageControllerLoadState = {
    ...controllerState,
    get loading() {
      return loading
    },
    set loading(value) {
      loading = value
    },
    get refreshing() {
      return refreshing
    },
    set refreshing(value) {
      refreshing = value
    },
    get activeOrgId() {
      return activeOrgId
    },
    get listRequestVersion() {
      return listRequestVersion
    },
    set listRequestVersion(value) {
      listRequestVersion = value
    },
    get queuedReload() {
      return queuedReload
    },
    set queuedReload(value) {
      queuedReload = value
    },
    get reloadInFlight() {
      return reloadInFlight
    },
    set reloadInFlight(value) {
      reloadInFlight = value
    },
  }

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
            void loadMachineList(loadState, orgId, { background: true, cancelled: () => cancelled })
          })
        }
      } else if (cachedPage.dirty) {
        untrack(() => {
          void loadMachineList(loadState, orgId, { background: true, cancelled: () => cancelled })
        })
      }
    } else {
      untrack(() => {
        void loadMachineList(loadState, orgId, { background: false, cancelled: () => cancelled })
      })
    }

    const disconnect = subscribeOrganizationMachineEvents(orgId, () => {
      markMachinesPageCacheDirty(orgId)
      requestMachineReload(loadState, orgId, () => cancelled)
    })

    return () => {
      cancelled = true
      if (activeOrgId === orgId) {
        activeOrgId = ''
      }
      disconnect()
    }
  })

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

  async function openMachine(machine: MachineItem, openEditorState = true) {
    applyViewState({ ...createEditorSelectionState(routeOrgId, machines, machine), searchQuery })
    editorOpen = openEditorState
    persistMachinesPageCache(routeOrgId)
    const cachedSnapshot = readMachineSnapshotCache(routeOrgId, machine.id)
    if (cachedSnapshot) {
      snapshot = cachedSnapshot.snapshot
      if (!cachedSnapshot.dirty) {
        loadingHealth = false
        return
      }
    }
    await loadMachineResources(controllerState, machine.id)
  }

  async function handleRefresh() {
    if (loading || refreshing || !routeOrgId) return
    await loadMachineList(loadState, routeOrgId, { background: workspaceState === 'ready' })
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
      owner: 'machines-page',
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

  const view: MachinesPageControllerView = {
    get routeOrgId() {
      return routeOrgId
    },
    get loading() {
      return loading
    },
    get refreshing() {
      return refreshing
    },
    get workspaceState() {
      return workspaceState
    },
    get listMessage() {
      return listMessage
    },
    get machines() {
      return machines
    },
    get filteredMachines() {
      return filteredMachines
    },
    get selectedId() {
      return selectedId
    },
    get selectedMachine() {
      return selectedMachine
    },
    get mode() {
      return mode
    },
    get draft() {
      return draft
    },
    set draft(value) {
      draft = value
    },
    get snapshot() {
      return snapshot
    },
    get probe() {
      return probe
    },
    get loadingHealth() {
      return loadingHealth
    },
    get refreshingHealthMachineId() {
      return refreshingHealthMachineId
    },
    get saving() {
      return saving
    },
    get testingMachineId() {
      return testingMachineId
    },
    get deletingMachineId() {
      return deletingMachineId
    },
    get editorOpen() {
      return editorOpen
    },
    set editorOpen(value) {
      editorOpen = value
    },
    get searchQuery() {
      return searchQuery
    },
    set searchQuery(value) {
      searchQuery = value
    },
    handleRefresh,
    startCreate: () => startMachineCreate(controllerState),
    openMachine,
    handleRefreshHealth: (machineId: string) =>
      handleMachineHealthRefresh(controllerState, machineId),
    handleSave,
    handleTest: (machineId: string) => handleMachineTest(controllerState, machineId),
    handleDelete: (machineId: string) => handleMachineDelete(controllerState, machineId),
    resetDraft: (machineId?: string) => resetMachineDraft(controllerState, machineId),
  }
  return view
}
