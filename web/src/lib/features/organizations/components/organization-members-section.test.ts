import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { authStore } from '$lib/stores/auth.svelte'
import OrganizationMembersSection from './organization-members-section.svelte'

const {
  listOrganizationMemberships,
  inviteOrganizationMember,
  resendOrganizationInvitation,
  cancelOrganizationInvitation,
  transferOrganizationOwnership,
  updateOrganizationMembership,
} = vi.hoisted(() => ({
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
    inviteOrganizationMember.mockReset()
    resendOrganizationInvitation.mockReset()
    cancelOrganizationInvitation.mockReset()
    transferOrganizationOwnership.mockReset()
    updateOrganizationMembership.mockReset()
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
        {
          id: 'membership-owner',
          organizationID: 'org-1',
          userID: 'user-owner',
          email: 'owner@example.com',
          role: 'owner',
          status: 'active',
          invitedBy: 'system',
          invitedAt: '2026-04-05T10:00:00Z',
          acceptedAt: '2026-04-05T10:05:00Z',
          createdAt: '2026-04-05T10:00:00Z',
          updatedAt: '2026-04-05T10:05:00Z',
          user: {
            id: 'user-owner',
            primaryEmail: 'owner@example.com',
            displayName: 'Owner',
          },
        },
      ])
      .mockResolvedValueOnce([
        {
          id: 'membership-owner',
          organizationID: 'org-1',
          userID: 'user-owner',
          email: 'owner@example.com',
          role: 'owner',
          status: 'active',
          invitedBy: 'system',
          invitedAt: '2026-04-05T10:00:00Z',
          acceptedAt: '2026-04-05T10:05:00Z',
          createdAt: '2026-04-05T10:00:00Z',
          updatedAt: '2026-04-05T10:05:00Z',
          user: {
            id: 'user-owner',
            primaryEmail: 'owner@example.com',
            displayName: 'Owner',
          },
        },
        {
          id: 'membership-invitee',
          organizationID: 'org-1',
          email: 'invitee@example.com',
          role: 'member',
          status: 'invited',
          invitedBy: 'user:user-owner',
          invitedAt: '2026-04-06T11:00:00Z',
          createdAt: '2026-04-06T11:00:00Z',
          updatedAt: '2026-04-06T11:00:00Z',
          activeInvitation: {
            id: 'invitation-1',
            status: 'pending',
            email: 'invitee@example.com',
            role: 'member',
            invitedBy: 'user:user-owner',
            sentAt: '2026-04-06T11:00:00Z',
            expiresAt: '2026-04-13T11:00:00Z',
          },
        },
      ])

    inviteOrganizationMember.mockResolvedValue({
      membership: {
        id: 'membership-invitee',
        organizationID: 'org-1',
        email: 'invitee@example.com',
        role: 'member',
        status: 'invited',
        invitedBy: 'user:user-owner',
        invitedAt: '2026-04-06T11:00:00Z',
        createdAt: '2026-04-06T11:00:00Z',
        updatedAt: '2026-04-06T11:00:00Z',
      },
      invitation: {
        id: 'invitation-1',
        status: 'pending',
        email: 'invitee@example.com',
        role: 'member',
        invitedBy: 'user:user-owner',
        sentAt: '2026-04-06T11:00:00Z',
        expiresAt: '2026-04-13T11:00:00Z',
      },
      acceptToken: 'accept-token-123',
    })

    const view = render(OrganizationMembersSection, {
      organizationId: 'org-1',
    })

    await waitFor(() => {
      expect(listOrganizationMemberships).toHaveBeenCalledWith(
        'org-1',
        expect.objectContaining({ signal: expect.any(AbortSignal) }),
      )
    })
    expect(view.getByText('Owner')).toBeTruthy()
    expect(view.getByText('1 owner')).toBeTruthy()

    await fireEvent.input(view.getByLabelText('Invite by email'), {
      target: { value: 'invitee@example.com' },
    })
    await fireEvent.click(view.getByRole('button', { name: 'Send invite' }))

    await waitFor(() => {
      expect(inviteOrganizationMember).toHaveBeenCalledWith('org-1', {
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
})
