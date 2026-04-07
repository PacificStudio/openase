import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

import { authStore } from '$lib/stores/auth.svelte'
import OrganizationMembersSection from './organization-members-section.svelte'
import { makeInvitation, makeMembership, orgId } from './organization-members-section.test-helpers'

const {
  getEffectivePermissions,
  listOrganizationMemberships,
  inviteOrganizationMember,
  resendOrganizationInvitation,
  cancelOrganizationInvitation,
  transferOrganizationOwnership,
  updateOrganizationMembership,
} = vi.hoisted(() => ({
  getEffectivePermissions: vi.fn(),
  listOrganizationMemberships: vi.fn(),
  inviteOrganizationMember: vi.fn(),
  resendOrganizationInvitation: vi.fn(),
  cancelOrganizationInvitation: vi.fn(),
  transferOrganizationOwnership: vi.fn(),
  updateOrganizationMembership: vi.fn(),
}))

const { invalidateAll } = vi.hoisted(() => ({
  invalidateAll: vi.fn().mockResolvedValue(undefined),
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('$app/navigation', () => ({
  invalidateAll,
}))

vi.mock('$lib/api/auth', () => ({
  getEffectivePermissions,
  listOrganizationMemberships,
  inviteOrganizationMember,
  resendOrganizationInvitation,
  cancelOrganizationInvitation,
  transferOrganizationOwnership,
  updateOrganizationMembership,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

describe('OrganizationMembersSection', () => {
  beforeAll(() => {
    HTMLElement.prototype.scrollIntoView ??= vi.fn()
    HTMLElement.prototype.hasPointerCapture ??= vi.fn(() => false)
    HTMLElement.prototype.releasePointerCapture ??= vi.fn()
    globalThis.ResizeObserver ??= class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  })

  beforeEach(() => {
    authStore.hydrate({
      authMode: 'oidc',
      authenticated: true,
      csrfToken: 'csrf-token',
      roles: ['org_owner'],
      permissions: ['org.update'],
      user: {
        id: 'user-owner',
        primaryEmail: 'owner@example.com',
        displayName: 'Owner',
      },
    })

    listOrganizationMemberships.mockReset()
    getEffectivePermissions.mockReset()
    inviteOrganizationMember.mockReset()
    resendOrganizationInvitation.mockReset()
    cancelOrganizationInvitation.mockReset()
    transferOrganizationOwnership.mockReset()
    updateOrganizationMembership.mockReset()
    getEffectivePermissions.mockResolvedValue({
      user: {
        id: 'user-owner',
        primary_email: 'owner@example.com',
        display_name: 'Owner',
      },
      scope: {
        kind: 'organization',
        id: orgId,
      },
      roles: ['org_owner'],
      permissions: ['org.update', 'rbac.manage'],
      groups: [],
    })
    invalidateAll.mockClear()
    toastStore.success.mockClear()
    toastStore.error.mockClear()
  })

  afterEach(() => {
    cleanup()
    authStore.clear()
  })

  it('loads members and submits a new invite', async () => {
    listOrganizationMemberships
      .mockResolvedValueOnce([
        makeMembership({
          id: 'membership-owner',
          userID: 'user-owner',
          email: 'owner@example.com',
          displayName: 'Owner',
          role: 'owner',
          status: 'active',
        }),
      ])
      .mockResolvedValueOnce([
        makeMembership({
          id: 'membership-owner',
          userID: 'user-owner',
          email: 'owner@example.com',
          displayName: 'Owner',
          role: 'owner',
          status: 'active',
        }),
        makeMembership({
          id: 'membership-invitee',
          email: 'invitee@example.com',
          role: 'member',
          status: 'invited',
          invitedBy: 'user:user-owner',
          invitedAt: '2026-04-06T11:00:00Z',
          createdAt: '2026-04-06T11:00:00Z',
          updatedAt: '2026-04-06T11:00:00Z',
          activeInvitation: makeInvitation(),
        }),
      ])

    inviteOrganizationMember.mockResolvedValue({
      membership: makeMembership({
        id: 'membership-invitee',
        email: 'invitee@example.com',
        role: 'member',
        status: 'invited',
        invitedBy: 'user:user-owner',
        invitedAt: '2026-04-06T11:00:00Z',
        createdAt: '2026-04-06T11:00:00Z',
        updatedAt: '2026-04-06T11:00:00Z',
      }),
      invitation: makeInvitation(),
      acceptToken: 'accept-token-123',
    })

    const view = render(OrganizationMembersSection, {
      organizationId: orgId,
    })

    await waitFor(() => {
      expect(listOrganizationMemberships).toHaveBeenCalledWith(
        orgId,
        expect.objectContaining({ signal: expect.any(AbortSignal) }),
      )
      expect(getEffectivePermissions).toHaveBeenCalledWith({ orgId })
    })
    expect(view.getByText('Owner')).toBeTruthy()
    expect(view.getByText('1 owner')).toBeTruthy()

    await fireEvent.input(view.getByLabelText('Invite by email'), {
      target: { value: 'invitee@example.com' },
    })
    await fireEvent.click(view.getByRole('button', { name: 'Send invite' }))

    await waitFor(() => {
      expect(inviteOrganizationMember).toHaveBeenCalledWith(orgId, {
        email: 'invitee@example.com',
        role: 'member',
      })
    })
    await waitFor(() => {
      expect(listOrganizationMemberships).toHaveBeenCalledTimes(2)
    })

    expect(view.getAllByText('invitee@example.com')).toHaveLength(2)
    expect(view.getByText('Latest accept token for invitee@example.com')).toBeTruthy()
    expect(view.getByText('accept-token-123')).toBeTruthy()
    expect(invalidateAll).toHaveBeenCalled()
    expect(toastStore.success).toHaveBeenCalledWith('Invitation sent to invitee@example.com.')
  })

  it('saves a role change for an existing member', async () => {
    listOrganizationMemberships
      .mockResolvedValueOnce([
        makeMembership({
          id: 'membership-member',
          userID: 'user-member',
          email: 'member@example.com',
          displayName: 'Member',
          role: 'member',
          status: 'active',
        }),
      ])
      .mockResolvedValueOnce([
        makeMembership({
          id: 'membership-member',
          userID: 'user-member',
          email: 'member@example.com',
          displayName: 'Member',
          role: 'admin',
          status: 'active',
        }),
      ])

    updateOrganizationMembership.mockResolvedValue(
      makeMembership({
        id: 'membership-member',
        userID: 'user-member',
        email: 'member@example.com',
        role: 'admin',
        status: 'active',
        displayName: 'Member',
      }),
    )

    const view = render(OrganizationMembersSection, {
      organizationId: orgId,
    })

    await waitFor(() => {
      expect(view.getByText('Member')).toBeTruthy()
    })

    await fireEvent.click(view.getByRole('button', { name: 'Save role' }))
    expect(updateOrganizationMembership).not.toHaveBeenCalled()

    const roleTrigger = view.getByTestId('organization-membership-role-membership-member')
    await fireEvent.pointerDown(roleTrigger)
    await fireEvent.keyDown(roleTrigger, { key: 'ArrowUp' })
    const adminOption = document.querySelector(
      '[data-slot="select-item"][data-value="admin"]',
    ) as HTMLElement | null
    expect(adminOption).toBeTruthy()
    await fireEvent.pointerUp(adminOption as HTMLElement)
    await fireEvent.click(adminOption as HTMLElement)

    const saveButton = view.getByRole('button', { name: 'Save role' })
    await waitFor(() => {
      expect(roleTrigger.textContent ?? '').toContain('admin')
      expect(saveButton.hasAttribute('disabled')).toBe(false)
    })
    await fireEvent.click(saveButton)

    await waitFor(() => {
      expect(updateOrganizationMembership).toHaveBeenCalledWith(orgId, 'membership-member', {
        role: 'admin',
      })
    })
    expect(toastStore.success).toHaveBeenCalledWith('member@example.com is now admin.')
  })
})
