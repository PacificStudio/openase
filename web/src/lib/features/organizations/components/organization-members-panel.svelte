<script lang="ts">
  import type { OrganizationMembership } from '$lib/api/auth'
  import { Badge } from '$ui/badge'
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

<div class="space-y-4">
  <!-- Header -->
  <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
    <div class="space-y-1">
      <div class="flex flex-wrap items-center gap-2">
        <h2 class="text-base font-semibold">{heading}</h2>
        <Badge variant="secondary">{counts.active} active</Badge>
        {#if counts.invited > 0}
          <Badge variant="outline">{counts.invited} invited</Badge>
        {/if}
        {#if counts.suspended > 0}
          <Badge variant="destructive">{counts.suspended} suspended</Badge>
        {/if}
      </div>
      <p class="text-muted-foreground text-sm">{description}</p>
    </div>
    <div class="flex shrink-0 flex-wrap items-center gap-2">
      <span class="text-muted-foreground text-xs"
        >{counts.owners} owner{counts.owners === 1 ? '' : 's'}</span
      >
      <span class="text-muted-foreground text-xs">·</span>
      <span class="text-muted-foreground text-xs">{membershipsCount} total seats</span>
    </div>
  </div>

  <!-- Invite / access notice -->
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
    <p class="text-muted-foreground text-sm">
      Organization admin access is required to edit memberships, invitations, and ownership.
    </p>
  {/if}

  <!-- Member list -->
  {#if loading}
    <div class="space-y-2">
      {#each { length: 3 } as _}
        <div class="border-border bg-card rounded-lg border p-3">
          <div class="flex items-center gap-3">
            <div class="bg-muted size-8 shrink-0 animate-pulse rounded-full"></div>
            <div class="flex-1 space-y-1.5">
              <div class="bg-muted h-3.5 w-32 animate-pulse rounded"></div>
              <div class="bg-muted h-3 w-48 animate-pulse rounded"></div>
            </div>
            <div class="bg-muted h-6 w-16 animate-pulse rounded-md"></div>
          </div>
        </div>
      {/each}
    </div>
  {:else if filteredMemberships.length === 0}
    <p class="text-muted-foreground py-4 text-sm">{emptyMessage}</p>
  {:else}
    <div class="space-y-2">
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
          onRoleDraftChange={(role) => onRoleDraftChange(entry.id, role)}
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
</div>
