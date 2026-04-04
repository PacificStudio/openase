<script lang="ts">
  import * as Avatar from '$ui/avatar'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import { LogOut, Moon, Settings } from '@lucide/svelte'

  let {
    userDisplayName = '',
    userPrimaryEmail = '',
    userAvatarURL = '',
    userInitials = 'U',
    logoutPending = false,
    settingsEnabled = false,
    settingsHref = '',
    onToggleTheme,
    onOpenSettings,
    onWarmSettings,
    onLogout,
  }: {
    userDisplayName?: string
    userPrimaryEmail?: string
    userAvatarURL?: string
    userInitials?: string
    logoutPending?: boolean
    settingsEnabled?: boolean
    settingsHref?: string
    onToggleTheme?: () => void
    onOpenSettings?: () => void
    onWarmSettings?: (href: string) => void
    onLogout?: () => void
  } = $props()
</script>

<DropdownMenu.Root>
  <DropdownMenu.Trigger>
    {#snippet child({ props })}
      <button {...props} class="rounded-full">
        <Avatar.Root class="size-7">
          {#if userAvatarURL}
            <Avatar.Image src={userAvatarURL} alt={userDisplayName || userPrimaryEmail || 'User'} />
          {/if}
          <Avatar.Fallback class="bg-primary/10 text-primary text-xs"
            >{userInitials}</Avatar.Fallback
          >
        </Avatar.Root>
      </button>
    {/snippet}
  </DropdownMenu.Trigger>
  <DropdownMenu.Content align="end" class="w-48">
    {#if userDisplayName || userPrimaryEmail}
      <DropdownMenu.Label class="space-y-0.5">
        {#if userDisplayName}
          <div class="text-foreground text-sm font-medium">{userDisplayName}</div>
        {/if}
        {#if userPrimaryEmail}
          <div class="text-muted-foreground text-xs">{userPrimaryEmail}</div>
        {/if}
      </DropdownMenu.Label>
      <DropdownMenu.Separator />
    {/if}
    <DropdownMenu.Item onclick={onToggleTheme}>
      <Moon class="mr-2 size-4" />
      Toggle Theme
    </DropdownMenu.Item>
    <DropdownMenu.Item
      onclick={onOpenSettings}
      onpointerenter={() => {
        if (settingsHref) {
          onWarmSettings?.(settingsHref)
        }
      }}
      disabled={!settingsEnabled}
    >
      <Settings class="mr-2 size-4" />
      Settings
    </DropdownMenu.Item>
    <DropdownMenu.Separator />
    <DropdownMenu.Item onclick={onLogout} disabled={logoutPending}>
      <LogOut class="mr-2 size-4" />
      {logoutPending ? 'Logging out…' : 'Logout'}
    </DropdownMenu.Item>
  </DropdownMenu.Content>
</DropdownMenu.Root>
