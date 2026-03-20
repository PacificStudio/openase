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

  <div class="flex-1 min-w-0">
    {#if activeSection === 'general'}
      <GeneralSettings />
    {:else if activeSection === 'statuses'}
      <StatusSettings />
    {:else if activeSection === 'repositories'}
      <div class="max-w-lg space-y-4">
        <h2 class="text-base font-semibold text-foreground">Repositories</h2>
        <p class="text-sm text-muted-foreground">
          Connected repository management coming soon.
        </p>
      </div>
    {:else if activeSection === 'workflows'}
      <div class="max-w-lg space-y-4">
        <h2 class="text-base font-semibold text-foreground">Workflows</h2>
        <p class="text-sm text-muted-foreground">
          Workflow configuration coming soon.
        </p>
      </div>
    {:else if activeSection === 'agents'}
      <div class="max-w-lg space-y-4">
        <h2 class="text-base font-semibold text-foreground">Agents</h2>
        <p class="text-sm text-muted-foreground">
          Agent pool settings coming soon.
        </p>
      </div>
    {:else if activeSection === 'connectors'}
      <div class="max-w-lg space-y-4">
        <h2 class="text-base font-semibold text-foreground">Connectors</h2>
        <p class="text-sm text-muted-foreground">
          Connector management coming soon.
        </p>
      </div>
    {:else if activeSection === 'notifications'}
      <div class="max-w-lg space-y-4">
        <h2 class="text-base font-semibold text-foreground">Notifications</h2>
        <p class="text-sm text-muted-foreground">
          Notification preferences coming soon.
        </p>
      </div>
    {:else if activeSection === 'security'}
      <div class="max-w-lg space-y-4">
        <h2 class="text-base font-semibold text-foreground">Security</h2>
        <p class="text-sm text-muted-foreground">
          Security and access control coming soon.
        </p>
      </div>
    {/if}
  </div>
</div>
