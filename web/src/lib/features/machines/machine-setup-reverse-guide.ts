import { i18nStore } from '$lib/i18n/store.svelte'
import type { MachineSnapshot } from './types'
import type { MachineSetupCommand, MachineSetupGuide, MachineLike } from './machine-setup-types'
import { humanizeSessionState, normalizeSessionState } from './machine-setup-format'

export function buildReverseConnectGuide(input: {
  machine: MachineLike | null
  snapshot: MachineSnapshot | null
  sshPreview: string | null
}): MachineSetupGuide {
  const { machine, snapshot, sshPreview } = input
  const sessionState = normalizeSessionState(machine?.daemon_status?.session_state)
  const sessionConnected = sessionState === 'connected'
  const registered = Boolean(machine?.daemon_status?.registered)
  const nextSteps: string[] = []
  const commands: MachineSetupCommand[] = [
    {
      title: i18nStore.t('machines.setup.reverseConnect.commands.issueToken.title'),
      description: i18nStore.t('machines.setup.reverseConnect.commands.issueToken.description'),
      command: `openase machine issue-channel-token --machine-id ${machine?.id ?? '<saved-machine-id>'} --format shell`,
    },
    {
      title: i18nStore.t('machines.setup.reverseConnect.commands.startDaemon.title'),
      description: i18nStore.t('machines.setup.reverseConnect.commands.startDaemon.description'),
      command: 'openase machine-agent run',
    },
  ]

  if (machine?.id && sshPreview) {
    commands.push({
      title: i18nStore.t('machines.setup.reverseConnect.commands.sshBootstrap.title'),
      description: i18nStore.t('machines.setup.reverseConnect.commands.sshBootstrap.description'),
      command: `openase machine ssh-bootstrap ${machine.id}`,
    })
    commands.push({
      title: i18nStore.t('machines.setup.reverseConnect.commands.sshDiagnostics.title'),
      description: i18nStore.t('machines.setup.reverseConnect.commands.sshDiagnostics.description'),
      command: `openase machine ssh-diagnostics ${machine.id}`,
    })
  }

  if (!registered) {
    nextSteps.push(i18nStore.t('machines.setup.reverseConnect.nextSteps.registerDaemon'))
  } else if (!sessionConnected) {
    nextSteps.push(
      i18nStore.t('machines.setup.reverseConnect.nextSteps.restartDaemon', {
        state: humanizeSessionState(sessionState).toLowerCase(),
      }),
    )
  } else {
    nextSteps.push(i18nStore.t('machines.setup.reverseConnect.nextSteps.runChecks'))
  }

  if (sshPreview) {
    nextSteps.push(i18nStore.t('machines.setup.reverseConnect.nextSteps.useSshBootstrap'))
  } else {
    nextSteps.push(i18nStore.t('machines.setup.reverseConnect.nextSteps.addSshHelper'))
  }
  if (snapshot?.monitor.l4?.checkedAt || snapshot?.monitor.l5?.checkedAt) {
    nextSteps.push(i18nStore.t('machines.setup.reverseConnect.nextSteps.reviewHealthPanel'))
  }

  return {
    topologyLabel: i18nStore.t('machines.setup.reverseConnect.topologyLabel'),
    topologySummary: i18nStore.t('machines.setup.reverseConnect.topologySummary'),
    runtimeLabel: i18nStore.t('machines.setup.reverseConnect.runtimeLabel'),
    runtimeSummary: i18nStore.t('machines.setup.reverseConnect.runtimeSummary'),
    helperLabel: sshPreview
      ? i18nStore.t('machines.setup.reverseConnect.helperLabelAvailable')
      : i18nStore.t('machines.setup.reverseConnect.helperLabelMissing'),
    helperSummary: sshPreview
      ? i18nStore.t('machines.setup.reverseConnect.helperSummaryAvailable')
      : i18nStore.t('machines.setup.reverseConnect.helperSummaryMissing'),
    stateLabel: sessionConnected
      ? i18nStore.t('machines.setup.reverseConnect.stateLabelConnected')
      : registered
        ? i18nStore.t('machines.setup.reverseConnect.stateLabelRegistered', {
            state: humanizeSessionState(sessionState),
          })
        : i18nStore.t('machines.setup.reverseConnect.stateLabelWaiting'),
    stateSummary: sessionConnected
      ? i18nStore.t('machines.setup.reverseConnect.stateSummaryConnected')
      : registered
        ? i18nStore.t('machines.setup.reverseConnect.stateSummaryRegistered', {
            state: humanizeSessionState(sessionState).toLowerCase(),
          })
        : i18nStore.t('machines.setup.reverseConnect.stateSummaryWaiting'),
    nextSteps,
    commands,
  }
}
