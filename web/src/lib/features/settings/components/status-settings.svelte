<script lang="ts">
  import { Separator } from '$ui/separator'
  import StatusSettingsCreate from './status-settings-create.svelte'
  import StatusSettingsList from './status-settings-list.svelte'
  import { createStatusSettingsState } from './status-settings-state.svelte'

  const state = createStatusSettingsState()
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Statuses</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Configure the board order, default status, and optional status-level concurrency limits.
    </p>
  </div>

  <Separator />

  <div class="space-y-6">
    <StatusSettingsCreate
      bind:name={state.ui.createName}
      bind:color={state.ui.createColor}
      bind:isDefault={state.ui.createDefault}
      bind:maxActiveRuns={state.ui.createMaxActiveRuns}
      creating={state.ui.creating}
      loading={state.ui.loading}
      resetting={state.ui.resetting}
      onCreate={state.createStatus}
      onReset={state.resetStatuses}
    />

    <StatusSettingsList
      statuses={state.ui.statuses}
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
