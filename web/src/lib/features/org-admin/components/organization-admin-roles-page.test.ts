import { cleanup, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import OrganizationAdminRolesPage from './organization-admin-roles-page.svelte'

const {
  createOrganizationRoleBinding,
  deleteOrganizationRoleBinding,
  getEffectivePermissions,
  listOrganizationRoleBindings,
} = vi.hoisted(() => ({
  createOrganizationRoleBinding: vi.fn(),
  deleteOrganizationRoleBinding: vi.fn(),
  getEffectivePermissions: vi.fn(),
  listOrganizationRoleBindings: vi.fn(),
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('$lib/api/auth', () => ({
  createOrganizationRoleBinding,
  deleteOrganizationRoleBinding,
  getEffectivePermissions,
  listOrganizationRoleBindings,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

describe('OrganizationAdminRolesPage', () => {
  beforeEach(() => {
    createOrganizationRoleBinding.mockReset()
    deleteOrganizationRoleBinding.mockReset()
    getEffectivePermissions.mockReset()
    listOrganizationRoleBindings.mockReset()
    toastStore.success.mockClear()
    toastStore.error.mockClear()
  })

  afterEach(() => {
    cleanup()
  })

  it('shows member-only binding controls for org admins', async () => {
    getEffectivePermissions.mockResolvedValue({
      user: {
        id: 'user-admin',
        primary_email: 'admin@example.com',
        display_name: 'Admin',
      },
      scope: {
        kind: 'organization',
        id: 'org-1',
      },
      roles: ['org_admin'],
      permissions: ['rbac.manage', 'org.update'],
      groups: [
        {
          group_key: 'oidc:operators',
          group_name: 'Operators',
          issuer: 'https://issuer.example.com',
        },
      ],
    })
    listOrganizationRoleBindings.mockResolvedValue([
      {
        id: 'binding-owner',
        scopeKind: 'organization',
        scopeID: 'org-1',
        subjectKind: 'user',
        subjectKey: 'owner@example.com',
        roleKey: 'org_owner',
        grantedBy: 'user:owner',
        createdAt: '2026-04-06T10:00:00Z',
      },
      {
        id: 'binding-member',
        scopeKind: 'organization',
        scopeID: 'org-1',
        subjectKind: 'group',
        subjectKey: 'oidc:operators',
        roleKey: 'org_member',
        grantedBy: 'user:owner',
        createdAt: '2026-04-06T10:00:00Z',
      },
    ])

    const view = render(OrganizationAdminRolesPage, { organizationId: 'org-1' })

    await waitFor(() => {
      expect(getEffectivePermissions).toHaveBeenCalledWith({ orgId: 'org-1' })
      expect(listOrganizationRoleBindings).toHaveBeenCalledWith('org-1')
    })

    expect(view.getByText('Operators')).toBeTruthy()
    expect(view.getByText('Owner approval required to change privileged bindings.')).toBeTruthy()

    const roleSelect = view.container.querySelectorAll('select')[1]
    const optionValues = Array.from(roleSelect.querySelectorAll('option')).map(
      (option) => option.value,
    )
    expect(optionValues).toEqual(['org_member'])
  })
})
