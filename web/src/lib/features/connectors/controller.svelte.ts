import {
  createWorkspaceController,
  toErrorMessage,
  type Organization,
  type Project,
  type WorkspaceStartOptions,
} from '$lib/features/workspace'
import {
  createConnector,
  deleteConnector,
  isConnectorAPIUnavailable,
  listProjectConnectors,
  syncConnector,
  testConnector,
  updateConnector,
} from './api'
import {
  defaultConnectorForm,
  toConnectorForm,
  toConnectorInput,
  type ConnectorForm,
  type IssueConnector,
} from './types'
import { deriveSelectionState, deriveUpdatedForm, findConnector } from './state'
import {
  deleteLocalConnector,
  readLocalConnectors,
  setLocalConnectorStatus,
  simulateLocalSync,
  simulateLocalTest,
  upsertLocalConnector,
} from './storage'
type ConnectorAction = '' | 'save' | 'sync' | 'test' | 'toggle-status' | 'delete'

export function createConnectorsController() {
  const workspace = createWorkspaceController()
  let connectors = $state<IssueConnector[]>([])
  let selectedConnectorId = $state('')
  let form = $state<ConnectorForm>(defaultConnectorForm())
  let busy = $state(false)
  let error = $state('')
  let notice = $state('')
  let pendingConnectorId = $state('')
  let pendingAction = $state<ConnectorAction>('')
  let persistenceMode = $state<'api' | 'local'>('api')
  async function start(options: WorkspaceStartOptions = {}) {
    await workspace.start(options)
    await refreshConnectors()
  }
  function destroy() {
    workspace.destroy()
  }
  async function refreshConnectors(preferredConnectorId = selectedConnectorId) {
    const projectId = workspace.state.selectedProjectId
    if (!projectId) {
      connectors = []
      selectedConnectorId = ''
      form = defaultConnectorForm()
      error = ''
      notice = ''
      return
    }

    busy = true
    error = ''
    notice = ''
    try {
      connectors = await listProjectConnectors(projectId)
      persistenceMode = 'api'
    } catch (loadError) {
      if (isConnectorAPIUnavailable(loadError)) {
        connectors = readLocalConnectors(projectId)
        persistenceMode = 'local'
        notice =
          'Connector API is not available in this binary yet. Using browser-local draft mode for the current project.'
      } else {
        connectors = []
        error = toErrorMessage(loadError)
      }
    } finally {
      busy = false
      syncSelection(preferredConnectorId)
    }
  }
  async function selectOrganization(organization: Organization) {
    await workspace.selectOrganization(organization)
    await refreshConnectors('')
  }
  async function selectProject(project: Project) {
    await workspace.selectProject(project)
    await refreshConnectors('')
  }
  function syncSelection(preferredConnectorId = selectedConnectorId) {
    const nextState = deriveSelectionState(connectors, preferredConnectorId)
    selectedConnectorId = nextState.selectedConnectorId
    form = nextState.form
  }
  function selectedConnector() {
    return findConnector(connectors, selectedConnectorId)
  }
  function startCreate() {
    selectedConnectorId = ''
    form = defaultConnectorForm()
    error = ''
  }
  function selectConnector(connectorId: string) {
    const connector = findConnector(connectors, connectorId)
    if (!connector) {
      return
    }
    selectedConnectorId = connector.id
    form = toConnectorForm(connector)
    error = ''
  }
  function updateForm<K extends keyof ConnectorForm>(key: K, value: ConnectorForm[K]) {
    form = deriveUpdatedForm(form, key, value)
  }
  async function saveCurrent() {
    const projectId = workspace.state.selectedProjectId
    if (!projectId) {
      error = 'Select a project before editing connectors.'
      return
    }
    pendingAction = 'save'
    pendingConnectorId = selectedConnectorId
    error = ''
    try {
      const input = toConnectorInput(form)
      if (persistenceMode === 'api') {
        const connector = selectedConnectorId
          ? await updateConnector(selectedConnectorId, input)
          : await createConnector(projectId, input)
        notice = selectedConnectorId ? 'Connector updated.' : 'Connector created.'
        await refreshConnectors(connector.id)
        return
      }

      connectors = upsertLocalConnector(projectId, selectedConnectorId, input)
      notice = selectedConnectorId
        ? 'Connector draft updated locally.'
        : 'Connector draft created locally.'
      syncSelection(selectedConnectorId || connectors[0]?.id || '')
    } catch (saveError) {
      if (isConnectorAPIUnavailable(saveError)) {
        persistenceMode = 'local'
        connectors = upsertLocalConnector(projectId, selectedConnectorId, toConnectorInput(form))
        notice = 'Connector API fell back to browser-local draft mode during save.'
        syncSelection(selectedConnectorId || connectors[0]?.id || '')
      } else {
        error = toErrorMessage(saveError)
      }
    } finally {
      pendingAction = ''
      pendingConnectorId = ''
    }
  }
  async function runSync(connectorId: string) {
    await runConnectorAction(connectorId, 'sync', async (projectId) => {
      if (persistenceMode === 'api') {
        await syncConnector(connectorId)
        notice = 'Sync requested.'
        await refreshConnectors(connectorId)
        return
      }

      connectors = simulateLocalSync(projectId, connectorId)
      notice = 'Local draft sync simulated.'
      syncSelection(connectorId)
    })
  }
  async function runTest(connectorId: string) {
    await runConnectorAction(connectorId, 'test', async (projectId) => {
      if (persistenceMode === 'api') {
        await testConnector(connectorId)
        notice = 'Connection test requested.'
        await refreshConnectors(connectorId)
        return
      }

      connectors = simulateLocalTest(projectId, connectorId)
      notice = 'Local draft health check simulated.'
      syncSelection(connectorId)
    })
  }
  async function toggleStatus(connectorId: string) {
    const connector = findConnector(connectors, connectorId)
    if (!connector) {
      return
    }

    const nextStatus = connector.status === 'paused' ? 'active' : 'paused'
    await runConnectorAction(connectorId, 'toggle-status', async (projectId) => {
      if (persistenceMode === 'api') {
        await updateConnector(connectorId, {
          ...toConnectorInput(toConnectorForm(connector)),
          status: nextStatus,
        })
        notice = nextStatus === 'paused' ? 'Connector paused.' : 'Connector resumed.'
        await refreshConnectors(connectorId)
        return
      }

      connectors = setLocalConnectorStatus(projectId, connectorId, nextStatus)
      notice =
        nextStatus === 'paused'
          ? 'Connector paused in local draft mode.'
          : 'Connector resumed in local draft mode.'
      syncSelection(connectorId)
    })
  }
  async function removeCurrent() {
    const projectId = workspace.state.selectedProjectId
    if (!projectId || !selectedConnectorId) {
      return
    }

    await runConnectorAction(selectedConnectorId, 'delete', async () => {
      if (persistenceMode === 'api') {
        await deleteConnector(selectedConnectorId)
        notice = 'Connector removed.'
        await refreshConnectors('')
        return
      }

      connectors = deleteLocalConnector(projectId, selectedConnectorId)
      selectedConnectorId = ''
      form = defaultConnectorForm()
      notice = 'Connector draft removed from local storage.'
      syncSelection('')
    })
  }
  async function runConnectorAction(
    connectorId: string,
    action: ConnectorAction,
    callback: (projectId: string) => Promise<void>,
  ) {
    const projectId = workspace.state.selectedProjectId
    if (!projectId) {
      error = 'Select a project before running connector actions.'
      return
    }

    pendingConnectorId = connectorId
    pendingAction = action
    error = ''
    try {
      await callback(projectId)
    } catch (actionError) {
      error = toErrorMessage(actionError)
    } finally {
      pendingConnectorId = ''
      pendingAction = ''
    }
  }
  return {
    workspace,
    get connectors() {
      return connectors
    },
    get selectedConnectorId() {
      return selectedConnectorId
    },
    get form() {
      return form
    },
    get busy() {
      return busy
    },
    get error() {
      return error
    },
    get notice() {
      return notice
    },
    get pendingConnectorId() {
      return pendingConnectorId
    },
    get pendingAction() {
      return pendingAction
    },
    get persistenceMode() {
      return persistenceMode
    },
    start,
    destroy,
    refreshConnectors,
    selectOrganization,
    selectProject,
    selectedConnector,
    selectConnector,
    startCreate,
    updateForm,
    saveCurrent,
    runSync,
    runTest,
    toggleStatus,
    removeCurrent,
  }
}
