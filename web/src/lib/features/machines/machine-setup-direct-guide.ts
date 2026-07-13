import { i18nStore } from '$lib/i18n/store.svelte'
import type { MachineSnapshot } from './types'
import type { MachineSetupCommand, MachineSetupGuide, MachineLike } from './machine-setup-types'

export function buildDirectConnectGuide(input: {
  machine: MachineLike | null
  advertisedEndpoint: string
  snapshot: MachineSnapshot | null
  sshPreview: string | null
}): MachineSetupGuide {
  const { machine, advertisedEndpoint, snapshot, sshPreview } = input
  const nextSteps: string[] = []
  const commands: MachineSetupCommand[] = []

  const runtimeLabel = i18nStore.t('machines.setup.directConnect.runtimeLabel')
  const runtimeSummary = i18nStore.t('machines.setup.directConnect.runtimeSummary')
  const helperLabel = sshPreview
    ? i18nStore.t('machines.setup.directConnect.helperLabelAvailable')
    : i18nStore.t('machines.setup.directConnect.helperLabelMissing')
  const helperSummary = sshPreview
    ? i18nStore.t('machines.setup.directConnect.helperSummaryAvailable')
    : i18nStore.t('machines.setup.directConnect.helperSummaryMissing')
  let stateLabel = i18nStore.t('machines.setup.directConnect.stateLabelWaiting')
  let stateSummary = i18nStore.t('machines.setup.directConnect.stateSummaryWaiting')

  if (!advertisedEndpoint) {
    nextSteps.push(i18nStore.t('machines.setup.directConnect.nextSteps.addEndpoint'))
  } else {
    const reachability = snapshot?.monitor.l1?.reachable
    stateLabel =
      reachability === true
        ? i18nStore.t('machines.setup.directConnect.stateLabelHealthy')
        : reachability === false
          ? i18nStore.t('machines.setup.directConnect.stateLabelFailing')
          : i18nStore.t('machines.setup.directConnect.stateLabelConfigured')
    stateSummary =
      reachability === true
        ? i18nStore.t('machines.setup.directConnect.stateSummaryHealthy')
        : reachability === false
          ? i18nStore.t('machines.setup.directConnect.stateSummaryFailing')
          : i18nStore.t('machines.setup.directConnect.stateSummaryConfigured')
    nextSteps.push(i18nStore.t('machines.setup.directConnect.nextSteps.runConnectionTest'))
  }

  if (sshPreview) {
    nextSteps.push(i18nStore.t('machines.setup.directConnect.nextSteps.useSshHelper'))
    commands.push({
      title: i18nStore.t('machines.setup.directConnect.commands.sshQuickSetup.title'),
      description: i18nStore.t('machines.setup.directConnect.commands.sshQuickSetup.description'),
      command: sshPreview,
    })
    if (machine?.id) {
      commands.push({
        title: i18nStore.t('machines.setup.directConnect.commands.sshBootstrap.title'),
        description: i18nStore.t('machines.setup.directConnect.commands.sshBootstrap.description'),
        command: `openase machine ssh-bootstrap ${machine.id}`,
      })
      commands.push({
        title: i18nStore.t('machines.setup.directConnect.commands.sshDiagnostics.title'),
        description: i18nStore.t(
          'machines.setup.directConnect.commands.sshDiagnostics.description',
        ),
        command: `openase machine ssh-diagnostics ${machine.id}`,
      })
    }
  } else {
    nextSteps.push(i18nStore.t('machines.setup.directConnect.nextSteps.addSshHelper'))
  }

  if (machine?.id) {
    commands.push({
      title: i18nStore.t('machines.setup.directConnect.commands.connectionTest.title'),
      description: i18nStore.t('machines.setup.directConnect.commands.connectionTest.description'),
      command: `openase machine test ${machine.id}`,
    })
  }

  return {
    topologyLabel: i18nStore.t('machines.setup.directConnect.topologyLabel'),
    topologySummary: i18nStore.t('machines.setup.directConnect.topologySummary'),
    runtimeLabel,
    runtimeSummary,
    helperLabel,
    helperSummary,
    stateLabel,
    stateSummary,
    nextSteps,
    commands,
  }
}
