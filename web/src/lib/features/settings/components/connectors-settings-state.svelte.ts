import { ApiError } from '$lib/api/client'
import {
  createIssueConnector,
  deleteIssueConnector,
  getIssueConnectorStats,
  listIssueConnectors,
  syncIssueConnector,
  testIssueConnector,
  updateIssueConnector,
  type IssueConnectorRecord,
} from '$lib/api/openase'
import { appStore } from '$lib/stores/app.svelte'
import { toastStore } from '$lib/stores/toast.svelte'
import {
  connectorStatsFallback,
  connectorToDraft,
  createConnectorsSettingsUI,
  createEmptyConnectorDraft,
  type ConnectorsSettingsUI,
  type ConnectorStats,
  parseConnectorCSV,
  parseConnectorStatusMapping,
  type ConnectorDraft,
} from '../connectors-model'

export function createConnectorsSettingsState() {
  const ui = $state<ConnectorsSettingsUI>(createConnectorsSettingsUI())

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      Object.assign(ui, createConnectorsSettingsUI())
      return
    }

    let cancelled = false
    const load = async () => {
      ui.loading = true
      try {
        const payload = await listIssueConnectors(projectId)
        if (cancelled) {
          return
        }
        assignConnectors(payload.connectors)
      } catch (error) {
        if (!cancelled) {
          toastStore.error(error instanceof ApiError ? error.detail : 'Failed to load connectors.')
        }
      } finally {
        if (!cancelled) {
          ui.loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  function assignConnectors(items: IssueConnectorRecord[]) {
    ui.connectors = items
    ui.statsByConnectorId = Object.fromEntries(
      items.map((connector) => [connector.id, connectorStatsFallback(connector)]),
    )
  }

  async function reload(projectId: string) {
    const payload = await listIssueConnectors(projectId)
    assignConnectors(payload.connectors)
  }

  return {
    ui,
    connectorStats(connector: IssueConnectorRecord): ConnectorStats {
      return ui.statsByConnectorId[connector.id] ?? connectorStatsFallback(connector)
    },
    resetEditor() {
      ui.editorMode = 'create'
      ui.editingConnectorId = ''
      ui.draft = createEmptyConnectorDraft()
    },
    async refreshStats(connectorId: string) {
      try {
        const payload = await getIssueConnectorStats(connectorId)
        ui.statsByConnectorId = { ...ui.statsByConnectorId, [connectorId]: payload.stats }
      } catch (error) {
        toastStore.error(
          error instanceof ApiError ? error.detail : 'Failed to refresh connector stats.',
        )
      }
    },
    startEdit(connector: IssueConnectorRecord) {
      ui.editorMode = 'edit'
      ui.editingConnectorId = connector.id
      ui.draft = connectorToDraft(connector)
      void this.refreshStats(connector.id)
    },
    updateDraft(field: keyof ConnectorDraft, value: string) {
      ui.draft = { ...ui.draft, [field]: value }
    },
    async save() {
      const projectId = appStore.currentProject?.id
      if (!projectId) {
        return
      }

      const name = ui.draft.name.trim()
      const projectRef = ui.draft.projectRef.trim()
      if (!name) {
        toastStore.error('Connector name is required.')
        return
      }
      if (!projectRef) {
        toastStore.error('Project ref is required.')
        return
      }

      const parsedStatusMapping = parseConnectorStatusMapping(ui.draft.statusMapping)
      if (!parsedStatusMapping.ok) {
        toastStore.error(parsedStatusMapping.error)
        return
      }

      ui.saving = true
      try {
        if (ui.editorMode === 'create') {
          const payload = await createIssueConnector(projectId, {
            type: 'github',
            name,
            status: ui.draft.status,
            config: {
              type: 'github',
              base_url: ui.draft.baseURL.trim(),
              auth_token: ui.draft.authToken.trim(),
              project_ref: projectRef,
              poll_interval: ui.draft.pollInterval.trim(),
              sync_direction: ui.draft.syncDirection,
              filters: { labels: parseConnectorCSV(ui.draft.labelFilter) },
              status_mapping: parsedStatusMapping.value,
              webhook_secret: ui.draft.webhookSecret.trim(),
              auto_workflow: ui.draft.autoWorkflow.trim(),
            },
          })
          await reload(projectId)
          this.resetEditor()
          toastStore.success(`Created connector "${payload.connector.name}".`)
        } else {
          const config: Parameters<typeof updateIssueConnector>[1]['config'] = {
            base_url: ui.draft.baseURL.trim(),
            project_ref: projectRef,
            poll_interval: ui.draft.pollInterval.trim(),
            sync_direction: ui.draft.syncDirection,
            filters: { labels: parseConnectorCSV(ui.draft.labelFilter) },
            status_mapping: parsedStatusMapping.value,
            auto_workflow: ui.draft.autoWorkflow.trim(),
          }
          if (ui.draft.authToken.trim() !== '') {
            config.auth_token = ui.draft.authToken.trim()
          }
          if (ui.draft.webhookSecret.trim() !== '') {
            config.webhook_secret = ui.draft.webhookSecret.trim()
          }

          const payload = await updateIssueConnector(ui.editingConnectorId, {
            name,
            status: ui.draft.status,
            config,
          })
          await reload(projectId)
          ui.draft = connectorToDraft(payload.connector)
          toastStore.success(`Updated connector "${payload.connector.name}".`)
        }
      } catch (error) {
        toastStore.error(error instanceof ApiError ? error.detail : 'Failed to save connector.')
      } finally {
        ui.saving = false
      }
    },
    async testConnector(connector: IssueConnectorRecord) {
      ui.busyConnectorId = connector.id
      try {
        const payload = await testIssueConnector(connector.id)
        if (payload.result.healthy) {
          toastStore.success(`Connector "${connector.name}" is healthy.`)
        } else {
          toastStore.error(payload.result.message)
        }
      } catch (error) {
        toastStore.error(error instanceof ApiError ? error.detail : 'Failed to test connector.')
      } finally {
        ui.busyConnectorId = ''
      }
    },
    async syncConnector(connector: IssueConnectorRecord) {
      const projectId = appStore.currentProject?.id
      if (!projectId) {
        return
      }

      ui.busyConnectorId = connector.id
      try {
        const payload = await syncIssueConnector(connector.id)
        await reload(projectId)
        ui.statsByConnectorId = {
          ...ui.statsByConnectorId,
          [connector.id]: this.connectorStats(payload.connector),
        }
        toastStore.success(
          `Synced ${payload.report.issues_synced} issues from "${connector.name}".`,
        )
      } catch (error) {
        toastStore.error(error instanceof ApiError ? error.detail : 'Failed to sync connector.')
      } finally {
        ui.busyConnectorId = ''
      }
    },
    async toggleConnectorStatus(connector: IssueConnectorRecord) {
      const projectId = appStore.currentProject?.id
      if (!projectId) {
        return
      }

      ui.busyConnectorId = connector.id
      const nextStatus = connector.status === 'paused' ? 'active' : 'paused'
      try {
        await updateIssueConnector(connector.id, { status: nextStatus })
        await reload(projectId)
        toastStore.success(
          nextStatus === 'active'
            ? `Resumed connector "${connector.name}".`
            : `Paused connector "${connector.name}".`,
        )
      } catch (error) {
        toastStore.error(
          error instanceof ApiError ? error.detail : 'Failed to update connector status.',
        )
      } finally {
        ui.busyConnectorId = ''
      }
    },
    async deleteConnector(connector: IssueConnectorRecord) {
      const projectId = appStore.currentProject?.id
      if (!projectId) {
        return
      }
      if (
        typeof window !== 'undefined' &&
        !window.confirm(`Delete connector "${connector.name}"?`)
      ) {
        return
      }

      ui.busyConnectorId = connector.id
      try {
        await deleteIssueConnector(connector.id)
        await reload(projectId)
        if (ui.editingConnectorId === connector.id) {
          this.resetEditor()
        }
        toastStore.success(`Deleted connector "${connector.name}".`)
      } catch (error) {
        toastStore.error(error instanceof ApiError ? error.detail : 'Failed to delete connector.')
      } finally {
        ui.busyConnectorId = ''
      }
    },
  }
}
