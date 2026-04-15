<script lang="ts">
  import { onMount } from 'svelte'
  import { PageScaffold } from '$lib/components/layout'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { currentHashSelection, writeHashSelection } from '$lib/utils/hash-state'
  import type { SettingsSection } from '../types'
  import { settingsSections } from '../types'
  import AgentSettings from './agent-settings.svelte'
  import AccessSettings from './access-settings.svelte'
  import ArchivedTicketsSettings from './archived-tickets-settings.svelte'
  import SettingsNav from './settings-nav.svelte'
  import GeneralSettings from './general-settings.svelte'
  import NotificationSettings from './notification-settings.svelte'
  import RepositoriesSettings from './repositories-settings.svelte'
  import SecuritySettings from './security-settings.svelte'
  import StatusSettings from './status-settings.svelte'

  let activeSection = $state<SettingsSection>('general')
  let hashSyncReady = $state(false)

  function handleSelect(section: SettingsSection) {
    activeSection = section
  }

  function syncSectionFromHash() {
    activeSection = currentHashSelection(settingsSections, 'general')
  }

  onMount(() => {
    syncSectionFromHash()
    hashSyncReady = true

    const handleHashChange = () => {
      syncSectionFromHash()
    }

    window.addEventListener('hashchange', handleHashChange)

    return () => {
      window.removeEventListener('hashchange', handleHashChange)
    }
  })

  $effect(() => {
    if (!hashSyncReady) {
      return
    }

    writeHashSelection(activeSection)
  })
</script>

<PageScaffold title={i18nStore.t('settings.page.title')} helpSection="settings">
  <div class="flex flex-col gap-6 lg:flex-row lg:gap-8">
    <SettingsNav active={activeSection} onSelect={handleSelect} />

    <div class="min-w-0 flex-1" data-tour="settings-content-panel">
      {#if activeSection === 'general'}
        <GeneralSettings />
      {:else if activeSection === 'statuses'}
        <StatusSettings />
      {:else if activeSection === 'repositories'}
        <RepositoriesSettings />
      {:else if activeSection === 'agents'}
        <AgentSettings />
      {:else if activeSection === 'notifications'}
        <NotificationSettings />
      {:else if activeSection === 'access'}
        <AccessSettings />
      {:else if activeSection === 'security'}
        <SecuritySettings />
      {:else if activeSection === 'archived'}
        <ArchivedTicketsSettings />
      {/if}
    </div>
  </div>
</PageScaffold>
