import type {
  OnboardingData,
  OnboardingStep,
  OnboardingStepId,
  ProjectBootstrapPreset,
} from './types'

const stepDefinitions: { id: OnboardingStepId; label: string; description: string }[] = [
  {
    id: 'github_token',
    label: 'Connect GitHub',
    description: 'Configure a GitHub token for repository access',
  },
  {
    id: 'repo',
    label: 'Create or link a repository',
    description: 'Add at least one Git repository to the project',
  },
  {
    id: 'provider',
    label: 'Select and configure an AI provider',
    description: 'Configure at least one available AI execution engine',
  },
  {
    id: 'agent_workflow',
    label: 'Create an agent and workflow',
    description: 'Automatically create the first working agent and workflow',
  },
  {
    id: 'first_ticket',
    label: 'Create the first ticket',
    description: 'Submit the first task so an agent can start working',
  },
  {
    id: 'ai_discovery',
    label: 'Try Project AI and Harness AI',
    description: 'Use AI assistants to further refine the project setup',
  },
]

export function isStepCompleted(step: OnboardingStepId, data: OnboardingData): boolean {
  switch (step) {
    case 'github_token':
      return data.github.hasToken && data.github.probeStatus === 'valid' && data.github.confirmed
    case 'repo':
      return data.repo.repos.length > 0
    case 'provider': {
      const selectedProvider = data.provider.providers.find(
        (provider) => provider.id === data.provider.selectedProviderId,
      )
      return Boolean(
        selectedProvider &&
        (selectedProvider.availability_state === 'available' ||
          selectedProvider.availability_state === 'ready'),
      )
    }
    case 'agent_workflow':
      return data.agentWorkflow.agents.length > 0 && data.agentWorkflow.workflows.length > 0
    case 'first_ticket':
      return data.firstTicket.ticketCount > 0
    case 'ai_discovery':
      return data.aiDiscovery.completed
  }
}

export function buildOnboardingSteps(data: OnboardingData): OnboardingStep[] {
  const steps: OnboardingStep[] = []
  let foundActive = false

  for (const def of stepDefinitions) {
    const completed = isStepCompleted(def.id, data)
    let status: OnboardingStep['status']

    if (completed) {
      status = 'completed'
    } else if (!foundActive) {
      status = 'active'
      foundActive = true
    } else {
      status = 'locked'
    }

    steps.push({ ...def, status })
  }

  return steps
}

export function currentActiveStep(data: OnboardingData): OnboardingStepId | null {
  for (const def of stepDefinitions) {
    if (!isStepCompleted(def.id, data)) {
      return def.id
    }
  }
  return null
}

export function isOnboardingComplete(data: OnboardingData): boolean {
  return currentActiveStep(data) === null
}

export function getBootstrapPreset(projectStatus: string): ProjectBootstrapPreset {
  switch (projectStatus) {
    case 'In Progress':
      return {
        roleName: 'Fullstack Developer',
        roleSlug: 'fullstack-developer',
        workflowType: 'Fullstack Developer',
        pickupStatusName: 'Backlog',
        finishStatusName: 'Done',
        agentNameSuggestion: 'fullstack-developer-01',
        exampleTicketTitle: 'Implement the first core feature of the project',
      }
    case 'Planned':
    case 'Backlog':
    default:
      return {
        roleName: 'Product Manager',
        roleSlug: 'product-manager',
        workflowType: 'Product Manager',
        pickupStatusName: 'Backlog',
        finishStatusName: 'Done',
        agentNameSuggestion: 'product-manager-01',
        exampleTicketTitle: 'Review project requirements and draft the first PRD',
      }
  }
}

export function isTerminalProjectStatus(status: string): boolean {
  return ['Completed', 'Canceled', 'Archived'].includes(status)
}
