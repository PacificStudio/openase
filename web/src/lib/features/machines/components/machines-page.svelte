<script lang="ts">
  import MachinesPageBody from './machines-page-body.svelte'
  import { createMachinesPageController } from './machines-page-controller.svelte'
  import { updateMachineDraft } from '../model'

  const controller = createMachinesPageController()
</script>

<MachinesPageBody
  routeOrgId={controller.routeOrgId}
  loading={controller.loading}
  refreshing={controller.refreshing}
  workspaceState={controller.workspaceState}
  listMessage={controller.listMessage}
  machines={controller.filteredMachines}
  selectedId={controller.selectedId}
  searchQuery={controller.searchQuery}
  selectedMachine={controller.selectedMachine}
  mode={controller.mode}
  draft={controller.draft}
  snapshot={controller.snapshot}
  probe={controller.probe}
  loadingHealth={controller.loadingHealth}
  refreshingHealthMachineId={controller.refreshingHealthMachineId}
  saving={controller.saving}
  testingMachineId={controller.testingMachineId}
  deletingMachineId={controller.deletingMachineId}
  bind:editorOpen={controller.editorOpen}
  onRefresh={() => void controller.handleRefresh()}
  onCreate={controller.startCreate}
  onSearchChange={(value) => (controller.searchQuery = value)}
  onSelectMachine={(machineId) => {
    const nextMachine = controller.machines.find((machine) => machine.id === machineId)
    if (nextMachine) void controller.openMachine(nextMachine)
  }}
  onDraftChange={(field, value) =>
    (controller.draft = updateMachineDraft(
      controller.draft,
      field,
      value,
      controller.selectedMachine,
    ))}
  onRetry={() => void controller.handleRefresh()}
  onRefreshHealth={(machineId) => void controller.handleRefreshHealth(machineId)}
  onSave={() => void controller.handleSave()}
  onTest={(machineId) => void controller.handleTest(machineId)}
  onDelete={(machineId) => void controller.handleDelete(machineId)}
  onReset={controller.resetDraft}
/>
