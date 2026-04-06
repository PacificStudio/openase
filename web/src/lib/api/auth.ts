import { api, ApiError } from './client'
import type { HumanAuthSession, HumanAuthUser } from '$lib/stores/auth.svelte'

type FetchLike = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>

type RawAuthSessionResponse = {
  auth_mode?: string
  authenticated?: boolean
  issuer_url?: string
  user?: {
    id?: string
    primary_email?: string
    display_name?: string
    avatar_url?: string
  }
  csrf_token?: string
  roles?: string[]
  permissions?: string[]
}

export type EffectivePermissionsResponse = {
  user: {
    id: string
    primary_email: string
    display_name: string
  }
  scope: {
    kind: string
    id: string
  }
  roles: string[]
  permissions: string[]
  groups: Array<{
    group_key: string
    group_name: string
    issuer: string
  }>
}

export type RoleBinding = {
  id: string
  scopeKind: string
  scopeID: string
  subjectKind: string
  subjectKey: string
  roleKey: string
  grantedBy: string
  expiresAt?: string
  createdAt: string
}

export type ManagedAuthSession = {
  id: string
  current: boolean
  device: {
    kind: string
    os: string
    browser: string
    label: string
  }
  createdAt: string
  lastActiveAt: string
  expiresAt: string
  idleExpiresAt: string
}

export type AuthAuditEvent = {
  id: string
  eventType: string
  actorID: string
  sessionID?: string
  message: string
  metadata: Record<string, unknown>
  createdAt: string
}

export type AuthStepUpCapability = {
  status: string
  summary: string
  supportedMethods: string[]
}

export type SessionGovernanceResponse = {
  authMode: string
  currentSessionID: string
  sessions: ManagedAuthSession[]
  auditEvents: AuthAuditEvent[]
  stepUp: AuthStepUpCapability
}

type RawRoleBinding = {
  id?: string
  scope_kind?: string
  scope_id?: string
  subject_kind?: string
  subject_key?: string
  role_key?: string
  granted_by?: string
  expires_at?: string
  created_at?: string
}

type RawManagedAuthSession = {
  id?: string
  current?: boolean
  device?: {
    kind?: string
    os?: string
    browser?: string
    label?: string
  }
  created_at?: string
  last_active_at?: string
  expires_at?: string
  idle_expires_at?: string
}

type RawAuthAuditEvent = {
  id?: string
  event_type?: string
  actor_id?: string
  session_id?: string
  message?: string
  metadata?: Record<string, unknown>
  created_at?: string
}

type RawSessionGovernanceResponse = {
  auth_mode?: string
  current_session_id?: string
  sessions?: RawManagedAuthSession[]
  audit_events?: RawAuthAuditEvent[]
  step_up?: {
    status?: string
    summary?: string
    supported_methods?: string[]
  }
}

function parseUser(raw?: RawAuthSessionResponse['user']): HumanAuthUser | undefined {
  if (!raw?.id || !raw.primary_email || !raw.display_name) {
    return undefined
  }
  return {
    id: raw.id,
    primaryEmail: raw.primary_email,
    displayName: raw.display_name,
    avatarURL: raw.avatar_url,
  }
}

export function normalizeReturnTo(raw: string | null | undefined) {
  const trimmed = raw?.trim() ?? ''
  if (!trimmed) {
    return '/'
  }
  try {
    const parsed = new URL(trimmed, 'http://openase.local')
    if (parsed.origin !== 'http://openase.local' || !parsed.pathname.startsWith('/')) {
      return '/'
    }
    return `${parsed.pathname}${parsed.search}${parsed.hash}`
  } catch {
    return '/'
  }
}

export async function getAuthSession(fetchFn?: FetchLike): Promise<HumanAuthSession> {
  const execute = fetchFn ?? window.fetch.bind(window)
  const response = await execute('/api/v1/auth/session', {
    method: 'GET',
    credentials: 'same-origin',
  })
  if (!response.ok) {
    throw new ApiError(response.status, await response.text().catch(() => response.statusText))
  }
  const payload = (await response.json()) as RawAuthSessionResponse
  return {
    authMode: payload.auth_mode?.trim() || 'disabled',
    authenticated: payload.authenticated === true,
    issuerURL: payload.issuer_url?.trim() || '',
    user: parseUser(payload.user),
    csrfToken: payload.csrf_token?.trim() || '',
    roles: Array.isArray(payload.roles) ? payload.roles.filter((value) => value.trim() !== '') : [],
    permissions: Array.isArray(payload.permissions)
      ? payload.permissions.filter((value) => value.trim() !== '')
      : [],
  }
}

export function logoutHumanSession() {
  return api.post<void>('/api/v1/auth/logout')
}

export function getEffectivePermissions(params: { projectId?: string; orgId?: string }) {
  return api.get<EffectivePermissionsResponse>('/api/v1/auth/me/permissions', {
    params: {
      project_id: params.projectId,
      org_id: params.orgId,
    },
  })
}

function parseRoleBinding(raw: RawRoleBinding): RoleBinding {
  return {
    id: raw.id ?? '',
    scopeKind: raw.scope_kind ?? '',
    scopeID: raw.scope_id ?? '',
    subjectKind: raw.subject_kind ?? '',
    subjectKey: raw.subject_key ?? '',
    roleKey: raw.role_key ?? '',
    grantedBy: raw.granted_by ?? '',
    expiresAt: raw.expires_at ?? undefined,
    createdAt: raw.created_at ?? '',
  }
}

function parseManagedAuthSession(raw: RawManagedAuthSession): ManagedAuthSession {
  return {
    id: raw.id ?? '',
    current: raw.current === true,
    device: {
      kind: raw.device?.kind ?? 'unknown',
      os: raw.device?.os ?? '',
      browser: raw.device?.browser ?? '',
      label: raw.device?.label ?? 'Unknown device',
    },
    createdAt: raw.created_at ?? '',
    lastActiveAt: raw.last_active_at ?? '',
    expiresAt: raw.expires_at ?? '',
    idleExpiresAt: raw.idle_expires_at ?? '',
  }
}

function parseAuthAuditEvent(raw: RawAuthAuditEvent): AuthAuditEvent {
  return {
    id: raw.id ?? '',
    eventType: raw.event_type ?? '',
    actorID: raw.actor_id ?? '',
    sessionID: raw.session_id ?? undefined,
    message: raw.message ?? '',
    metadata: raw.metadata ?? {},
    createdAt: raw.created_at ?? '',
  }
}

function parseRoleBindingList(payload: { role_bindings?: RawRoleBinding[] }) {
  return Array.isArray(payload.role_bindings)
    ? payload.role_bindings.map((item) => parseRoleBinding(item))
    : []
}

export async function listOrganizationRoleBindings(orgId: string) {
  const payload = await api.get<{ role_bindings?: RawRoleBinding[] }>(
    `/api/v1/organizations/${orgId}/role-bindings`,
  )
  return parseRoleBindingList(payload)
}

export async function createOrganizationRoleBinding(
  orgId: string,
  body: {
    subject_kind: string
    subject_key: string
    role_key: string
    expires_at?: string
  },
) {
  const payload = await api.post<{ role_binding?: RawRoleBinding }>(
    `/api/v1/organizations/${orgId}/role-bindings`,
    { body },
  )
  return payload.role_binding ? parseRoleBinding(payload.role_binding) : null
}

export function deleteOrganizationRoleBinding(orgId: string, bindingId: string) {
  return api.delete<void>(`/api/v1/organizations/${orgId}/role-bindings/${bindingId}`)
}

export async function listProjectRoleBindings(projectId: string) {
  const payload = await api.get<{ role_bindings?: RawRoleBinding[] }>(
    `/api/v1/projects/${projectId}/role-bindings`,
  )
  return parseRoleBindingList(payload)
}

export async function createProjectRoleBinding(
  projectId: string,
  body: {
    subject_kind: string
    subject_key: string
    role_key: string
    expires_at?: string
  },
) {
  const payload = await api.post<{ role_binding?: RawRoleBinding }>(
    `/api/v1/projects/${projectId}/role-bindings`,
    { body },
  )
  return payload.role_binding ? parseRoleBinding(payload.role_binding) : null
}

export function deleteProjectRoleBinding(projectId: string, bindingId: string) {
  return api.delete<void>(`/api/v1/projects/${projectId}/role-bindings/${bindingId}`)
}

export async function getSessionGovernance() {
  const payload = await api.get<RawSessionGovernanceResponse>('/api/v1/auth/sessions')
  return {
    authMode: payload.auth_mode?.trim() || 'disabled',
    currentSessionID: payload.current_session_id?.trim() || '',
    sessions: Array.isArray(payload.sessions) ? payload.sessions.map(parseManagedAuthSession) : [],
    auditEvents: Array.isArray(payload.audit_events)
      ? payload.audit_events.map(parseAuthAuditEvent)
      : [],
    stepUp: {
      status: payload.step_up?.status?.trim() || 'reserved',
      summary: payload.step_up?.summary?.trim() || '',
      supportedMethods: Array.isArray(payload.step_up?.supported_methods)
        ? payload.step_up?.supported_methods.filter((value) => value.trim() !== '')
        : [],
    },
  } satisfies SessionGovernanceResponse
}

export function revokeAuthSession(id: string) {
  return api.delete<void>(`/api/v1/auth/sessions/${id}`)
}

export function revokeAllOtherAuthSessions() {
  return api.post<{ revoked_count: number }>('/api/v1/auth/sessions/revoke-all')
}

export function adminRevokeUserAuthSessions(userId: string) {
  return api.post<{ revoked_count: number; user_id: string }>(
    `/api/v1/auth/users/${userId}/sessions/revoke`,
  )
}
