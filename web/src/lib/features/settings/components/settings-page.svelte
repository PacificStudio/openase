<script lang="ts">
  import type { SettingsSection } from '../types'
  import SettingsNav from './settings-nav.svelte'
  import GeneralSettings from './general-settings.svelte'
  import NotificationSettings from './notification-settings.svelte'
  import SettingsPlaceholder from './settings-placeholder.svelte'
  import StatusSettings from './status-settings.svelte'

  let activeSection = $state<SettingsSection>('general')

  function handleSelect(section: SettingsSection) {
    activeSection = section
  }
</script>

<div class="flex gap-8">
  <SettingsNav active={activeSection} onSelect={handleSelect} />

  <div class="min-w-0 flex-1">
    {#if activeSection === 'general'}
      <GeneralSettings />
    {:else if activeSection === 'statuses'}
      <StatusSettings />
    {:else if activeSection === 'repositories'}
      <SettingsPlaceholder section="repositories" title="Repositories" />
    {:else if activeSection === 'workflows'}
      <SettingsPlaceholder section="workflows" title="Workflows" />
    {:else if activeSection === 'agents'}
      <SettingsPlaceholder section="agents" title="Agents" />
    {:else if activeSection === 'connectors'}
      <SettingsPlaceholder section="connectors" title="Connectors" />
    {:else if activeSection === 'notifications'}
      <NotificationSettings />
    {:else if activeSection === 'security'}
      <SettingsPlaceholder section="security" title="Security" />
    {/if}
  </div>
</div>
