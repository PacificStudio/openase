const orgId = 'org-1'
const baseTimestamps = {
  invitedAt: '2026-04-05T10:00:00Z',
  createdAt: '2026-04-05T10:00:00Z',
  updatedAt: '2026-04-05T10:05:00Z',
}

function makeMembership({
  id,
  userID,
  email,
  displayName,
  role,
  status,
  invitedBy = 'system',
  invitedAt = baseTimestamps.invitedAt,
  acceptedAt = status === 'active' ? '2026-04-05T10:05:00Z' : undefined,
  createdAt = baseTimestamps.createdAt,
  updatedAt = baseTimestamps.updatedAt,
  activeInvitation,
}: {
  id: string
  userID?: string
  email: string
  displayName?: string
  role: 'owner' | 'admin' | 'member'
  status: 'active' | 'invited'
  invitedBy?: string
  invitedAt?: string
  acceptedAt?: string
  createdAt?: string
  updatedAt?: string
  activeInvitation?: Record<string, string>
}) {
  return {
    id,
    organizationID: orgId,
    userID,
    email,
    role,
    status,
    invitedBy,
    invitedAt,
    acceptedAt,
    createdAt,
    updatedAt,
    user: userID
      ? {
          id: userID,
          primaryEmail: email,
          displayName: displayName ?? email,
        }
      : undefined,
    activeInvitation,
  }
}

function makeInvitation(overrides: Partial<Record<string, string>> = {}) {
  return {
    id: 'invitation-1',
    status: 'pending',
    email: 'invitee@example.com',
    role: 'member',
    invitedBy: 'user:user-owner',
    sentAt: '2026-04-06T11:00:00Z',
    expiresAt: '2026-04-13T11:00:00Z',
    ...overrides,
  }
}

export { makeInvitation, makeMembership, orgId }
