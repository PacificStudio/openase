<script lang="ts">
  import { SUPPORTED_LOCALES, type AppLocale } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import * as Avatar from '$ui/avatar'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import { LifeBuoy, LogOut, Moon, Settings } from '@lucide/svelte'

  let {
    userDisplayName = '',
    userPrimaryEmail = '',
    userAvatarURL = '',
    userInitials = 'U',
    logoutPending = false,
    settingsEnabled = false,
    settingsHref = '',
    restartTourEnabled = false,
    onToggleTheme,
    onOpenSettings,
    onWarmSettings,
    onRestartTour,
    onLogout,
  }: {
    userDisplayName?: string
    userPrimaryEmail?: string
    userAvatarURL?: string
    userInitials?: string
    logoutPending?: boolean
    settingsEnabled?: boolean
    settingsHref?: string
    restartTourEnabled?: boolean
    onToggleTheme?: () => void
    onOpenSettings?: () => void
    onWarmSettings?: (href: string) => void
    onRestartTour?: () => void
    onLogout?: () => void
  } = $props()

  const localeOptions: readonly AppLocale[] = SUPPORTED_LOCALES
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
    <DropdownMenu.Label class="text-muted-foreground text-xs">
      {i18nStore.t('common.language')}
    </DropdownMenu.Label>
    {#each localeOptions as locale}
      <DropdownMenu.Item
        onclick={() => i18nStore.setLocale(locale)}
        disabled={i18nStore.locale === locale}
      >
        <span>{i18nStore.labelForLocale(locale)}</span>
        {#if i18nStore.locale === locale}
          <span class="text-muted-foreground ml-auto text-[10px]">
            {i18nStore.t('layout.currentLanguage', {
              language: i18nStore.labelForLocale(locale),
            })}
          </span>
        {/if}
      </DropdownMenu.Item>
    {/each}
    <DropdownMenu.Separator />
    <DropdownMenu.Item onclick={onToggleTheme}>
      <Moon class="mr-2 size-4" />
      {i18nStore.t('layout.toggleTheme')}
    </DropdownMenu.Item>
    <DropdownMenu.Item onclick={onRestartTour} disabled={!restartTourEnabled}>
      <LifeBuoy class="mr-2 size-4" />
      {i18nStore.t('layout.restartTour')}
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
      {i18nStore.t('layout.settings')}
    </DropdownMenu.Item>
    <DropdownMenu.Separator />
    <DropdownMenu.Item onclick={onLogout} disabled={logoutPending}>
      <LogOut class="mr-2 size-4" />
      {logoutPending ? i18nStore.t('layout.loggingOut') : i18nStore.t('layout.logout')}
    </DropdownMenu.Item>
  </DropdownMenu.Content>
</DropdownMenu.Root>
