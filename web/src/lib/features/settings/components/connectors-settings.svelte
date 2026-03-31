<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { Separator } from '$ui/separator'
  import ConnectorEditorPanel from './connector-editor-panel.svelte'
  import ConnectorsList from './connectors-list.svelte'
  import { createConnectorsSettingsState } from './connectors-settings-state.svelte'

  const currentProjectName = $derived(appStore.currentProject?.name ?? 'this project')
  const state = createConnectorsSettingsState()
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Connectors</h2>
    <p class="text-muted-foreground mt-1 max-w-3xl text-sm">
      Manage external connectors for {currentProjectName}.
    </p>
  </div>

  <Separator />

  <div class="grid gap-6 xl:grid-cols-[minmax(0,1.15fr),minmax(0,0.85fr)]">
    <ConnectorsList
      projectName={currentProjectName}
      loading={state.ui.loading}
      connectors={state.ui.connectors}
      busyConnectorId={state.ui.busyConnectorId}
      connectorStats={(connector) => state.connectorStats(connector)}
      onCreate={() => state.resetEditor()}
      onEdit={(connector) => state.startEdit(connector)}
      onRefreshStats={(connectorId) => void state.refreshStats(connectorId)}
      onTest={(connector) => void state.testConnector(connector)}
      onSync={(connector) => void state.syncConnector(connector)}
      onToggleStatus={(connector) => void state.toggleConnectorStatus(connector)}
      onDelete={(connector) => void state.deleteConnector(connector)}
    />

    <ConnectorEditorPanel
      editorMode={state.ui.editorMode}
      draft={state.ui.draft}
      saving={state.ui.saving}
      onDraftChange={(field, value) => state.updateDraft(field, value)}
      onSave={() => void state.save()}
      onReset={() => state.resetEditor()}
    />
  </div>
</div>
