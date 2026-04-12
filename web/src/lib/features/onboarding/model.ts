import type {
  BootstrapPresetKey,
  OnboardingData,
  OnboardingStep,
  OnboardingStepId,
  ProjectBootstrapPreset,
} from './types'
import { presetText } from './preset-copy'
import { stepList, stepText } from './step-copy'

const stepIds: OnboardingStepId[] = [
  'github_token',
  'repo',
  'provider',
  'agent_workflow',
  'first_ticket',
  'ai_discovery',
]

const stepDefinitions: Omit<OnboardingStep, 'status'>[] = stepIds.map((id) => ({
  id,
  label: stepText(id, 'label'),
  description: stepText(id, 'description'),
  purpose: stepText(id, 'purpose'),
  configHighlights: stepList(id, 'configHighlights'),
  skipRisks: stepList(id, 'skipRisks'),
}))

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

export const bootstrapPresets: ProjectBootstrapPreset[] = [
  {
    key: 'fullstack',
    title: presetText('fullstack', 'title'),
    subtitle: presetText('fullstack', 'subtitle'),
    roleName: presetText('fullstack', 'roleName'),
    roleSlug: presetText('fullstack', 'roleSlug'),
    workflowType: presetText('fullstack', 'workflowType'),
    pickupStatusName: presetText('fullstack', 'pickupStatusName'),
    finishStatusName: presetText('fullstack', 'finishStatusName'),
    agentNameSuggestion: presetText('fullstack', 'agentNameSuggestion'),
    exampleTicketTitle: presetText('fullstack', 'exampleTicketTitle'),
    exampleTicketDescription: presetText('fullstack', 'exampleTicketDescription'),
  },
  {
    key: 'pm',
    title: presetText('pm', 'title'),
    subtitle: presetText('pm', 'subtitle'),
    roleName: presetText('pm', 'roleName'),
    roleSlug: presetText('pm', 'roleSlug'),
    workflowType: presetText('pm', 'workflowType'),
    pickupStatusName: presetText('pm', 'pickupStatusName'),
    finishStatusName: presetText('pm', 'finishStatusName'),
    agentNameSuggestion: presetText('pm', 'agentNameSuggestion'),
    exampleTicketTitle: presetText('pm', 'exampleTicketTitle'),
    exampleTicketDescription: presetText('pm', 'exampleTicketDescription'),
  },
  {
    key: 'researcher',
    title: presetText('researcher', 'title'),
    subtitle: presetText('researcher', 'subtitle'),
    roleName: presetText('researcher', 'roleName'),
    roleSlug: presetText('researcher', 'roleSlug'),
    workflowType: presetText('researcher', 'workflowType'),
    pickupStatusName: presetText('researcher', 'pickupStatusName'),
    finishStatusName: presetText('researcher', 'finishStatusName'),
    agentNameSuggestion: presetText('researcher', 'agentNameSuggestion'),
    exampleTicketTitle: presetText('researcher', 'exampleTicketTitle'),
    exampleTicketDescription: presetText('researcher', 'exampleTicketDescription'),
  },
]

export function getBootstrapPreset(projectStatus: string): ProjectBootstrapPreset {
  const keyByStatus: Record<string, BootstrapPresetKey> = {
    'In Progress': 'fullstack',
  }
  const key: BootstrapPresetKey = keyByStatus[projectStatus] ?? 'pm'
  return bootstrapPresets.find((p) => p.key === key)!
}

export function isTerminalProjectStatus(status: string): boolean {
  return ['Completed', 'Canceled', 'Archived'].includes(status)
}
