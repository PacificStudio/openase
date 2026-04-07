import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'
import { authStore } from '$lib/stores/auth.svelte'
import { OrganizationAdminPage } from '$lib/features/organizations'
import { currentOrg } from './security-settings.test-helpers'
import {
  createdOrganizationUserBinding,
  hydrateOidcAuth,
  mockEffectivePermissionsByScope,
  organizationGroupBinding,
} from './security-settings-human-auth.fixtures'

vi.hoisted(() => {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation(() => ({
      matches: false,
      media: '',
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
})

const {
  createOrganizationRoleBinding,
  deleteOrganizationRoleBinding,
  getEffectivePermissions,
  listOrganizationMemberships,
  listOrganizationRoleBindings,
} = vi.hoisted(() => ({
  createOrganizationRoleBinding: vi.fn(),
  deleteOrganizationRoleBinding: vi.fn(),
  getEffectivePermissions: vi.fn(),
  listOrganizationMemberships: vi.fn(),
  listOrganizationRoleBindings: vi.fn(),
}))

vi.mock('$lib/api/auth', () => ({
  createOrganizationRoleBinding,
  deleteOrganizationRoleBinding,
  getEffectivePermissions,
  listOrganizationMemberships,
  listOrganizationRoleBindings,
  cancelOrganizationInvitation: vi.fn(),
  inviteOrganizationMember: vi.fn(),
  resendOrganizationInvitation: vi.fn(),
  transferOrganizationOwnership: vi.fn(),
  updateOrganizationMembership: vi.fn(),
}))

vi.mock('$app/navigation', () => ({
  invalidateAll: vi.fn().mockResolvedValue(undefined),
}))

describe('Organization admin RBAC', () => {
  afterEach(() => {
    cleanup()
    authStore.clear()
    appStore.currentOrg = null
    vi.clearAllMocks()
  })

  it('creates an organization role binding from the org admin surface', async () => {
    hydrateOidcAuth()
    appStore.currentOrg = currentOrg()
    getEffectivePermissions.mockImplementation(mockEffectivePermissionsByScope)
    listOrganizationRoleBindings.mockResolvedValue([organizationGroupBinding()])
    listOrganizationMemberships.mockResolvedValue([])
    createOrganizationRoleBinding.mockResolvedValue(createdOrganizationUserBinding())

    const { findByText, findAllByPlaceholderText } = render(OrganizationAdminPage, {
      organizationId: currentOrg().id,
    })

    expect(await findByText('Organization effective access')).toBeTruthy()
    expect(await findByText('Members')).toBeTruthy()

    const orgSection = (await findByText('Organization RBAC')).closest(
      '.border-border',
    ) as HTMLElement
    const subjectInputs = await findAllByPlaceholderText('user@example.com')
    const orgInput = subjectInputs.find((element) => orgSection.contains(element as Node))
    expect(orgInput).toBeTruthy()
    await fireEvent.input(orgInput as HTMLElement, { target: { value: 'bob@example.com' } })

    const addButton = within(orgSection).getByRole('button', { name: 'Add binding' })
    await fireEvent.click(addButton)

    await waitFor(() => {
      expect(createOrganizationRoleBinding).toHaveBeenCalledWith(currentOrg().id, {
        subject_kind: 'user',
        subject_key: 'bob@example.com',
        role_key: 'org_member',
        expires_at: undefined,
      })
    })
  })
})
