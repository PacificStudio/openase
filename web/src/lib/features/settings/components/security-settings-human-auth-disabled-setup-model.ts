import type { OIDCFormState } from '$lib/features/auth'
import type { SecurityAuthSettings } from '$lib/api/contracts'

export type OIDCTestResult = {
  status: string
  message: string
  issuer_url: string
  authorization_endpoint: string
  token_endpoint: string
  redirect_url: string
  warnings: string[]
}

export type OIDCEnableResult = {
  status: string
  message: string
  restart_required: boolean
  next_steps: string[]
}

export type SecuritySettingsHumanAuthDisabledSetupProps = {
  auth: SecurityAuthSettings
  form: OIDCFormState
  actionKey?: string
  error?: string
  testResult?: OIDCTestResult | null
  enableResult?: OIDCEnableResult | null
  onIssuerURL: (value: string) => void
  onClientID: (value: string) => void
  onClientSecret: (value: string) => void
  onRedirectMode: (value: 'auto' | 'fixed') => void
  onFixedRedirectURL: (value: string) => void
  onScopes: (value: string) => void
  onAllowedDomains: (value: string) => void
  onBootstrapAdmins: (value: string) => void
  onSessionTTL: (value: string) => void
  onSessionIdleTTL: (value: string) => void
  onSave: () => void
  onTest: () => void
  onEnable: () => void
}
