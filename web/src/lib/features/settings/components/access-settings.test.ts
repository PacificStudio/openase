import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'
import { authStore } from '$lib/stores/auth.svelte'
import AccessSettings from './access-settings.svelte'
import { configuredSecurity, currentOrg, currentProject } from './security-settings.test-helpers'
import {
  effectivePermissionsMock,
  hydrateOidcAuth,
  oidcUser,
} from './security-settings-human-auth.fixtures'

const { getSecuritySettings } = vi.hoisted(() => ({
  getSecuritySettings: vi.fn(),
}))

const {
  createProjectRoleBinding,
  deleteProjectRoleBinding,
  getEffectivePermissions,
  listProjectRoleBindings,
} = vi.hoisted(() => ({
  createProjectRoleBinding: vi.fn(),
  deleteProjectRoleBinding: vi.fn(),
  getEffectivePermissions: vi.fn(),
  listProjectRoleBindings: vi.fn(),
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('$lib/api/openase', () => ({
  getSecuritySettings,
}))

vi.mock('$lib/api/auth', () => ({
  createProjectRoleBinding,
  deleteProjectRoleBinding,
  getEffectivePermissions,
  listProjectRoleBindings,
  normalizeReturnTo: vi.fn((value?: string | null) => value?.trim() || '/'),
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

describe('Access settings', () => {
  afterEach(() => {
    cleanup()
    authStore.clear()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('shows migration guidance for local bootstrap access without loading oidc access state', async () => {
    authStore.hydrate({
      authMode: 'oidc',
      loginRequired: false,
      authenticated: true,
      principalKind: 'local_bootstrap',
      authConfigured: true,
      sessionGovernanceAvailable: false,
      canManageAuth: true,
      roles: ['instance_admin'],
      permissions: ['security_setting.read', 'security_setting.update'],
    })
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })

    const { findByText, queryByText } = render(AccessSettings)

    expect(await findByText('Local bootstrap project access')).toBeTruthy()
    expect(queryByText('Project effective access')).toBeNull()
    expect(getEffectivePermissions).not.toHaveBeenCalled()
    expect(listProjectRoleBindings).not.toHaveBeenCalled()
  })

  it('renders oidc project access and creates a project role binding', async () => {
    hydrateOidcAuth()
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    getEffectivePermissions.mockResolvedValue(
      effectivePermissionsMock('project', currentProject().id),
    )
    listProjectRoleBindings.mockResolvedValue([
      {
        id: 'binding-existing',
        scopeKind: 'project',
        scopeID: currentProject().id,
        subjectKind: 'group',
        subjectKey: 'platform-admins',
        roleKey: 'project_admin',
        grantedBy: 'user:user-1',
        createdAt: '2026-04-04T09:00:00Z',
      },
    ])
    createProjectRoleBinding.mockResolvedValue({
      id: 'binding-created',
      scopeKind: 'project',
      scopeID: currentProject().id,
      subjectKind: 'user',
      subjectKey: 'bob@example.com',
      roleKey: 'project_member',
      grantedBy: 'user:user-1',
      createdAt: '2026-04-04T10:00:00Z',
    })

    const { findByText, findByPlaceholderText } = render(AccessSettings)

    expect(await findByText('Project effective access')).toBeTruthy()
    expect(await findByText(oidcUser().displayName)).toBeTruthy()
    expect(await findByText('Platform Admins')).toBeTruthy()

    const projectSection = (await findByText('Project RBAC')).closest(
      '.border-border',
    ) as HTMLElement
    const subjectInput = await findByPlaceholderText('user@example.com')
    expect(projectSection.contains(subjectInput)).toBe(true)

    await fireEvent.input(subjectInput, { target: { value: 'bob@example.com' } })
    await fireEvent.click(within(projectSection).getByRole('button', { name: 'Add binding' }))

    await waitFor(() => {
      expect(createProjectRoleBinding).toHaveBeenCalledWith(currentProject().id, {
        subject_kind: 'user',
        subject_key: 'bob@example.com',
        role_key: 'project_member',
        expires_at: undefined,
      })
    })
  })
})
