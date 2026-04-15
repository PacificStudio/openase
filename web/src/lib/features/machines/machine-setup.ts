import type { Machine } from '$lib/api/contracts'
import { i18nStore } from '$lib/i18n/store.svelte'
import type { MachineDraft, MachineSnapshot } from './types'
import { normalizeReachabilityMode } from './machine-guidance'
import {
  buildSSHHelperPreview,
  humanizeSessionState,
  normalizeSessionState,
} from './machine-setup-format'

export { buildSSHHelperPreview, friendlyTransportLabel } from './machine-setup-format'

export type MachineSetupCommand = {
  title: string
  description: string
  command: string
}

export type MachineSetupGuide = {
  topologyLabel: string
  topologySummary: string
  runtimeLabel: string
  runtimeSummary: string
  helperLabel: string
  helperSummary: string
  stateLabel: string
  stateSummary: string
  nextSteps: string[]
  commands: MachineSetupCommand[]
}

type MachineLike = Pick<
  Machine,
  | 'id'
  | 'host'
  | 'ssh_user'
  | 'ssh_key_path'
  | 'advertised_endpoint'
  | 'reachability_mode'
  | 'execution_mode'
  | 'ssh_helper_enabled'
  | 'daemon_status'
>

type DraftLike = Pick<
  MachineDraft,
  'host' | 'sshUser' | 'sshKeyPath' | 'advertisedEndpoint' | 'reachabilityMode' | 'executionMode'
>

export function buildMachineSetupGuide(input: {
  machine: MachineLike | null
  draft?: DraftLike | null
  snapshot?: MachineSnapshot | null
}): MachineSetupGuide {
  const machine = input.machine
  const draft = input.draft
  const snapshot = input.snapshot ?? null

  const host = draft?.host ?? machine?.host ?? ''
  const reachabilityMode = normalizeReachabilityMode(
    draft?.reachabilityMode ?? machine?.reachability_mode,
    host,
  )
  const sshUser = (draft?.sshUser ?? machine?.ssh_user ?? '').trim()
  const sshKeyPath = (draft?.sshKeyPath ?? machine?.ssh_key_path ?? '').trim()
  const advertisedEndpoint = (
    draft?.advertisedEndpoint ??
    machine?.advertised_endpoint ??
    ''
  ).trim()
  const sshPreview = buildSSHHelperPreview({
    host,
    sshUser,
    sshKeyPath,
  })

  if (reachabilityMode === 'local') {
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

  if (reachabilityMode === 'reverse_connect') {
    return buildReverseConnectGuide(machine, snapshot, sshPreview)
  }

  return buildDirectConnectGuide({
    machine,
    advertisedEndpoint,
    snapshot,
    sshPreview,
  })
}

function buildDirectConnectGuide(input: {
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
        description: i18nStore.t('machines.setup.directConnect.commands.sshDiagnostics.description'),
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

function buildReverseConnectGuide(
  machine: MachineLike | null,
  snapshot: MachineSnapshot | null,
  sshPreview: string | null,
): MachineSetupGuide {
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
