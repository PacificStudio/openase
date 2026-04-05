<script lang="ts">
  import { Button } from '$ui/button'
  import type { OnboardingData, OnboardingStep } from '../types'
  import StepAiDiscovery from './step-ai-discovery.svelte'
  import StepAgentWorkflow from './step-agent-workflow.svelte'
  import StepFirstTicket from './step-first-ticket.svelte'
  import StepGithubToken from './step-github-token.svelte'
  import StepProvider from './step-provider.svelte'
  import StepRepo from './step-repo.svelte'

  let {
    step,
    data,
    projectId,
    orgId,
    projectStatus,
    selectedProviderId,
    onOpenProjectAI,
    onRefreshData,
    onDataChange,
    onOnboardingComplete,
  }: {
    step: OnboardingStep
    data: OnboardingData
    projectId: string
    orgId: string
    projectStatus: string
    selectedProviderId: string
    onOpenProjectAI: (prompt: string) => void
    onRefreshData: () => void
    onDataChange: (nextData: OnboardingData) => void
    onOnboardingComplete: () => void
  } = $props()
</script>

{#if step.id === 'github_token'}
  <StepGithubToken
    {projectId}
    initialState={data.github}
    onComplete={(updated) => {
      onDataChange({ ...data, github: updated })
      onRefreshData()
    }}
  />
{:else if step.id === 'repo'}
  <StepRepo
    {projectId}
    initialState={data.repo}
    onComplete={(repos) => {
      onDataChange({ ...data, repo: { ...data.repo, repos } })
    }}
  />
{:else if step.id === 'provider'}
  <StepProvider
    {projectId}
    {orgId}
    initialState={data.provider}
    onStateChange={(provider) => {
      onDataChange({ ...data, provider })
    }}
    onComplete={(providerId) => {
      onDataChange({
        ...data,
        provider: { ...data.provider, selectedProviderId: providerId },
      })
      onRefreshData()
    }}
  />
{:else if step.id === 'agent_workflow'}
  <StepAgentWorkflow
    {projectId}
    providerId={selectedProviderId}
    {projectStatus}
    initialState={data.agentWorkflow}
    onComplete={(agents, workflows) => {
      onDataChange({
        ...data,
        agentWorkflow: { ...data.agentWorkflow, agents, workflows },
      })
    }}
  />
{:else if step.id === 'first_ticket'}
  <StepFirstTicket
    {projectId}
    {orgId}
    {projectStatus}
    statuses={data.agentWorkflow.statuses}
    ticketCount={data.firstTicket.ticketCount}
    onComplete={() => {
      onDataChange({
        ...data,
        firstTicket: { ticketCount: data.firstTicket.ticketCount + 1 },
      })
    }}
  />
{:else if step.id === 'ai_discovery'}
  <StepAiDiscovery
    {orgId}
    {projectId}
    hasWorkflow={data.agentWorkflow.workflows.length > 0}
    {onOpenProjectAI}
    onComplete={() => {
      onDataChange({
        ...data,
        aiDiscovery: { completed: true },
      })
      onOnboardingComplete()
    }}
  />
{/if}

{#if step.status !== 'completed'}
  <div class="mt-4 flex items-center justify-between gap-3 border-t pt-4">
    <p class="text-muted-foreground text-xs">
      If you do not want to continue setup, you can skip the tour and finish now.
    </p>
    <Button variant="ghost" size="sm" class="text-xs" onclick={onOnboardingComplete}>
      Skip tour
    </Button>
  </div>
{/if}
