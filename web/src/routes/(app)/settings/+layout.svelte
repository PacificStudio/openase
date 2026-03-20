<script lang="ts">
  import { page } from '$app/state'
  import SettingsSectionLayout, {
    type SettingsNavItem,
  } from '$lib/components/layout/SettingsSectionLayout.svelte'

  let { children } = $props()

  const items = $derived.by<SettingsNavItem[]>(() => [
    {
      label: 'General',
      href: '/settings',
      description: 'Project summary, repositories, and runtime defaults.',
    },
    {
      label: 'Connectors',
      href: '/settings/connectors',
      description: 'Issue ingestion, sync health, and delivery wiring.',
    },
    {
      label: 'Notifications',
      href: '/settings/notifications',
      description: 'Channels, rules, and delivery tests.',
    },
    {
      label: 'Workflows',
      description: 'Workflow defaults and harness governance.',
      disabled: true,
    },
    {
      label: 'Agents',
      description: 'Provider defaults, safety controls, and budgets.',
      disabled: true,
    },
    {
      label: 'Status Columns',
      description: 'Board topology and WIP definitions.',
      disabled: true,
    },
  ])
</script>

<SettingsSectionLayout
  title="Settings"
  description="Keep configuration in a secondary navigation instead of stacking giant forms into the main workspace."
  {items}
  currentPath={page.url.pathname}
>
  {@render children()}
</SettingsSectionLayout>
