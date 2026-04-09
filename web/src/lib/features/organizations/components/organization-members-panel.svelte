<script lang="ts">
  import type { OrganizationMembership } from '$lib/api/auth'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Mail, Plus, X } from '@lucide/svelte'
  import OrganizationMemberRow from './organization-member-row.svelte'
  import OrganizationMembersInvitePanel from './organization-members-invite-panel.svelte'

  type MembershipRole = 'owner' | 'admin' | 'member'

  let {
    heading,
    description,
    emptyMessage,
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
    pendingInvitations,
    activeMembers,
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
    emptyMessage: string
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
    pendingInvitations: OrganizationMembership[]
    activeMembers: OrganizationMembership[]
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

  let inviteOpen = $state(false)

  $effect(() => {
    // auto-open invite panel when a recent invite clears the email field on success
    if (!inviteEmail && inviteOpen && !submittingInvite) {
      // stay open so user can send another
    }
  })

  function memberRow(entry: OrganizationMembership) {
    return {
      entry,
      currentUserId,
      canManageMembership: canManageEntry(entry),
      roleDraft: roleDrafts[entry.id] ?? (entry.role as MembershipRole),
      roleEditable: canManageEntry(entry),
      roleOptions: roleOptionsForEntry(entry),
      roleDirty: (roleDrafts[entry.id] ?? entry.role) !== entry.role,
      busyState: {
        role: isBusy(`role:${entry.id}`),
        resend: isBusy(`resend:${entry.id}`),
        cancel: isBusy(`cancel:${entry.id}`),
        transfer: isBusy(`transfer:${entry.id}`),
        suspend: isBusy(`suspended:${entry.id}`),
        reactivate: isBusy(`active:${entry.id}`),
        remove: isBusy(`removed:${entry.id}`),
      },
    }
  }
</script>

<div class="space-y-4">
  <!-- Header -->
  <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
    <div class="space-y-1">
      <div class="flex flex-wrap items-center gap-2">
        <h2 class="text-base font-semibold">{heading}</h2>
        <Badge variant="secondary">{counts.active} active</Badge>
        {#if counts.invited > 0}
          <Badge variant="outline">{counts.invited} pending</Badge>
        {/if}
        {#if counts.suspended > 0}
          <Badge variant="destructive">{counts.suspended} suspended</Badge>
        {/if}
      </div>
      <p class="text-muted-foreground text-sm">{description}</p>
    </div>
    <div class="flex shrink-0 flex-wrap items-center gap-2">
      <span class="text-muted-foreground text-xs">
        {counts.owners}
        {counts.owners === 1 ? 'owner' : 'owners'}
      </span>
      <span class="text-muted-foreground text-xs">·</span>
      <span class="text-muted-foreground text-xs">{membershipsCount} total seats</span>
      {#if canManageMemberships}
        <Button size="sm" variant="outline" onclick={() => (inviteOpen = !inviteOpen)}>
          {#if inviteOpen}
            <X class="size-3.5" />
            Cancel
          {:else}
            <Plus class="size-3.5" />
            Invite
          {/if}
        </Button>
      {/if}
    </div>
  </div>

  {#if !canManageMemberships}
    <p class="text-muted-foreground text-sm">
      Organization admin access is required to edit memberships, invitations, and ownership.
    </p>
  {:else if inviteOpen}
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
  {/if}

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
  {:else}
    <!-- Pending invitations -->
    {#if pendingInvitations.length > 0}
      <div class="space-y-2">
        <div class="flex items-center gap-2">
          <Mail class="text-muted-foreground size-3.5" />
          <span class="text-muted-foreground text-xs font-medium">
            Pending · {pendingInvitations.length}
          </span>
        </div>
        {#each pendingInvitations as entry (entry.id)}
          {@const row = memberRow(entry)}
          <OrganizationMemberRow
            {...row}
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

    <!-- Active + suspended members -->
    {#if activeMembers.length > 0}
      <div class="space-y-2">
        {#if pendingInvitations.length > 0}
          <span class="text-muted-foreground text-xs font-medium">
            Members · {activeMembers.length}
          </span>
        {/if}
        {#each activeMembers as entry (entry.id)}
          {@const row = memberRow(entry)}
          <OrganizationMemberRow
            {...row}
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
    {:else if pendingInvitations.length === 0}
      <p class="text-muted-foreground py-4 text-sm">{emptyMessage}</p>
    {/if}
  {/if}
</div>
