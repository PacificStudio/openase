<script lang="ts">
  import { invalidateAll } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import {
    cancelOrganizationInvitation,
    inviteOrganizationMember,
    listOrganizationMemberships,
    resendOrganizationInvitation,
    transferOrganizationOwnership,
    updateOrganizationMembership,
    type OrganizationMembership,
  } from '$lib/api/auth'
  import { authStore } from '$lib/stores/auth.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Badge } from '$ui/badge'
  import * as Card from '$ui/card'
  import OrganizationMembersInvitePanel from './organization-members-invite-panel.svelte'
  import OrganizationMemberRow from './organization-member-row.svelte'

  let { organizationId }: { organizationId: string } = $props()

  const currentUserId = $derived(authStore.user?.id ?? '')
  let memberships = $state<OrganizationMembership[]>([])
  let loading = $state(false)
  let inviteEmail = $state('')
  let inviteRole = $state<'owner' | 'admin' | 'member'>('member')
  let submittingInvite = $state(false)
  let recentInviteToken = $state('')
  let recentInviteEmail = $state('')
  let busyKeys = $state<Set<string>>(new Set())
  const counts = $derived({
    owners: memberships.filter((item) => item.role === 'owner' && item.status === 'active').length,
    active: memberships.filter((item) => item.status === 'active').length,
    invited: memberships.filter((item) => item.status === 'invited').length,
    suspended: memberships.filter((item) => item.status === 'suspended').length,
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

  async function loadMemberships(signal?: AbortSignal) {
    if (!organizationId) {
      memberships = []
      return
    }
    loading = true
    try {
      memberships = await listOrganizationMemberships(organizationId, { signal })
    } catch (caughtError) {
      if (signal?.aborted) return
      memberships = []
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
      return
    }

    const controller = new AbortController()
    void loadMemberships(controller.signal)
    return () => {
      controller.abort()
    }
  })
</script>

<Card.Root
  class="overflow-hidden rounded-3xl border border-sky-100/80 bg-gradient-to-br from-sky-50 via-white to-emerald-50 shadow-sm"
>
  <Card.Header class="gap-4 border-b border-sky-100/80 bg-white/70 backdrop-blur">
    <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
      <div class="space-y-1.5">
        <div class="flex flex-wrap items-center gap-2">
          <Card.Title>Members</Card.Title>
          <Badge variant="secondary">{counts.active} active</Badge>
          <Badge variant="outline">{counts.invited} invited</Badge>
          {#if counts.suspended > 0}
            <Badge variant="destructive">{counts.suspended} suspended</Badge>
          {/if}
        </div>
        <Card.Description>
          Owners and admins handle invites here. Active memberships drive org visibility and project
          authorization.
        </Card.Description>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <Badge variant="default">{counts.owners} owner{counts.owners === 1 ? '' : 's'}</Badge>
        <Badge variant="outline">{memberships.length} total seats</Badge>
      </div>
    </div>
  </Card.Header>

  <Card.Content class="space-y-5 p-6">
    <OrganizationMembersInvitePanel
      bind:inviteEmail
      bind:inviteRole
      {submittingInvite}
      {recentInviteToken}
      {recentInviteEmail}
      onInvite={handleInvite}
      onCopyToken={async () => {
        try {
          await navigator.clipboard.writeText(recentInviteToken)
          toastStore.success('Accept token copied.')
        } catch {
          toastStore.error('Failed to copy accept token.')
        }
      }}
    />

    {#if loading}
      <div class="text-muted-foreground rounded-2xl border border-dashed px-4 py-8 text-sm">
        Loading organization members…
      </div>
    {:else if memberships.length === 0}
      <div class="text-muted-foreground rounded-2xl border border-dashed px-4 py-8 text-sm">
        No memberships yet.
      </div>
    {:else}
      <div class="space-y-3">
        {#each memberships as entry (entry.id)}
          <OrganizationMemberRow
            {entry}
            {currentUserId}
            busyState={{
              resend: isBusy(`resend:${entry.id}`),
              cancel: isBusy(`cancel:${entry.id}`),
              transfer: isBusy(`transfer:${entry.id}`),
              suspend: isBusy(`suspended:${entry.id}`),
              reactivate: isBusy(`active:${entry.id}`),
              remove: isBusy(`removed:${entry.id}`),
            }}
            onResend={() => handleInvitationAction(entry, 'resend')}
            onCancel={() => handleInvitationAction(entry, 'cancel')}
            onTransfer={() => handleTransferOwnership(entry)}
            onSuspend={() => handleMembershipStatus(entry, 'suspended')}
            onReactivate={() => handleMembershipStatus(entry, 'active')}
            onRemove={() => handleMembershipStatus(entry, 'removed')}
          />
        {/each}
      </div>
    {/if}
  </Card.Content>
</Card.Root>
