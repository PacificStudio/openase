<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'

  let {
    inviteEmail = $bindable(''),
    inviteRole = $bindable<'owner' | 'admin' | 'member'>('member'),
    roleOptions = ['owner', 'admin', 'member'],
    submittingInvite = false,
    recentInviteToken = '',
    recentInviteEmail = '',
    onInvite,
    onCopyToken,
  }: {
    inviteEmail?: string
    inviteRole?: 'owner' | 'admin' | 'member'
    roleOptions?: Array<'owner' | 'admin' | 'member'>
    submittingInvite?: boolean
    recentInviteToken?: string
    recentInviteEmail?: string
    onInvite: () => void
    onCopyToken: () => void
  } = $props()
</script>

<div
  class="border-border bg-card grid gap-4 rounded-lg border p-4 lg:grid-cols-[minmax(0,1.4fr)_220px_auto] lg:items-end"
>
  <div class="space-y-2">
    <Label for="organization-member-email">Invite by email</Label>
    <Input
      id="organization-member-email"
      bind:value={inviteEmail}
      type="email"
      placeholder="teammate@example.com"
    />
  </div>

  <div class="space-y-2">
    <Label>Baseline role</Label>
    <Select.Root
      type="single"
      value={inviteRole}
      onValueChange={(value) => {
        if (
          (value === 'owner' || value === 'admin' || value === 'member') &&
          roleOptions.includes(value)
        ) {
          inviteRole = value
        }
      }}
    >
      <Select.Trigger class="w-full capitalize">{inviteRole}</Select.Trigger>
      <Select.Content>
        {#each roleOptions as roleOption (roleOption)}
          <Select.Item value={roleOption} class="capitalize">{roleOption}</Select.Item>
        {/each}
      </Select.Content>
    </Select.Root>
  </div>

  <Button class="min-w-32" onclick={onInvite} disabled={submittingInvite}>
    {submittingInvite ? 'Sending…' : 'Send invite'}
  </Button>
</div>

{#if recentInviteToken}
  <div class="border-border bg-card rounded-lg border p-4">
    <div class="flex flex-col gap-2 lg:flex-row lg:items-start lg:justify-between">
      <div class="space-y-1">
        <p class="text-sm font-medium">
          Latest accept token for {recentInviteEmail}
        </p>
        <p class="text-muted-foreground text-xs">
          Useful for local testing until a delivery channel is wired.
        </p>
      </div>
      <Button variant="outline" size="sm" onclick={onCopyToken}>Copy token</Button>
    </div>
    <code class="bg-muted text-foreground mt-3 block overflow-x-auto rounded-md px-3 py-2 text-xs">
      {recentInviteToken}
    </code>
  </div>
{/if}
