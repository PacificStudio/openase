<script lang="ts">
  import { invalidateAll } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import {
    cancelOrganizationInvitation,
    getEffectivePermissions,
    inviteOrganizationMember,
    listOrganizationMemberships,
    resendOrganizationInvitation,
    transferOrganizationOwnership,
    updateOrganizationMembership,
    type EffectivePermissionsResponse,
    type OrganizationMembership,
  } from '$lib/api/auth'
  import { authStore } from '$lib/stores/auth.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import OrganizationMembersPanel from './organization-members-panel.svelte'

  type MembershipRole = 'owner' | 'admin' | 'member'
  type SectionMode = 'all' | 'members' | 'invitations'

  let {
    organizationId,
    mode = 'all',
    heading = 'Members',
    description = 'Owners and admins handle invites here. Active memberships drive org visibility and project authorization.',
    emptyMessage = 'No memberships yet.',
  }: {
    organizationId: string
    mode?: SectionMode
    heading?: string
    description?: string
    emptyMessage?: string
  } = $props()

  const currentUserId = $derived(authStore.user?.id ?? '')
  let memberships = $state<OrganizationMembership[]>([])
  let orgPermissions = $state<EffectivePermissionsResponse | null>(null)
  let loading = $state(false)
  let inviteEmail = $state('')
  let inviteRole = $state<MembershipRole>('member')
  let submittingInvite = $state(false)
  let recentInviteToken = $state('')
  let recentInviteEmail = $state('')
  let roleDrafts = $state<Record<string, MembershipRole>>({})
  let busyKeys = $state<Set<string>>(new Set())

  const counts = $derived({
    owners: memberships.filter((item) => item.role === 'owner' && item.status === 'active').length,
    active: memberships.filter((item) => item.status === 'active').length,
    invited: memberships.filter((item) => item.status === 'invited').length,
    suspended: memberships.filter((item) => item.status === 'suspended').length,
  })
  const effectiveRoles = $derived(orgPermissions?.roles ?? [])
  const canManageMemberships = $derived(orgPermissions?.permissions.includes('org.update') ?? false)
  const canManagePrivilegedRoles = $derived(
    effectiveRoles.includes('instance_admin') || effectiveRoles.includes('org_owner'),
  )
  const filteredMemberships = $derived.by(() => {
    if (mode === 'invitations') {
      return memberships.filter((item) => item.activeInvitation)
    }
    if (mode === 'members') {
      return memberships.filter((item) => !item.activeInvitation)
    }
    return memberships
  })

  function setBusy(key: string, active: boolean) {
    if (active) {
      busyKeys = new Set([...busyKeys, key])
      return
    }
    const next = new Set(busyKeys)
    next.delete(key)
    busyKeys = next
  }
  function isBusy(key: string) {
    return busyKeys.has(key)
  }
  function syncRoleDrafts(items: OrganizationMembership[]) {
    const nextDrafts: Record<string, MembershipRole> = {}
    for (const item of items) {
      if (item.role === 'owner' || item.role === 'admin' || item.role === 'member') {
        nextDrafts[item.id] = item.role
      }
    }
    roleDrafts = nextDrafts
  }
  async function loadMemberships(signal?: AbortSignal) {
    if (!organizationId) {
      memberships = []
      orgPermissions = null
      return
    }
    loading = true
    try {
      const [nextMemberships, nextPermissions] = await Promise.all([
        listOrganizationMemberships(organizationId, { signal }),
        getEffectivePermissions({ orgId: organizationId }),
      ])
      memberships = nextMemberships
      orgPermissions = nextPermissions
      syncRoleDrafts(nextMemberships)
      if (
        !nextPermissions.roles.includes('instance_admin') &&
        !nextPermissions.roles.includes('org_owner')
      ) {
        inviteRole = 'member'
      }
    } catch (caughtError) {
      if (signal?.aborted) return
      memberships = []
      orgPermissions = null
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : 'Failed to load organization members.',
      )
    } finally {
      if (!signal?.aborted) loading = false
    }
  }
  async function refreshAfterMutation(successMessage: string) {
    await loadMemberships()
    await invalidateAll()
    toastStore.success(successMessage)
  }
  function canManageEntry(entry: OrganizationMembership) {
    if (!canManageMemberships) {
      return false
    }
    if (canManagePrivilegedRoles) {
      return true
    }
    return entry.role === 'member'
  }
  function roleOptionsForEntry(entry: OrganizationMembership): MembershipRole[] {
    if (!canManageEntry(entry)) {
      return [entry.role as MembershipRole]
    }
    if (canManagePrivilegedRoles) {
      return ['owner', 'admin', 'member']
    }
    return ['member']
  }
  async function handleInvite() {
    const email = inviteEmail.trim()
    if (!email) {
      toastStore.error('Invite email is required.')
      return
    }

    submittingInvite = true
    try {
      const result = await inviteOrganizationMember(organizationId, {
        email,
        role: inviteRole,
      })
      recentInviteToken = result.acceptToken
      recentInviteEmail = result.membership.email
      inviteEmail = ''
      if (!canManagePrivilegedRoles) {
        inviteRole = 'member'
      }
      await refreshAfterMutation(`Invitation sent to ${result.membership.email}.`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : 'Failed to invite organization member.',
      )
    } finally {
      submittingInvite = false
    }
  }
  async function handleSaveRole(entry: OrganizationMembership) {
    const role = roleDrafts[entry.id] ?? (entry.role as MembershipRole)
    if (role === entry.role) {
      return
    }

    const busyKey = `role:${entry.id}`
    setBusy(busyKey, true)
    try {
      await updateOrganizationMembership(organizationId, entry.id, { role })
      await refreshAfterMutation(`${entry.email} is now ${role}.`)
    } catch (caughtError) {
      roleDrafts = { ...roleDrafts, [entry.id]: entry.role as MembershipRole }
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : `Failed to update ${entry.email}.`,
      )
    } finally {
      setBusy(busyKey, false)
    }
  }
  async function handleInvitationAction(
    entry: OrganizationMembership,
    action: 'resend' | 'cancel',
  ) {
    const invitationId = entry.activeInvitation?.id
    if (!invitationId) return

    const busyKey = `${action}:${entry.id}`
    setBusy(busyKey, true)
    try {
      if (action === 'resend') {
        const result = await resendOrganizationInvitation(organizationId, invitationId)
        recentInviteToken = result.acceptToken
        recentInviteEmail = result.membership.email
        await refreshAfterMutation(`Invitation resent to ${result.membership.email}.`)
      } else {
        await cancelOrganizationInvitation(organizationId, invitationId)
        await refreshAfterMutation(`Invitation canceled for ${entry.email}.`)
      }
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : `Failed to ${action} invitation for ${entry.email}.`,
      )
    } finally {
      setBusy(busyKey, false)
    }
  }
  async function handleMembershipStatus(
    entry: OrganizationMembership,
    status: 'active' | 'suspended' | 'removed',
  ) {
    const busyKey = `${status}:${entry.id}`
    setBusy(busyKey, true)
    try {
      await updateOrganizationMembership(organizationId, entry.id, { status })
      const verb =
        status === 'active' ? 'reactivated' : status === 'suspended' ? 'suspended' : 'removed'
      await refreshAfterMutation(`${entry.email} ${verb}.`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : `Failed to update ${entry.email}.`,
      )
    } finally {
      setBusy(busyKey, false)
    }
  }
  async function handleTransferOwnership(entry: OrganizationMembership) {
    const busyKey = `transfer:${entry.id}`
    setBusy(busyKey, true)
    try {
      await transferOrganizationOwnership(organizationId, entry.id, {
        previous_owner_role: 'admin',
      })
      await refreshAfterMutation(`${entry.email} is now an owner.`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : `Failed to transfer ownership to ${entry.email}.`,
      )
    } finally {
      setBusy(busyKey, false)
    }
  }
  $effect(() => {
    if (!organizationId) {
      memberships = []
      orgPermissions = null
      return
    }

    const controller = new AbortController()
    void loadMemberships(controller.signal)
    return () => {
      controller.abort()
    }
  })
</script>

<!-- prettier-ignore -->
<OrganizationMembersPanel {heading} {description} {counts} membershipsCount={memberships.length}
  {canManageMemberships} {canManagePrivilegedRoles} bind:inviteEmail bind:inviteRole
  {submittingInvite} {recentInviteToken} {recentInviteEmail} {loading} {emptyMessage}
  {filteredMemberships} {currentUserId} {roleDrafts} {canManageEntry} {roleOptionsForEntry}
  {isBusy} onInvite={handleInvite}
  onCopyToken={async () => {
    try {
      await navigator.clipboard.writeText(recentInviteToken)
      toastStore.success('Accept token copied.')
    } catch {
      toastStore.error('Failed to copy accept token.')
    }
  }}
  onRoleDraftChange={(entryId, role) => {
    roleDrafts = { ...roleDrafts, [entryId]: role }
  }}
  onSaveRole={handleSaveRole} onResend={(entry) => handleInvitationAction(entry, 'resend')}
  onCancel={(entry) => handleInvitationAction(entry, 'cancel')} onTransfer={handleTransferOwnership}
  onSuspend={(entry) => handleMembershipStatus(entry, 'suspended')}
  onReactivate={(entry) => handleMembershipStatus(entry, 'active')}
  onRemove={(entry) => handleMembershipStatus(entry, 'removed')} />
