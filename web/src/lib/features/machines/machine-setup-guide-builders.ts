import { i18nStore } from '$lib/i18n/store.svelte'
import type { MachineSnapshot } from './types'
import type { MachineSetupGuide } from './machine-setup-types'

export { buildDirectConnectGuide } from './machine-setup-direct-guide'
export { buildReverseConnectGuide } from './machine-setup-reverse-guide'

export function applyMaintenanceOverride(
  status: string | undefined,
  guide: MachineSetupGuide,
): MachineSetupGuide {
  if (status !== 'maintenance') {
    return guide
  }

  return {
    ...guide,
    stateLabel: i18nStore.t('machines.setup.maintenance.stateLabel'),
    stateSummary: i18nStore.t('machines.setup.maintenance.stateSummary'),
    nextSteps: [
      i18nStore.t('machines.setup.maintenance.nextSteps.exitMaintenance'),
      ...guide.nextSteps,
    ],
  }
}

export function buildLocalGuide(snapshot: MachineSnapshot | null): MachineSetupGuide {
  return {
    topologyLabel: i18nStore.t('machines.setup.local.topologyLabel'),
    topologySummary: i18nStore.t('machines.setup.local.topologySummary'),
    runtimeLabel: i18nStore.t('machines.setup.local.runtimeLabel'),
    runtimeSummary: i18nStore.t('machines.setup.local.runtimeSummary'),
    helperLabel: i18nStore.t('machines.setup.local.helperLabel'),
    helperSummary: i18nStore.t('machines.setup.local.helperSummary'),
    stateLabel: snapshot?.checkedAt
      ? i18nStore.t('machines.setup.local.stateLabelRecorded')
      : i18nStore.t('machines.setup.local.stateLabelWaiting'),
    stateSummary: snapshot?.checkedAt
      ? i18nStore.t('machines.setup.local.stateSummaryRecorded')
      : i18nStore.t('machines.setup.local.stateSummaryWaiting'),
    nextSteps: [
      i18nStore.t('machines.setup.local.nextSteps.confirmWorkspace'),
      i18nStore.t('machines.setup.local.nextSteps.refreshChecks'),
    ],
    commands: [],
  }
}
