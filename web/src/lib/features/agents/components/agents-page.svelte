<script lang="ts">
  import { connectEventStream } from '$lib/api/sse'
  import { appStore } from '$lib/stores/app.svelte'
  import { createAgentsPageState } from '../page-state.svelte'
  import AgentsPageDrawers from './agents-page-drawers.svelte'
  import AgentsPagePanel from './agents-page-panel.svelte'

  const state = createAgentsPageState()

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    const defaultProviderId = appStore.currentOrg?.default_agent_provider_id
    if (!projectId || !orgId) {
      state.reset(defaultProviderId)
      return
    }

    void state.loadData({ projectId, orgId, defaultProviderId, showLoading: true })

    return connectEventStream(`/api/v1/projects/${projectId}/agents/stream`, {
      onEvent: () => {
        void state.loadData({ projectId, orgId, defaultProviderId, showLoading: false })
      },
      onError: (streamError) => {
        console.error('Agents stream error:', streamError)
      },
    })
  })

  $effect(() => {
    state.setProviderConfigOpen(state.view.providerConfigOpen)
  })

  $effect(() => {
    state.setOutputSheetOpen(state.view.outputSheetOpen)
  })
</script>

<div class="space-y-4">
  <AgentsPagePanel
    bind:activeTab={state.view.activeTab}
    agents={state.view.agents}
    providers={state.view.providers}
    loading={state.view.loading}
    error={state.view.error}
    pageFeedback={state.view.pageFeedback}
    canRegister={!!appStore.currentProject?.id && state.view.providerItems.length > 0}
    registerButtonTitle={state.view.providerItems.length === 0
      ? 'Register a provider before creating agents.'
      : appStore.currentProject?.id
        ? undefined
        : 'Project context is unavailable.'}
    onOpenRegister={() =>
      state.handleRegisterOpenChange(true, appStore.currentOrg?.default_agent_provider_id)}
    onSelectTicket={(ticketId) => {
      appStore.openRightPanel({ type: 'ticket', id: ticketId })
    }}
    onOpenOutput={(agent) => state.handleOpenOutput(agent)}
    onConfigureProvider={(provider) => state.handleConfigureProvider(provider)}
  />
</div>

<AgentsPageDrawers
  bind:registerSheetOpen={state.view.registerSheetOpen}
  bind:providerConfigOpen={state.view.providerConfigOpen}
  bind:outputSheetOpen={state.view.outputSheetOpen}
  providerItems={state.view.providerItems}
  registrationDraft={state.view.registrationDraft}
  registerSaving={state.view.registerSaving}
  registerError={state.view.registerError}
  registerFeedback={state.view.registerFeedback}
  onRegistrationDraftChange={(field, value) => state.updateRegistrationDraft(field, value)}
  onRegisterAgent={() =>
    state.handleRegisterAgent({
      projectId: appStore.currentProject?.id,
      orgId: appStore.currentOrg?.id,
      defaultProviderId: appStore.currentOrg?.default_agent_provider_id,
    })}
  onRegisterOpenChange={(open) =>
    state.handleRegisterOpenChange(open, appStore.currentOrg?.default_agent_provider_id)}
  selectedProvider={state.selectedProvider}
  providerDraft={state.view.providerDraft}
  providerSaving={state.view.providerSaving}
  providerFeedback={state.view.providerFeedback}
  providerError={state.view.providerError}
  selectedOutputAgent={state.selectedOutputAgent}
  outputEntries={state.view.outputEntries}
  outputLoading={state.view.outputLoading}
  outputError={state.view.outputError}
  onProviderDraftChange={(field, value) => state.handleProviderDraftChange(field, value)}
  onProviderSave={() => state.handleProviderSave()}
  onRefreshOutput={() => state.refreshOutput()}
/>
