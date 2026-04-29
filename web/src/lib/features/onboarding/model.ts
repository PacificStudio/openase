import type {
  BootstrapPresetKey,
  OnboardingData,
  OnboardingStep,
  OnboardingStepId,
  ProjectBootstrapPreset,
} from './types'
import type { AppLocale } from '$lib/i18n'
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

function buildStepDefinitions(locale: AppLocale): Omit<OnboardingStep, 'status'>[] {
  return stepIds.map((id) => ({
    id,
    label: stepText(locale, id, 'label'),
    description: stepText(locale, id, 'description'),
    purpose: stepText(locale, id, 'purpose'),
    configHighlights: stepList(locale, id, 'configHighlights'),
    skipRisks: stepList(locale, id, 'skipRisks'),
  }))
}

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

export function buildOnboardingSteps(
  data: OnboardingData,
  locale: AppLocale = 'en',
): OnboardingStep[] {
  const steps: OnboardingStep[] = []
  let foundActive = false

  for (const def of buildStepDefinitions(locale)) {
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
  for (const def of stepIds) {
    if (!isStepCompleted(def, data)) {
      return def
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
