<script lang="ts">
  import type { OrganizationMembership } from '$lib/api/auth'
  import * as Avatar from '$ui/avatar'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Select from '$ui/select'

  let {
    entry,
    currentUserId = '',
    canManageMembership = false,
    roleDraft = 'member',
    roleEditable = false,
    roleOptions = ['member'],
    roleDirty = false,
    busyState,
    onRoleDraftChange,
    onSaveRole,
    onResend,
    onCancel,
    onTransfer,
    onSuspend,
    onReactivate,
    onRemove,
  }: {
    entry: OrganizationMembership
    currentUserId?: string
    canManageMembership?: boolean
    roleDraft?: 'owner' | 'admin' | 'member'
    roleEditable?: boolean
    roleOptions?: Array<'owner' | 'admin' | 'member'>
    roleDirty?: boolean
    busyState: {
      role: boolean
      resend: boolean
      cancel: boolean
      transfer: boolean
      suspend: boolean
      reactivate: boolean
      remove: boolean
    }
    onRoleDraftChange?: (role: 'owner' | 'admin' | 'member') => void
    onSaveRole?: () => void
    onResend: () => void
    onCancel: () => void
    onTransfer: () => void
    onSuspend: () => void
    onReactivate: () => void
    onRemove: () => void
  } = $props()

  const dateFormatter = new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  })

  function formatTimestamp(value?: string) {
    if (!value) return ''
    const parsed = new Date(value)
    return Number.isNaN(parsed.valueOf()) ? value : dateFormatter.format(parsed)
  }

  function initials() {
    const label = entry.user?.displayName || entry.email
    return label
      .split(/\s+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((part) => part[0]?.toUpperCase() ?? '')
      .join('')
  }

  function roleBadgeVariant(role: string) {
    if (role === 'owner') return 'default'
    if (role === 'admin') return 'secondary'
    return 'outline'
  }

  function statusBadgeVariant(status: string) {
    if (status === 'active') return 'secondary'
    if (status === 'invited') return 'outline'
    return 'destructive'
  }

  function statusCopy() {
    if (entry.status === 'invited' && entry.activeInvitation) {
      return `Invite expires ${formatTimestamp(entry.activeInvitation.expiresAt)}`
    }
    if (entry.status === 'suspended' && entry.suspendedAt) {
      return `Suspended ${formatTimestamp(entry.suspendedAt)}`
    }
    if (entry.status === 'removed' && entry.removedAt) {
      return `Removed ${formatTimestamp(entry.removedAt)}`
    }
    if (entry.status === 'active' && entry.acceptedAt) {
      return `Active since ${formatTimestamp(entry.acceptedAt)}`
    }
    return `Invited ${formatTimestamp(entry.invitedAt)}`
  }
</script>

<div
  class="grid gap-4 rounded-2xl border border-slate-200/80 bg-white/85 p-4 shadow-sm lg:grid-cols-[minmax(0,1fr)_auto]"
>
  <div class="flex min-w-0 gap-3">
    <Avatar.Root class="size-11 border border-slate-200 bg-white">
      {#if entry.user?.avatarURL}
        <Avatar.Image src={entry.user.avatarURL} alt={entry.user.displayName || entry.email} />
      {/if}
      <Avatar.Fallback class="bg-sky-100 text-xs font-semibold text-sky-900">
        {initials()}
      </Avatar.Fallback>
    </Avatar.Root>

    <div class="min-w-0 flex-1 space-y-2">
      <div class="flex flex-wrap items-center gap-2">
        <p class="truncate text-sm font-semibold text-slate-950">
          {entry.user?.displayName || entry.email}
        </p>
        {#if entry.user?.id === currentUserId}
          <Badge variant="secondary">You</Badge>
        {/if}
        <Badge variant={roleBadgeVariant(entry.role)} class="capitalize">{entry.role}</Badge>
        <Badge variant={statusBadgeVariant(entry.status)} class="capitalize">
          {entry.status}
        </Badge>
      </div>

      <div class="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
        <span>{entry.user?.primaryEmail || entry.email}</span>
        <span>{statusCopy()}</span>
        {#if entry.activeInvitation}
          <span>Invite sent {formatTimestamp(entry.activeInvitation.sentAt)}</span>
        {/if}
      </div>

      {#if canManageMembership}
        <div class="flex flex-wrap items-center gap-2">
          {#if roleEditable}
            <div class="min-w-40">
              <Select.Root
                type="single"
                value={roleDraft}
                onValueChange={(value) => {
                  if (value === 'owner' || value === 'admin' || value === 'member') {
                    onRoleDraftChange?.(value)
                  }
                }}
              >
                <Select.Trigger
                  class="h-8 w-full text-xs capitalize"
                  aria-label={`Role for ${entry.email}`}
                  data-testid={`organization-membership-role-${entry.id}`}
                >
                  {roleDraft}
                </Select.Trigger>
                <Select.Content>
                  {#each roleOptions as roleOption (roleOption)}
                    <Select.Item value={roleOption} class="capitalize">{roleOption}</Select.Item>
                  {/each}
                </Select.Content>
              </Select.Root>
            </div>
            <Button
              variant="outline"
              size="sm"
              disabled={!roleDirty || busyState.role}
              onclick={onSaveRole}
            >
              {busyState.role ? 'Saving…' : 'Save role'}
            </Button>
          {:else}
            <span class="text-muted-foreground text-xs">
              {entry.role === 'owner' || entry.role === 'admin'
                ? 'Owner approval required for privileged role changes.'
                : 'Role is fixed by current access policy.'}
            </span>
          {/if}
        </div>
      {/if}
    </div>
  </div>

  <div class="flex flex-wrap items-start justify-end gap-2">
    {#if canManageMembership && entry.status === 'invited' && entry.activeInvitation}
      <Button variant="outline" size="sm" disabled={busyState.resend} onclick={onResend}>
        {busyState.resend ? 'Resending…' : 'Resend'}
      </Button>
      <Button variant="outline" size="sm" disabled={busyState.cancel} onclick={onCancel}>
        {busyState.cancel ? 'Canceling…' : 'Cancel'}
      </Button>
    {/if}

    {#if canManageMembership && entry.status === 'active' && entry.role !== 'owner'}
      <Button size="sm" disabled={busyState.transfer} onclick={onTransfer}>
        {busyState.transfer ? 'Promoting…' : 'Make owner'}
      </Button>
      <Button variant="outline" size="sm" disabled={busyState.suspend} onclick={onSuspend}>
        {busyState.suspend ? 'Suspending…' : 'Suspend'}
      </Button>
    {/if}

    {#if canManageMembership && entry.status === 'suspended'}
      <Button variant="outline" size="sm" disabled={busyState.reactivate} onclick={onReactivate}>
        {busyState.reactivate ? 'Reactivating…' : 'Reactivate'}
      </Button>
    {/if}

    {#if canManageMembership && (entry.status === 'active' || entry.status === 'suspended')}
      <Button variant="outline" size="sm" disabled={busyState.remove} onclick={onRemove}>
        {busyState.remove ? 'Removing…' : 'Remove'}
      </Button>
    {/if}
  </div>
</div>
