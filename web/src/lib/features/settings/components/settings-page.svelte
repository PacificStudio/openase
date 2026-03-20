<script lang="ts">
  import type { SettingsSection } from '../types'
  import SettingsNav from './settings-nav.svelte'
  import GeneralSettings from './general-settings.svelte'
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
      <div class="max-w-lg space-y-4">
        <h2 class="text-foreground text-base font-semibold">Repositories</h2>
        <p class="text-muted-foreground text-sm">Connected repository management coming soon.</p>
      </div>
    {:else if activeSection === 'workflows'}
      <div class="max-w-lg space-y-4">
        <h2 class="text-foreground text-base font-semibold">Workflows</h2>
        <p class="text-muted-foreground text-sm">Workflow configuration coming soon.</p>
      </div>
    {:else if activeSection === 'agents'}
      <div class="max-w-lg space-y-4">
        <h2 class="text-foreground text-base font-semibold">Agents</h2>
        <p class="text-muted-foreground text-sm">Agent pool settings coming soon.</p>
      </div>
    {:else if activeSection === 'connectors'}
      <div class="max-w-lg space-y-4">
        <h2 class="text-foreground text-base font-semibold">Connectors</h2>
        <p class="text-muted-foreground text-sm">Connector management coming soon.</p>
      </div>
    {:else if activeSection === 'notifications'}
      <div class="max-w-lg space-y-4">
        <h2 class="text-foreground text-base font-semibold">Notifications</h2>
        <p class="text-muted-foreground text-sm">Notification preferences coming soon.</p>
      </div>
    {:else if activeSection === 'security'}
      <div class="max-w-lg space-y-4">
        <h2 class="text-foreground text-base font-semibold">Security</h2>
        <p class="text-muted-foreground text-sm">Security and access control coming soon.</p>
      </div>
    {/if}
  </div>
</div>
