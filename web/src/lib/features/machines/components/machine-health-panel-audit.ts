import { i18nStore } from '$lib/i18n/store.svelte'
import type { MachineSnapshot } from '../types'
import type { HealthAuditRow } from './machine-health-panel-types'
import { toTruthyState } from './machine-health-panel-status'

export function buildAuditRows(snapshot: MachineSnapshot): HealthAuditRow[] {
  if (!snapshot.fullAudit) {
    return []
  }

  const gitIdentity =
    [snapshot.fullAudit.git?.userName, snapshot.fullAudit.git?.userEmail]
      .filter(Boolean)
      .join(' · ') || null

  const network = snapshot.fullAudit.network

  return [
    {
      kind: 'git',
      label: i18nStore.t('machines.shared.git'),
      installed: toTruthyState(snapshot.fullAudit.git?.installed),
      identity: gitIdentity,
    },
    {
      kind: 'gh-cli',
      label: i18nStore.t('machines.shared.githubCli'),
      installed: toTruthyState(snapshot.fullAudit.ghCLI?.installed),
      authStatus: snapshot.fullAudit.ghCLI?.authStatus ?? null,
    },
    {
      kind: 'network',
      label: i18nStore.t('machines.shared.network'),
      endpoints: [
        { name: 'GitHub', reachable: toTruthyState(network?.githubReachable) },
        { name: 'PyPI', reachable: toTruthyState(network?.pypiReachable) },
        { name: 'npm', reachable: toTruthyState(network?.npmReachable) },
      ],
      auditTimestamp: snapshot.fullAudit.checkedAt ?? null,
    },
  ]
}
