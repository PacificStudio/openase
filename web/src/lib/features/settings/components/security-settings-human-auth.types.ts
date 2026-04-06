import type { SecuritySettingsResponse } from '$lib/api/contracts'

export type ApprovalPoliciesSummary = {
  status: string
  rules_count: number
  summary: string
}

export type SecuritySettingsSecurity = SecuritySettingsResponse['security']
