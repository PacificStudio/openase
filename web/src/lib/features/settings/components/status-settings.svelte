<script lang="ts">
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { Button } from '$ui/button'
  import { Separator } from '$ui/separator'
  import { RotateCcw } from '@lucide/svelte'
  import StatusSettingsList from './status-settings-list.svelte'
  import { createStatusSettingsState } from './status-settings-state.svelte'

  const state = createStatusSettingsState()
</script>

<div class="space-y-6 pb-8">
  <div class="flex items-start justify-between gap-4">
    <div>
      <h2 class="text-foreground text-base font-semibold">
        {i18nStore.t('settings.statuses.title')}
      </h2>
      <p class="text-muted-foreground mt-1 text-sm">
        {i18nStore.t('settings.statuses.description')}
      </p>
    </div>
    <Button
      variant="outline"
      size="sm"
      disabled={state.ui.resetting || state.ui.loading}
      onclick={state.resetStatuses}
    >
      <RotateCcw class="size-3.5" />
      {state.ui.resetting
        ? i18nStore.t('settings.statuses.actions.resetting')
        : i18nStore.t('settings.statuses.actions.reset')}
    </Button>
  </div>

  <Separator />

  <StatusSettingsList
    statuses={state.ui.statuses}
    loading={state.ui.loading}
    resetting={state.ui.resetting}
    creating={state.ui.creating}
    busyStatusId={state.ui.busyStatusId}
    onSave={state.saveStatus}
    onDelete={state.deleteStatus}
    onMoveInStage={state.moveStatusInStage}
    onSetDefault={state.setDefaultStatus}
    onMoveToStage={state.moveToStage}
    onCreateInStage={state.createStatusInStage}
  />
</div>
