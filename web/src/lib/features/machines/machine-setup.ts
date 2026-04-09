import type { Machine } from '$lib/api/contracts'
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
      topologyLabel: 'Local control-plane host',
      topologySummary: 'OpenASE runs on the same host, so no remote bootstrap path is needed.',
      runtimeLabel: 'Local process runtime',
      runtimeSummary: 'Commands execute directly on the control-plane machine.',
      helperLabel: 'SSH helper unused',
      helperSummary: 'Local machines do not use SSH helper access.',
      stateLabel: snapshot?.checkedAt ? 'Local checks recorded' : 'Waiting for local checks',
      stateSummary: snapshot?.checkedAt
        ? 'The latest snapshot reflects the local host.'
        : 'Run checks to confirm the local runtime environment.',
      nextSteps: [
        'Confirm the workspace root and agent CLI path on this host.',
        'Run checks after local toolchain changes to refresh runtime health.',
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

  const runtimeLabel = 'Listener runtime'
  const runtimeSummary =
    'The control plane opens a websocket connection to the machine’s advertised listener endpoint.'
  const helperLabel = sshPreview ? 'SSH helper available' : 'No SSH helper saved'
  const helperSummary = sshPreview
    ? 'Use SSH only for quick bootstrap, diagnostics, or emergency repair.'
    : 'Add an SSH user and key path if you want a helper lane for bootstrap or diagnostics.'
  let stateLabel = 'Waiting for listener'
  let stateSummary =
    'Add the listener endpoint, then run a connection test so OpenASE can verify reachability.'

  if (!advertisedEndpoint) {
    nextSteps.push('Add the direct-connect listener endpoint before running connection checks.')
  } else {
    const reachability = snapshot?.monitor.l1?.reachable
    stateLabel =
      reachability === true
        ? 'Listener healthy'
        : reachability === false
          ? 'Listener checks failing'
          : 'Listener configured'
    stateSummary =
      reachability === true
        ? 'The listener endpoint is saved and recent reachability checks succeeded.'
        : reachability === false
          ? 'The listener endpoint is saved, but the latest reachability check failed.'
          : 'The listener endpoint is saved. Run a connection test to verify it from the control plane.'
    nextSteps.push('Run a connection test from OpenASE to verify the listener path end to end.')
  }

  if (sshPreview) {
    nextSteps.push(
      'Use the SSH helper lane for bootstrap or repair, then return here to rerun checks.',
    )
    commands.push({
      title: 'SSH quick setup',
      description:
        'Open a helper session on the machine to install or repair OpenASE before retesting.',
      command: sshPreview,
    })
  } else {
    nextSteps.push(
      'Optional: add SSH helper credentials if you want a direct bootstrap and diagnostics path.',
    )
  }

  if (machine?.id) {
    commands.push({
      title: 'Control-plane connection test',
      description: 'Verify that OpenASE can reach the advertised direct-connect path.',
      command: `openase machine test ${machine.id}`,
    })
  }

  return {
    topologyLabel: 'Control plane connects directly',
    topologySummary: 'Choose this when OpenASE can dial the machine over the network.',
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
      title: '1. Issue a machine channel token on the control plane',
      description: 'Copy the exported OPENASE_MACHINE_* values from this command.',
      command: `openase machine issue-channel-token --machine-id ${machine?.id ?? '<saved-machine-id>'} --format shell`,
    },
    {
      title: '2. Paste those exports on the remote machine and start the daemon',
      description:
        'Run this after pasting the exported environment variables onto the remote host.',
      command: 'openase machine-agent run',
    },
  ]

  if (machine?.id && sshPreview) {
    commands.push({
      title: 'Optional SSH bootstrap',
      description:
        'Upload the OpenASE binary and install the reverse-connect user service over SSH.',
      command: `openase machine ssh-bootstrap ${machine.id}`,
    })
    commands.push({
      title: 'SSH diagnostics',
      description:
        'Inspect bootstrap, service, workspace, and daemon registration health over the helper lane.',
      command: `openase machine ssh-diagnostics ${machine.id}`,
    })
  }

  if (!registered) {
    nextSteps.push(
      'Issue a machine channel token, paste the exported OPENASE_MACHINE_* variables on the remote host, and start `openase machine-agent run`.',
    )
  } else if (!sessionConnected) {
    nextSteps.push(
      `The machine is registered but the current daemon session is ${humanizeSessionState(sessionState).toLowerCase()}. Restart the daemon or rerun bootstrap to reconnect it.`,
    )
  } else {
    nextSteps.push('The daemon is connected. Run checks to confirm runtime and tooling readiness.')
  }

  if (sshPreview) {
    nextSteps.push(
      'Use the SSH bootstrap helper if you want OpenASE to install or refresh the daemon service for you.',
    )
  } else {
    nextSteps.push(
      'Add SSH helper credentials if you want an assisted bootstrap and diagnostics lane.',
    )
  }
  if (snapshot?.monitor.l4?.checkedAt || snapshot?.monitor.l5?.checkedAt) {
    nextSteps.push(
      'Review the health panel below for runtime and tooling readiness after the daemon connects.',
    )
  }

  return {
    topologyLabel: 'Machine dials out to OpenASE',
    topologySummary:
      'Choose this when the machine can reach the control plane but cannot accept direct inbound control-plane connections.',
    runtimeLabel: 'Reverse-connect daemon',
    runtimeSummary:
      'Runtime execution rides on the machine-agent websocket session instead of an advertised listener endpoint.',
    helperLabel: sshPreview ? 'SSH bootstrap available' : 'No SSH bootstrap helper saved',
    helperSummary: sshPreview
      ? 'SSH remains optional. Use it when you want OpenASE to upload the binary and install the daemon service for you.'
      : 'Self-serve daemon startup works without SSH. Add helper credentials only if you want assisted bootstrap and diagnostics.',
    stateLabel: sessionConnected
      ? 'Daemon connected'
      : registered
        ? `Daemon ${humanizeSessionState(sessionState)}`
        : 'Waiting for daemon registration',
    stateSummary: sessionConnected
      ? 'The control plane currently sees an active reverse-connect daemon session.'
      : registered
        ? `The machine is registered, but the current daemon session is ${humanizeSessionState(sessionState).toLowerCase()}.`
        : 'The control plane has not seen an active reverse-connect daemon registration for this machine yet.',
    nextSteps,
    commands,
  }
}
