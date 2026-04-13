import { i18nStore } from '$lib/i18n/store.svelte'

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
    get label() {
      return i18nStore.t('orgAdmin.roles.owner.label')
    },
    get summary() {
      return i18nStore.t('orgAdmin.roles.owner.summary')
    },
  },
  {
    key: 'org_admin',
    get label() {
      return i18nStore.t('orgAdmin.roles.admin.label')
    },
    get summary() {
      return i18nStore.t('orgAdmin.roles.admin.summary')
    },
  },
  {
    key: 'org_member',
    get label() {
      return i18nStore.t('orgAdmin.roles.member.label')
    },
    get summary() {
      return i18nStore.t('orgAdmin.roles.member.summary')
    },
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
  if (!value) return i18nStore.t('common.never')
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
