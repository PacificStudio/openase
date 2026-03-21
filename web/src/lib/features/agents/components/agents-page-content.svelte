<script lang="ts">
  import type { ContentViewModel } from './agents-page-content-view-model'
  import AgentsPageDrawers from './agents-page-drawers.svelte'
  import AgentsPagePanel from './agents-page-panel.svelte'

  let {
    activeTab = $bindable('instances'),
    registerSheetOpen = $bindable(false),
    providerConfigOpen = $bindable(false),
    viewModel,
  }: {
    activeTab?: string
    registerSheetOpen?: boolean
    providerConfigOpen?: boolean
    viewModel: ContentViewModel
  } = $props()
</script>

<div class="space-y-4">
  <AgentsPagePanel
    bind:activeTab
    agents={viewModel.agents}
    providers={viewModel.providers}
    loading={viewModel.loading}
    error={viewModel.error}
    pageError={viewModel.pageError}
    pageFeedback={viewModel.pageFeedback}
    runtimeControlPendingAgentId={viewModel.runtimeControlPendingAgentId}
    canRegister={viewModel.canRegister}
    registerButtonTitle={viewModel.registerButtonTitle}
    onOpenRegister={viewModel.onOpenRegister}
    onSelectTicket={viewModel.onSelectTicket}
    onConfigureProvider={viewModel.onConfigureProvider}
    onPauseAgent={viewModel.onPauseAgent}
    onResumeAgent={viewModel.onResumeAgent}
  />
</div>

<AgentsPageDrawers
  bind:registerSheetOpen
  bind:providerConfigOpen
  providerItems={viewModel.providerItems}
  registrationDraft={viewModel.registrationDraft}
  registerSaving={viewModel.registerSaving}
  registerError={viewModel.registerError}
  registerFeedback={viewModel.registerFeedback}
  onRegistrationDraftChange={viewModel.onRegistrationDraftChange}
  onRegisterAgent={viewModel.onRegisterAgent}
  onRegisterOpenChange={viewModel.onRegisterOpenChange}
  selectedProvider={viewModel.selectedProvider}
  providerDraft={viewModel.providerDraft}
  providerSaving={viewModel.providerSaving}
  providerFeedback={viewModel.providerFeedback}
  providerError={viewModel.providerError}
  onProviderDraftChange={viewModel.onProviderDraftChange}
  onProviderSave={viewModel.onProviderSave}
/>
