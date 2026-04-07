<script lang="ts">
  import type { OrganizationMembership } from '$lib/api/auth'
  import { Badge } from '$ui/badge'
  import * as Card from '$ui/card'
  import OrganizationMemberRow from './organization-member-row.svelte'
  import OrganizationMembersInvitePanel from './organization-members-invite-panel.svelte'

  type MembershipRole = 'owner' | 'admin' | 'member'

  let {
    heading,
    description,
    counts,
    membershipsCount,
    canManageMemberships,
    canManagePrivilegedRoles,
    inviteEmail = $bindable(''),
    inviteRole = $bindable('member'),
    submittingInvite,
    recentInviteToken,
    recentInviteEmail,
    loading,
    emptyMessage,
    filteredMemberships,
    currentUserId,
    roleDrafts,
    canManageEntry,
    roleOptionsForEntry,
    isBusy,
    onInvite,
    onCopyToken,
    onRoleDraftChange,
    onSaveRole,
    onResend,
    onCancel,
    onTransfer,
    onSuspend,
    onReactivate,
    onRemove,
  }: {
    heading: string
    description: string
    counts: {
      owners: number
      active: number
      invited: number
      suspended: number
    }
    membershipsCount: number
    canManageMemberships: boolean
    canManagePrivilegedRoles: boolean
    inviteEmail?: string
    inviteRole?: MembershipRole
    submittingInvite: boolean
    recentInviteToken: string
    recentInviteEmail: string
    loading: boolean
    emptyMessage: string
    filteredMemberships: OrganizationMembership[]
    currentUserId: string
    roleDrafts: Record<string, MembershipRole>
    canManageEntry: (entry: OrganizationMembership) => boolean
    roleOptionsForEntry: (entry: OrganizationMembership) => MembershipRole[]
    isBusy: (key: string) => boolean
    onInvite: () => void
    onCopyToken: () => Promise<void>
    onRoleDraftChange: (entryId: string, role: MembershipRole) => void
    onSaveRole: (entry: OrganizationMembership) => void
    onResend: (entry: OrganizationMembership) => void
    onCancel: (entry: OrganizationMembership) => void
    onTransfer: (entry: OrganizationMembership) => void
    onSuspend: (entry: OrganizationMembership) => void
    onReactivate: (entry: OrganizationMembership) => void
    onRemove: (entry: OrganizationMembership) => void
  } = $props()
</script>

<Card.Root
  class="overflow-hidden rounded-3xl border border-sky-100/80 bg-gradient-to-br from-sky-50 via-white to-emerald-50 shadow-sm"
>
  <Card.Header class="gap-4 border-b border-sky-100/80 bg-white/70 backdrop-blur">
    <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
      <div class="space-y-1.5">
        <div class="flex flex-wrap items-center gap-2">
          <Card.Title>{heading}</Card.Title>
          <Badge variant="secondary">{counts.active} active</Badge>
          <Badge variant="outline">{counts.invited} invited</Badge>
          {#if counts.suspended > 0}
            <Badge variant="destructive">{counts.suspended} suspended</Badge>
          {/if}
        </div>
        <Card.Description>{description}</Card.Description>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <Badge variant="default">{counts.owners} owner{counts.owners === 1 ? '' : 's'}</Badge>
        <Badge variant="outline">{membershipsCount} total seats</Badge>
        {#if canManagePrivilegedRoles}
          <Badge variant="secondary">Privileged role changes enabled</Badge>
        {:else if canManageMemberships}
          <Badge variant="outline">Member lifecycle only</Badge>
        {/if}
      </div>
    </div>
  </Card.Header>

  <Card.Content class="space-y-5 p-6">
    {#if canManageMemberships}
      <OrganizationMembersInvitePanel
        bind:inviteEmail
        bind:inviteRole
        roleOptions={canManagePrivilegedRoles ? ['owner', 'admin', 'member'] : ['member']}
        {submittingInvite}
        {recentInviteToken}
        {recentInviteEmail}
        {onInvite}
        {onCopyToken}
      />
    {:else}
      <div class="text-muted-foreground rounded-2xl border border-dashed px-4 py-4 text-sm">
        Organization admin access is required to edit memberships, invitations, and ownership.
      </div>
    {/if}

    {#if loading}
      <div class="text-muted-foreground rounded-2xl border border-dashed px-4 py-8 text-sm">
        Loading organization members…
      </div>
    {:else if filteredMemberships.length === 0}
      <div class="text-muted-foreground rounded-2xl border border-dashed px-4 py-8 text-sm">
        {emptyMessage}
      </div>
    {:else}
      <div class="space-y-3">
        {#each filteredMemberships as entry (entry.id)}
          <OrganizationMemberRow
            {entry}
            {currentUserId}
            canManageMembership={canManageEntry(entry)}
            roleDraft={roleDrafts[entry.id] ?? (entry.role as MembershipRole)}
            roleEditable={canManageEntry(entry)}
            roleOptions={roleOptionsForEntry(entry)}
            roleDirty={(roleDrafts[entry.id] ?? entry.role) !== entry.role}
            busyState={{
              role: isBusy(`role:${entry.id}`),
              resend: isBusy(`resend:${entry.id}`),
              cancel: isBusy(`cancel:${entry.id}`),
              transfer: isBusy(`transfer:${entry.id}`),
              suspend: isBusy(`suspended:${entry.id}`),
              reactivate: isBusy(`active:${entry.id}`),
              remove: isBusy(`removed:${entry.id}`),
            }}
            onRoleDraftChange={(role) => {
              onRoleDraftChange(entry.id, role)
            }}
            onSaveRole={() => onSaveRole(entry)}
            onResend={() => onResend(entry)}
            onCancel={() => onCancel(entry)}
            onTransfer={() => onTransfer(entry)}
            onSuspend={() => onSuspend(entry)}
            onReactivate={() => onReactivate(entry)}
            onRemove={() => onRemove(entry)}
          />
        {/each}
      </div>
    {/if}
  </Card.Content>
</Card.Root>
