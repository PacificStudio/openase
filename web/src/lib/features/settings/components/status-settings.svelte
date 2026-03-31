<script lang="ts">
  import { Separator } from '$ui/separator'
  import StageSettingsPanel from './stage-settings-panel.svelte'
  import StatusSettingsCreate from './status-settings-create.svelte'
  import StatusSettingsList from './status-settings-list.svelte'
  import { createStatusSettingsState } from './status-settings-state.svelte'

  const state = createStatusSettingsState()
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Statuses</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Configure the stage groups that workflows share, then assign statuses into those stages.
    </p>
  </div>

  <Separator />

  <div class="grid gap-6 xl:grid-cols-[minmax(0,1.05fr)_minmax(0,1fr)]">
    <StageSettingsPanel
      stages={state.ui.stages}
      statuses={state.ui.statuses}
      loading={state.ui.loading}
      creating={state.ui.creatingStage}
      busyStageId={state.ui.busyStageId}
      onCreate={state.createStage}
      onSave={state.saveStage}
      onDelete={state.deleteStage}
      onMove={state.moveStage}
    />

    <div class="space-y-6">
      <StatusSettingsCreate
        bind:name={state.ui.createName}
        bind:color={state.ui.createColor}
        bind:isDefault={state.ui.createDefault}
        bind:stageId={state.ui.createStageId}
        stages={state.ui.stages}
        creating={state.ui.creating}
        loading={state.ui.loading}
        resetting={state.ui.resetting}
        onCreate={state.createStatus}
        onReset={state.resetStatuses}
      />

      <StatusSettingsList
        statuses={state.ui.statuses}
        stages={state.ui.stages}
        loading={state.ui.loading}
        resetting={state.ui.resetting}
        busyStatusId={state.ui.busyStatusId}
        onSave={state.saveStatus}
        onDelete={state.deleteStatus}
        onMove={state.moveStatus}
        onSetDefault={state.setDefaultStatus}
      />
    </div>
  </div>
</div>
