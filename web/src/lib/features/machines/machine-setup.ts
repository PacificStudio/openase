import { normalizeReachabilityMode } from './machine-guidance'
import {
  buildDirectConnectGuide,
  buildLocalGuide,
  buildReverseConnectGuide,
  applyMaintenanceOverride,
} from './machine-setup-guide-builders'
import {
  buildSSHHelperPreview,
  humanizeSessionState,
  normalizeSessionState,
} from './machine-setup-format'
import type { BuildMachineSetupGuideInput, MachineSetupGuide } from './machine-setup-types'

export { buildSSHHelperPreview, friendlyTransportLabel } from './machine-setup-format'
export type {
  BuildMachineSetupGuideInput,
  DraftLike,
  MachineLike,
  MachineSetupCommand,
  MachineSetupGuide,
} from './machine-setup-types'

export function buildMachineSetupGuide(input: BuildMachineSetupGuideInput): MachineSetupGuide {
  const { machine, draft } = input
  const snapshot = input.snapshot ?? null
  const host = draft?.host ?? machine?.host ?? ''
  const reachabilityMode = normalizeReachabilityMode(
    draft?.reachabilityMode ?? machine?.reachability_mode,
    host,
  )
  const sshPreview = buildSSHHelperPreview({
    host,
    sshUser: (draft?.sshUser ?? machine?.ssh_user ?? '').trim(),
    sshKeyPath: (draft?.sshKeyPath ?? machine?.ssh_key_path ?? '').trim(),
  })

  const baseGuide =
    reachabilityMode === 'local'
      ? buildLocalGuide(snapshot)
      : reachabilityMode === 'reverse_connect'
        ? buildReverseConnectGuide({ machine, snapshot, sshPreview })
        : buildDirectConnectGuide({
            machine,
            advertisedEndpoint: (
              draft?.advertisedEndpoint ??
              machine?.advertised_endpoint ??
              ''
            ).trim(),
            snapshot,
            sshPreview,
          })

  return applyMaintenanceOverride(draft?.status ?? machine?.status, baseGuide)
}

export { humanizeSessionState, normalizeSessionState }
