import type { MachineCLIStatus } from '../types'

export type HealthStatCard = {
  label: string
  value: string
  meta: string
}

export type HealthLevelCard = {
  id: string
  label: string
  state: string
  value: string
  meta: string
}

export type TruthyState = 'yes' | 'no' | 'unknown'

export type AuditNetworkEndpoint = {
  name: string
  reachable: TruthyState
}

export type HealthAuditRow =
  | {
      kind: 'git'
      label: string
      installed: TruthyState
      identity: string | null
    }
  | {
      kind: 'gh-cli'
      label: string
      installed: TruthyState
      authStatus: string | null
    }
  | {
      kind: 'network'
      label: string
      endpoints: AuditNetworkEndpoint[]
      auditTimestamp: string | null
    }

export type MachineLevelStateInput = {
  error?: string
  checkedAt?: string
  state?: string
}

export type RuntimeStatusLike = Pick<MachineCLIStatus, 'name'>
