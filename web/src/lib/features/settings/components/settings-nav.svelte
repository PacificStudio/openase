<script lang="ts">
  import { cn } from '$lib/utils'
  import { Settings, GitBranch, Columns3, Bot, Bell, Shield, Archive } from '@lucide/svelte'
  import type { Component } from 'svelte'
  import type { SettingsSection } from '../types'

  let {
    active,
    onSelect,
  }: {
    active: SettingsSection
    onSelect: (section: SettingsSection) => void
  } = $props()

  type NavItem = {
    key: SettingsSection
    label: string
    icon: Component
  }

  const items: NavItem[] = [
    { key: 'general', label: 'General', icon: Settings },
    { key: 'repositories', label: 'Repositories', icon: GitBranch },
    { key: 'statuses', label: 'Statuses', icon: Columns3 },
    { key: 'agents', label: 'Agents', icon: Bot },
    { key: 'notifications', label: 'Notifications', icon: Bell },
    { key: 'security', label: 'Security', icon: Shield },
    { key: 'archived', label: 'Archived Tickets', icon: Archive },
  ]
</script>

<nav class="w-[200px] shrink-0 space-y-0.5">
  {#each items as item (item.key)}
    {@const Icon = item.icon}
    <button
      type="button"
      class={cn(
        'flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors',
        active === item.key
          ? 'bg-muted text-foreground font-medium'
          : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground',
      )}
      onclick={() => onSelect(item.key)}
    >
      <Icon class="size-4 shrink-0" />
      {item.label}
    </button>
  {/each}
</nav>
