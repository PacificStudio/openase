import { ApiError } from '$lib/api/client'

export type ScopeKind = 'instance' | 'organization' | 'project'
export type SubjectKind = 'user' | 'group'

export type RoleOption = {
  key: string
  label: string
  summary: string
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
    label: 'Instance Admin',
    summary: 'Full control across orgs, projects, security, jobs, and RBAC.',
  },
  {
    key: 'org_owner',
    label: 'Org Owner',
    summary: 'Full organization and project control, including RBAC.',
  },
  {
    key: 'org_admin',
    label: 'Org Admin',
    summary: 'Manage organization settings and descendant project operations.',
  },
  {
    key: 'org_member',
    label: 'Org Member',
    summary: 'Read organization resources and perform standard project work.',
  },
  {
    key: 'project_admin',
    label: 'Project Admin',
    summary: 'Manage project settings, repos, workflows, jobs, security, and bindings.',
  },
  {
    key: 'project_operator',
    label: 'Project Operator',
    summary: 'Operate project runtime surfaces without full security or RBAC control.',
  },
  {
    key: 'project_reviewer',
    label: 'Project Reviewer',
    summary: 'Review conversations, tickets, and proposals with approval capability.',
  },
  {
    key: 'project_member',
    label: 'Project Member',
    summary: 'Standard contributor access for tickets, comments, and conversations.',
  },
  {
    key: 'project_viewer',
    label: 'Project Viewer',
    summary: 'Read-only access to project state and diagnostics.',
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

export function bindingPlaceholder(subjectKind: SubjectKind) {
  return subjectKind === 'group' ? 'oidc:platform-admins' : 'user@example.com'
}

export function scopeTitle(scope: ScopeKind) {
  if (scope === 'instance') {
    return 'Instance RBAC'
  }
  return scope === 'organization' ? 'Organization RBAC' : 'Project RBAC'
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
