<script lang="ts">
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'

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

  let skipConfirmOpen = $state(false)

  function handleSkipClick() {
    skipConfirmOpen = true
  }

  function handleConfirmSkip() {
    skipConfirmOpen = false
    onOnboardingComplete()
  }
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
    onComplete={(agents, workflows, presetKey) => {
      onDataChange({
        ...data,
        agentWorkflow: { ...data.agentWorkflow, agents, workflows, selectedPresetKey: presetKey },
      })
    }}
  />
{:else if step.id === 'first_ticket'}
  <StepFirstTicket
    {projectId}
    {orgId}
    {projectStatus}
    selectedPresetKey={data.agentWorkflow.selectedPresetKey}
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
  <div class="mt-4 flex justify-end border-t pt-4">
    <Button variant="ghost" size="sm" class="text-xs" onclick={handleSkipClick}>Skip tour</Button>
  </div>
{/if}

<Dialog.Root bind:open={skipConfirmOpen}>
  <Dialog.Content class="max-w-sm">
    <Dialog.Header>
      <Dialog.Title class="text-sm">Skip onboarding?</Dialog.Title>
      <Dialog.Description class="text-muted-foreground text-xs">
        The guided tour will be dismissed. You can configure the remaining steps later from the
        project settings.
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer class="gap-2">
      <Button variant="ghost" size="sm" class="text-xs" onclick={() => (skipConfirmOpen = false)}>
        Continue setup
      </Button>
      <Button variant="destructive" size="sm" class="text-xs" onclick={handleConfirmSkip}>
        Skip anyway
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
