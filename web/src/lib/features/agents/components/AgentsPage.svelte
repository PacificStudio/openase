<script lang="ts">
  import MasterDetailLayout from '$lib/components/layout/MasterDetailLayout.svelte'
  import { ActivityFeedPanel, RunningNowPanel } from '$lib/features/dashboard'
  import type { createAgentsController } from '../controller.svelte'
  import AgentDetailPanel from './AgentDetailPanel.svelte'
  import ProviderConfigPanel from './ProviderConfigPanel.svelte'

  let {
    controller,
  }: {
    controller: ReturnType<typeof createAgentsController>
  } = $props()
</script>

<svelte:head>
  <title>Agents · OpenASE</title>
</svelte:head>

<MasterDetailLayout detailWidthClass="xl:grid-cols-[22rem_minmax(0,1fr)]">
  {#snippet main()}
    <RunningNowPanel
      agents={controller.workspace.dashboard.agents}
      busy={controller.workspace.dashboard.busy}
      error={controller.workspace.dashboard.error}
      selectedAgentId={controller.workspace.dashboard.selectedAgentId}
      heartbeatNow={controller.workspace.dashboard.heartbeatNow}
      agentStreamState={controller.workspace.dashboard.agentStreamState}
      onSelectAgent={(agentId) => void controller.workspace.dashboard.selectAgent(agentId)}
    />

    <ProviderConfigPanel
      providers={controller.providers}
      provider={controller.providerForSelectedAgent()}
      busy={controller.providerBusy}
      error={controller.providerError}
      countAgentsForProvider={controller.agentsForProvider}
    />
  {/snippet}

  {#snippet detail()}
    <AgentDetailPanel
      agent={controller.selectedAgent()}
      project={controller.workspace.state.selectedProject}
      heartbeatNow={controller.workspace.dashboard.heartbeatNow}
    />

    <ActivityFeedPanel
      activityEvents={controller.workspace.dashboard.activityEvents}
      activityStreamState={controller.workspace.dashboard.activityStreamState}
      selectedAgentName={controller.workspace.dashboard.selectedAgentName()}
    />
  {/snippet}
</MasterDetailLayout>
