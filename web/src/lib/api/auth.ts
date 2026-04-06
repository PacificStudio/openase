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

export type UserDirectoryIdentitySummary = {
  id: string
  issuer: string
  subject: string
  email: string
  emailVerified: boolean
  lastSyncedAt: string
}

export type UserDirectoryEntry = {
  id: string
  status: string
  primaryEmail: string
  displayName: string
  avatarURL: string
  lastLoginAt?: string
  createdAt: string
  updatedAt: string
  primaryIdentity?: UserDirectoryIdentitySummary
}

export type UserDirectoryIdentityDetail = UserDirectoryIdentitySummary & {
  claimsVersion: number
  rawClaimsJSON: string
  createdAt: string
  updatedAt: string
}

export type UserDirectoryGroup = {
  id: string
  issuer: string
  groupKey: string
  groupName: string
  lastSyncedAt: string
}

export type UserStatusAudit = {
  status: string
  reason: string
  source: string
  actorID: string
  changedAt: string
  revokedSessionCount: number
}

export type UserDirectoryDetail = {
  user: UserDirectoryEntry
  identities: UserDirectoryIdentityDetail[]
  groups: UserDirectoryGroup[]
  activeSessionCount: number
  latestStatusAudit?: UserStatusAudit
  recentAuditEvents: AuthAuditEvent[]
}

export type UserStatusTransitionResult = {
  user: UserDirectoryEntry
  changed: boolean
  revokedSessionCount: number
  latestStatusAudit?: UserStatusAudit
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

type RawUserDirectoryIdentitySummary = {
  id?: string
  issuer?: string
  subject?: string
  email?: string
  email_verified?: boolean
  last_synced_at?: string
}

type RawUserDirectoryEntry = {
  id?: string
  status?: string
  primary_email?: string
  display_name?: string
  avatar_url?: string
  last_login_at?: string
  created_at?: string
  updated_at?: string
  primary_identity?: RawUserDirectoryIdentitySummary
}

type RawUserDirectoryIdentityDetail = RawUserDirectoryIdentitySummary & {
  claims_version?: number
  raw_claims_json?: string
  created_at?: string
  updated_at?: string
}

type RawUserDirectoryGroup = {
  id?: string
  issuer?: string
  group_key?: string
  group_name?: string
  last_synced_at?: string
}

type RawUserStatusAudit = {
  status?: string
  reason?: string
  source?: string
  actor_id?: string
  changed_at?: string
  revoked_session_count?: number
}

type RawUserDirectoryDetail = {
  user?: RawUserDirectoryEntry
  identities?: RawUserDirectoryIdentityDetail[]
  groups?: RawUserDirectoryGroup[]
  active_session_count?: number
  latest_status_audit?: RawUserStatusAudit
  recent_audit_events?: RawAuthAuditEvent[]
}

type RawUserStatusTransitionResult = {
  user?: RawUserDirectoryEntry
  changed?: boolean
  revoked_session_count?: number
  latest_status_audit?: RawUserStatusAudit
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

function parseUserDirectoryIdentitySummary(
  raw?: RawUserDirectoryIdentitySummary,
): UserDirectoryIdentitySummary | undefined {
  if (!raw?.id || !raw.issuer || !raw.subject) {
    return undefined
  }
  return {
    id: raw.id,
    issuer: raw.issuer,
    subject: raw.subject,
    email: raw.email ?? '',
    emailVerified: raw.email_verified === true,
    lastSyncedAt: raw.last_synced_at ?? '',
  }
}

function parseUserDirectoryEntry(raw: RawUserDirectoryEntry): UserDirectoryEntry {
  return {
    id: raw.id ?? '',
    status: raw.status ?? 'active',
    primaryEmail: raw.primary_email ?? '',
    displayName: raw.display_name ?? '',
    avatarURL: raw.avatar_url ?? '',
    lastLoginAt: raw.last_login_at ?? undefined,
    createdAt: raw.created_at ?? '',
    updatedAt: raw.updated_at ?? '',
    primaryIdentity: parseUserDirectoryIdentitySummary(raw.primary_identity),
  }
}

function parseUserDirectoryIdentityDetail(
  raw: RawUserDirectoryIdentityDetail,
): UserDirectoryIdentityDetail {
  const summary = parseUserDirectoryIdentitySummary(raw)
  return {
    id: summary?.id ?? '',
    issuer: summary?.issuer ?? '',
    subject: summary?.subject ?? '',
    email: summary?.email ?? '',
    emailVerified: summary?.emailVerified ?? false,
    lastSyncedAt: summary?.lastSyncedAt ?? '',
    claimsVersion: raw.claims_version ?? 0,
    rawClaimsJSON: raw.raw_claims_json ?? '',
    createdAt: raw.created_at ?? '',
    updatedAt: raw.updated_at ?? '',
  }
}

function parseUserDirectoryGroup(raw: RawUserDirectoryGroup): UserDirectoryGroup {
  return {
    id: raw.id ?? '',
    issuer: raw.issuer ?? '',
    groupKey: raw.group_key ?? '',
    groupName: raw.group_name ?? '',
    lastSyncedAt: raw.last_synced_at ?? '',
  }
}

function parseUserStatusAudit(raw?: RawUserStatusAudit): UserStatusAudit | undefined {
  if (!raw?.status) {
    return undefined
  }
  return {
    status: raw.status,
    reason: raw.reason ?? '',
    source: raw.source ?? '',
    actorID: raw.actor_id ?? '',
    changedAt: raw.changed_at ?? '',
    revokedSessionCount: raw.revoked_session_count ?? 0,
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

export async function listInstanceRoleBindings() {
  const payload = await api.get<{ role_bindings?: RawRoleBinding[] }>(
    '/api/v1/instance/role-bindings',
  )
  return parseRoleBindingList(payload)
}

export async function createInstanceRoleBinding(body: {
  subject_kind: string
  subject_key: string
  role_key: string
  expires_at?: string
}) {
  const payload = await api.post<{ role_binding?: RawRoleBinding }>(
    '/api/v1/instance/role-bindings',
    {
      body,
    },
  )
  return payload.role_binding ? parseRoleBinding(payload.role_binding) : null
}

export function deleteInstanceRoleBinding(bindingId: string) {
  return api.delete<void>(`/api/v1/instance/role-bindings/${bindingId}`)
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

export async function listInstanceUsers(params: {
  query?: string
  status?: 'all' | 'active' | 'disabled'
  limit?: number
}) {
  const payload = await api.get<{ users?: RawUserDirectoryEntry[] }>('/api/v1/instance/users', {
    params: {
      q: params.query,
      status: params.status,
      limit: params.limit,
    },
  })
  return Array.isArray(payload.users)
    ? payload.users.map((item) => parseUserDirectoryEntry(item))
    : []
}

export async function getInstanceUserDetail(userId: string): Promise<UserDirectoryDetail> {
  const payload = await api.get<RawUserDirectoryDetail>(`/api/v1/instance/users/${userId}`)
  return {
    user: parseUserDirectoryEntry(payload.user ?? {}),
    identities: Array.isArray(payload.identities)
      ? payload.identities.map((item) => parseUserDirectoryIdentityDetail(item))
      : [],
    groups: Array.isArray(payload.groups)
      ? payload.groups.map((item) => parseUserDirectoryGroup(item))
      : [],
    activeSessionCount: payload.active_session_count ?? 0,
    latestStatusAudit: parseUserStatusAudit(payload.latest_status_audit),
    recentAuditEvents: Array.isArray(payload.recent_audit_events)
      ? payload.recent_audit_events.map((item) => parseAuthAuditEvent(item))
      : [],
  }
}

export async function transitionInstanceUserStatus(
  userId: string,
  body: {
    status: 'active' | 'disabled'
    reason: string
    revoke_sessions?: boolean
  },
): Promise<UserStatusTransitionResult> {
  const payload = await api.post<RawUserStatusTransitionResult>(
    `/api/v1/instance/users/${userId}/status`,
    {
      body,
    },
  )
  return {
    user: parseUserDirectoryEntry(payload.user ?? {}),
    changed: payload.changed === true,
    revokedSessionCount: payload.revoked_session_count ?? 0,
    latestStatusAudit: parseUserStatusAudit(payload.latest_status_audit),
  }
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
