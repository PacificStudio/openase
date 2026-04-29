<script lang="ts">
  import { cn } from '$lib/utils'
  import { Settings, GitBranch, Columns3, Bot, Bell, Shield, Archive } from '@lucide/svelte'
  import type { Component } from 'svelte'
  import type { SettingsSection } from '../types'
  import type { TranslationKey } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    active,
    onSelect,
  }: {
    active: SettingsSection
    onSelect: (section: SettingsSection) => void
  } = $props()

  type NavItem = {
    key: SettingsSection
    labelKey: TranslationKey
    icon: Component
  }

  const items: NavItem[] = [
    { key: 'general', labelKey: 'settings.nav.labels.general', icon: Settings },
    { key: 'repositories', labelKey: 'settings.nav.labels.repositories', icon: GitBranch },
    { key: 'statuses', labelKey: 'settings.nav.labels.statuses', icon: Columns3 },
    { key: 'agents', labelKey: 'settings.nav.labels.agents', icon: Bot },
    { key: 'notifications', labelKey: 'settings.nav.labels.notifications', icon: Bell },
    { key: 'security', labelKey: 'settings.nav.labels.security', icon: Shield },
    { key: 'archived', labelKey: 'settings.nav.labels.archived', icon: Archive },
  ]
</script>

<nav
  class="flex w-full shrink-0 flex-wrap gap-1 pb-1 lg:w-[200px] lg:flex-col lg:gap-0.5"
  data-tour="settings-nav"
>
  {#each items as item (item.key)}
    {@const Icon = item.icon}
    <button
      type="button"
      class={cn(
        'flex shrink-0 items-center gap-2.5 rounded-md px-3 py-2 text-sm whitespace-nowrap transition-colors lg:w-full',
        active === item.key
          ? 'bg-muted text-foreground font-medium'
          : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground',
      )}
      onclick={() => onSelect(item.key)}
    >
      <Icon class="size-4 shrink-0" />
      {i18nStore.t(item.labelKey)}
    </button>
  {/each}
</nav>
