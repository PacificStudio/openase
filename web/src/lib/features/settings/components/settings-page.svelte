<script lang="ts">
  import type { SettingsSection } from '../types'
  import AgentSettings from './agent-settings.svelte'
  import ConnectorsSettings from './connectors-settings.svelte'
  import SettingsNav from './settings-nav.svelte'
  import GeneralSettings from './general-settings.svelte'
  import NotificationSettings from './notification-settings.svelte'
  import RepositoriesSettings from './repositories-settings.svelte'
  import SecuritySettings from './security-settings.svelte'
  import StatusSettings from './status-settings.svelte'
  import WorkflowSettings from './workflow-settings.svelte'

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
      <RepositoriesSettings />
    {:else if activeSection === 'workflows'}
      <WorkflowSettings />
    {:else if activeSection === 'agents'}
      <AgentSettings />
    {:else if activeSection === 'connectors'}
      <ConnectorsSettings />
    {:else if activeSection === 'notifications'}
      <NotificationSettings />
    {:else if activeSection === 'security'}
      <SecuritySettings />
    {/if}
  </div>
</div>
