import { ApiError } from '$lib/api/client'
import type { TranslationKey } from '$lib/i18n'

export type ScopeKind = 'instance' | 'organization' | 'project'
export type SubjectKind = 'user' | 'group'

export type RoleOption = {
  key: string
  labelKey: TranslationKey
  summaryKey: TranslationKey
}

export type BindingDraft = {
  subjectKind: SubjectKind
  subjectKey: string
  roleKey: string
  expiresAtLocal: string
}

export const roleOptions: RoleOption[] = [
  {
    key: 'instance_admin',
    labelKey: 'settings.security.humanAuth.roles.instanceAdmin.label',
    summaryKey: 'settings.security.humanAuth.roles.instanceAdmin.summary',
  },
  {
    key: 'org_owner',
    labelKey: 'settings.security.humanAuth.roles.orgOwner.label',
    summaryKey: 'settings.security.humanAuth.roles.orgOwner.summary',
  },
  {
    key: 'org_admin',
    labelKey: 'settings.security.humanAuth.roles.orgAdmin.label',
    summaryKey: 'settings.security.humanAuth.roles.orgAdmin.summary',
  },
  {
    key: 'org_member',
    labelKey: 'settings.security.humanAuth.roles.orgMember.label',
    summaryKey: 'settings.security.humanAuth.roles.orgMember.summary',
  },
  {
    key: 'project_admin',
    labelKey: 'settings.security.humanAuth.roles.projectAdmin.label',
    summaryKey: 'settings.security.humanAuth.roles.projectAdmin.summary',
  },
  {
    key: 'project_operator',
    labelKey: 'settings.security.humanAuth.roles.projectOperator.label',
    summaryKey: 'settings.security.humanAuth.roles.projectOperator.summary',
  },
  {
    key: 'project_reviewer',
    labelKey: 'settings.security.humanAuth.roles.projectReviewer.label',
    summaryKey: 'settings.security.humanAuth.roles.projectReviewer.summary',
  },
  {
    key: 'project_member',
    labelKey: 'settings.security.humanAuth.roles.projectMember.label',
    summaryKey: 'settings.security.humanAuth.roles.projectMember.summary',
  },
  {
    key: 'project_viewer',
    labelKey: 'settings.security.humanAuth.roles.projectViewer.label',
    summaryKey: 'settings.security.humanAuth.roles.projectViewer.summary',
  },
]

const defaultRoleByScope: Record<ScopeKind, string> = {
  instance: 'instance_admin',
  organization: 'org_member',
  project: 'project_member',
}

export function roleOptionsForScope(scope: ScopeKind) {
  if (scope === 'instance') {
    return roleOptions.filter((option) => option.key === 'instance_admin')
  }
  if (scope === 'organization') {
    return roleOptions.filter((option) => option.key.startsWith('org_'))
  }
  return roleOptions.filter((option) => option.key.startsWith('project_'))
}

export function defaultBindingDraft(roleKey = 'project_member'): BindingDraft {
  return {
    subjectKind: 'user',
    subjectKey: '',
    roleKey,
    expiresAtLocal: '',
  }
}

export function defaultBindingDraftForScope(scope: ScopeKind): BindingDraft {
  return defaultBindingDraft(defaultRoleByScope[scope])
}

export function formatError(caughtError: unknown, fallback: string) {
  return caughtError instanceof ApiError ? caughtError.detail : fallback
}

export function resolveRoleOption(roleKey: string) {
  return roleOptions.find((option) => option.key === roleKey)
}

export function formatTimestamp(value: string | undefined) {
  if (!value) {
    return 'Never'
  }
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }
  return parsed.toLocaleString()
}

const authAuditEventLabels: Record<string, string> = {
  'login.success': 'Login succeeded',
  'login.failed': 'Login failed',
  logout: 'Logged out',
  'session.revoked': 'Session revoked',
  'session.expired': 'Session expired',
  'user.enabled': 'User enabled',
  'user.disabled': 'User disabled',
  'user.disabled_after_login': 'User disabled after login',
}

export function formatAuthAuditEventLabel(eventType: string) {
  return authAuditEventLabels[eventType] ?? eventType
}

export type AuthAuditEventSeverity = 'success' | 'warning' | 'danger' | 'neutral'

const authAuditEventSeverity: Record<string, AuthAuditEventSeverity> = {
  'login.success': 'success',
  'login.failed': 'danger',
  logout: 'neutral',
  'session.revoked': 'warning',
  'session.expired': 'neutral',
  'user.enabled': 'success',
  'user.disabled': 'danger',
  'user.disabled_after_login': 'danger',
}

export function formatAuthAuditEventSeverity(eventType: string): AuthAuditEventSeverity {
  return authAuditEventSeverity[eventType] ?? 'neutral'
}

export const authAuditEventDotClass: Record<AuthAuditEventSeverity, string> = {
  success: 'bg-emerald-500',
  warning: 'bg-amber-500',
  danger: 'bg-red-500',
  neutral: 'bg-muted-foreground/40',
}

export function bindingPlaceholder(subjectKind: SubjectKind) {
  return subjectKind === 'group' ? 'oidc:platform-admins' : 'user@example.com'
}

export function scopeTitle(scope: ScopeKind): TranslationKey {
  if (scope === 'instance') {
    return 'settings.security.humanAuth.scopeTitles.instance'
  }
  return scope === 'organization'
    ? 'settings.security.humanAuth.scopeTitles.organization'
    : 'settings.security.humanAuth.scopeTitles.project'
}

export function createBindingPayload(scope: ScopeKind, draft: BindingDraft) {
  const subjectKey = draft.subjectKey.trim()
  if (!subjectKey) {
    throw new Error('Subject key is required.')
  }

  let expiresAt: string | undefined
  if (draft.expiresAtLocal.trim() !== '') {
    const parsed = new Date(draft.expiresAtLocal)
    if (Number.isNaN(parsed.getTime())) {
      throw new Error('Expiration must be a valid date and time.')
    }
    expiresAt = parsed.toISOString()
  }

  return {
    subject_kind: draft.subjectKind,
    subject_key: subjectKey,
    role_key: draft.roleKey.trim() || defaultRoleByScope[scope],
    expires_at: expiresAt,
  }
}
