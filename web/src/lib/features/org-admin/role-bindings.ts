type SubjectKind = 'user' | 'group'

type BindingDraft = {
  subjectKind: SubjectKind
  subjectKey: string
  roleKey: string
  expiresAtLocal: string
}

type RoleOption = {
  key: string
  label: string
  summary: string
}

const roleOptions: RoleOption[] = [
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
]

function defaultBindingDraftForScope(): BindingDraft {
  return {
    subjectKind: 'user',
    subjectKey: '',
    roleKey: 'org_member',
    expiresAtLocal: '',
  }
}

function resolveRoleOption(roleKey: string) {
  return roleOptions.find((option) => option.key === roleKey)
}

function formatTimestamp(value: string | undefined) {
  if (!value) return 'Never'
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString()
}

function createBindingPayload(draft: BindingDraft) {
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
    role_key: draft.roleKey.trim() || 'org_member',
    expires_at: expiresAt,
  }
}

export { createBindingPayload, defaultBindingDraftForScope, formatTimestamp, resolveRoleOption }
export type { BindingDraft }
