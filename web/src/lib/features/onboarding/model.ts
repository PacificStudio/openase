import type {
  BootstrapPresetKey,
  OnboardingData,
  OnboardingStep,
  OnboardingStepId,
  ProjectBootstrapPreset,
} from './types'

const stepIds: OnboardingStepId[] = [
  'github_token',
  'repo',
  'provider',
  'agent_workflow',
  'first_ticket',
  'ai_discovery',
]

type StepCopyEntry = {
  fallback: string | string[]
  translationKey?: string | string[]
}

const stepCopy: Record<string, StepCopyEntry> = {
  'github_token.label': {
    translationKey: 'onboarding.step.githubToken.label',
    fallback: 'Connect GitHub',
  },
  'github_token.description': {
    translationKey: 'onboarding.step.githubToken.description',
    fallback: 'Configure a GitHub token for repository access',
  },
  'github_token.purpose': {
    translationKey: 'onboarding.step.githubToken.purpose',
    fallback:
      'This lets OpenASE create repositories, inspect namespaces, and verify which GitHub identity the project will use for future automation.',
  },
  'github_token.configHighlights': {
    translationKey: [
      'onboarding.step.githubToken.configHighlights.projectScope',
      'onboarding.step.githubToken.configHighlights.verification',
    ],
    fallback: [
      'The token is stored at project scope so later GitHub actions reuse the same identity.',
      'Verification checks both token health and the exact GitHub account OpenASE will act as.',
    ],
  },
  'github_token.skipRisks': {
    translationKey: [
      'onboarding.step.githubToken.skipRisks.creationBlocked',
      'onboarding.step.githubToken.skipRisks.unverifiedIdentity',
      'onboarding.step.githubToken.skipRisks.wrongAccount',
    ],
    fallback: [
      'Repository creation and repository search in onboarding will be blocked.',
      'Later steps may look configured, but agents still will not have a working code host identity.',
      'Using the wrong token can attach the project to the wrong GitHub account.',
    ],
  },
  'repo.label': {
    translationKey: 'onboarding.step.repo.label',
    fallback: 'Create or link a repository',
  },
  'repo.description': {
    translationKey: 'onboarding.step.repo.description',
    fallback: 'Add at least one Git repository to the project',
  },
  'repo.purpose': {
    translationKey: 'onboarding.step.repo.purpose',
    fallback:
      'This defines the codebase an agent will read from, write to, and scope tickets against during execution.',
  },
  'repo.configHighlights': {
    translationKey: [
      'onboarding.step.repo.configHighlights.chooseCreateOrLink',
      'onboarding.step.repo.configHighlights.repoUrlBranch',
    ],
    fallback: [
      'You can either create a fresh GitHub repository or link an existing codebase.',
      'The repository URL and default branch decide where new work starts from.',
    ],
  },
  'repo.skipRisks': {
    translationKey: [
      'onboarding.step.repo.skipRisks.noCloneSource',
      'onboarding.step.repo.skipRisks.noScope',
      'onboarding.step.repo.skipRisks.extraSetup',
    ],
    fallback: [
      'Agents have nowhere to clone from or open pull requests against.',
      'Tickets may be created without a repository scope, which weakens execution context.',
      'You will need extra manual setup before any coding workflow can start.',
    ],
  },
  'provider.label': {
    translationKey: 'onboarding.step.provider.label',
    fallback: 'Select and configure an AI provider',
  },
  'provider.description': {
    translationKey: 'onboarding.step.provider.description',
    fallback: 'Configure at least one available AI execution engine',
  },
  'provider.purpose': {
    translationKey: 'onboarding.step.provider.purpose',
    fallback:
      'This chooses the execution engine that actually runs agents, such as Codex, Claude Code, or Gemini CLI.',
  },
  'provider.configHighlights': {
    translationKey: [
      'onboarding.step.provider.configHighlights.availabilityChecks',
      'onboarding.step.provider.configHighlights.defaultProvider',
    ],
    fallback: [
      'Availability checks confirm that the machine, model, and credentials are genuinely ready.',
      'The selected default provider is what onboarding uses when it creates the first agent.',
    ],
  },
  'provider.skipRisks': {
    translationKey: [
      'onboarding.step.provider.skipRisks.noProvider',
      'onboarding.step.provider.skipRisks.bootstrapFail',
      'onboarding.step.provider.skipRisks.unclearTradeoffs',
    ],
    fallback: [
      'OpenASE cannot launch agents if no healthy provider is available.',
      'Bootstrap may fail later because the default provider is missing or offline.',
      'Provider cost, capability, and auth tradeoffs remain unclear until you configure one.',
    ],
  },
  'agent_workflow.label': {
    translationKey: 'onboarding.step.agentWorkflow.label',
    fallback: 'Create an agent and workflow',
  },
  'agent_workflow.description': {
    translationKey: 'onboarding.step.agentWorkflow.description',
    fallback: 'Automatically create the first working agent and workflow',
  },
  'agent_workflow.purpose': {
    translationKey: 'onboarding.step.agentWorkflow.purpose',
    fallback:
      'This wires a role, a provider, and ticket pickup rules together so the project can execute work automatically.',
  },
  'agent_workflow.configHighlights': {
    translationKey: [
      'onboarding.step.agentWorkflow.configHighlights.roleBinding',
      'onboarding.step.agentWorkflow.configHighlights.workflowRules',
    ],
    fallback: [
      'OpenASE picks a starter role from the project status and binds it to the default provider.',
      'The workflow decides which statuses the agent can pick up and what counts as finished.',
    ],
  },
  'agent_workflow.skipRisks': {
    translationKey: [
      'onboarding.step.agentWorkflow.skipRisks.ticketsPileUp',
      'onboarding.step.agentWorkflow.skipRisks.noOrchestration',
      'onboarding.step.agentWorkflow.skipRisks.manualRisk',
    ],
    fallback: [
      'Tickets can pile up with no agent configured to claim them.',
      'Even with repos and providers ready, orchestration will not start automatically.',
      'Manual workflow setup later is possible, but easier to misconfigure.',
    ],
  },
  'first_ticket.label': {
    translationKey: 'onboarding.step.firstTicket.label',
    fallback: 'Create the first ticket',
  },
  'first_ticket.description': {
    translationKey: 'onboarding.step.firstTicket.description',
    fallback: 'Submit the first task so an agent can start working',
  },
  'first_ticket.purpose': {
    translationKey: 'onboarding.step.firstTicket.purpose',
    fallback:
      'This gives the orchestrator a real unit of work so you can verify the full setup end to end.',
  },
  'first_ticket.configHighlights': {
    translationKey: [
      'onboarding.step.firstTicket.configHighlights.initialStatus',
      'onboarding.step.firstTicket.configHighlights.repoScope',
    ],
    fallback: [
      'The initial status determines whether the workflow can auto-pick the ticket.',
      'An optional repository scope narrows the task to the most relevant codebase.',
    ],
  },
  'first_ticket.skipRisks': {
    translationKey: [
      'onboarding.step.firstTicket.skipRisks.noWork',
      'onboarding.step.firstTicket.skipRisks.wrongStatus',
      'onboarding.step.firstTicket.skipRisks.noConfirmation',
    ],
    fallback: [
      'Setup may look complete, but nothing will execute because there is no queued work.',
      'A later ticket created with the wrong status may never be picked up.',
      'You lose the fastest way to confirm the automation pipeline is actually working.',
    ],
  },
  'ai_discovery.label': {
    translationKey: 'onboarding.step.aiDiscovery.label',
    fallback: 'Try Project AI',
  },
  'ai_discovery.description': {
    translationKey: 'onboarding.step.aiDiscovery.description',
    fallback: 'Use Project AI to refine the next steps for the project',
  },
  'ai_discovery.purpose': {
    translationKey: 'onboarding.step.aiDiscovery.purpose',
    fallback:
      'This shows how Project AI can turn the current project context into follow-up tasks and planning help.',
  },
  'ai_discovery.configHighlights': {
    translationKey: [
      'onboarding.step.aiDiscovery.configHighlights.contextualSuggestions',
      'onboarding.step.aiDiscovery.configHighlights.handOff',
    ],
    fallback: [
      'It uses the project, repositories, and tickets you just configured as context for suggestions.',
      'This is the fastest handoff from initial setup into ongoing planning and backlog expansion.',
    ],
  },
  'ai_discovery.skipRisks': {
    translationKey: [
      'onboarding.step.aiDiscovery.skipRisks.missedTransition',
      'onboarding.step.aiDiscovery.skipRisks.manualTriaging',
      'onboarding.step.aiDiscovery.skipRisks.lowerRisk',
    ],
    fallback: [
      'You miss the guided transition from setup into the next wave of project work.',
      'New users may underuse Project AI and do more manual triage than necessary.',
      'This step is lower risk than the execution steps above, but it still improves adoption.',
    ],
  },
}

function stepText(stepId: OnboardingStepId, key: string) {
  const entry = stepCopy[`${stepId}.${key}`]
  if (!entry || typeof entry.fallback !== 'string') {
    return ''
  }
  return entry.fallback
}

function stepList(stepId: OnboardingStepId, key: string) {
  const entry = stepCopy[`${stepId}.${key}`]
  if (!entry || !Array.isArray(entry.fallback)) {
    return []
  }
  return entry.fallback
}

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

type PresetCopyEntry = {
  fallback: string
  translationKey?: string
}

const presetCopy: Record<string, PresetCopyEntry> = {
  'fullstack.title': {
    translationKey: 'onboarding.preset.fullstack.title',
    fallback: 'Write code',
  },
  'fullstack.subtitle': {
    translationKey: 'onboarding.preset.fullstack.subtitle',
    fallback: 'I have a codebase and want to ship features.',
  },
  'fullstack.roleName': {
    translationKey: 'onboarding.preset.fullstack.roleName',
    fallback: 'Fullstack Developer',
  },
  'fullstack.roleSlug': {
    translationKey: 'onboarding.preset.fullstack.roleSlug',
    fallback: 'fullstack-developer',
  },
  'fullstack.workflowType': {
    translationKey: 'onboarding.preset.fullstack.workflowType',
    fallback: 'Fullstack Developer',
  },
  'fullstack.pickupStatusName': {
    translationKey: 'onboarding.preset.fullstack.pickupStatusName',
    fallback: 'Todo',
  },
  'fullstack.finishStatusName': {
    translationKey: 'onboarding.preset.fullstack.finishStatusName',
    fallback: 'Done',
  },
  'fullstack.agentNameSuggestion': {
    translationKey: 'onboarding.preset.fullstack.agentNameSuggestion',
    fallback: 'fullstack-dev-01',
  },
  'fullstack.exampleTicketTitle': {
    translationKey: 'onboarding.preset.fullstack.exampleTicketTitle',
    fallback: 'Implement user authentication',
  },
  'fullstack.exampleTicketDescription': {
    translationKey: 'onboarding.preset.fullstack.exampleTicketDescription',
    fallback: 'Add login/logout and protect routes that require a signed-in user.',
  },
  'pm.title': {
    translationKey: 'onboarding.preset.pm.title',
    fallback: 'Plan the project',
  },
  'pm.subtitle': {
    translationKey: 'onboarding.preset.pm.subtitle',
    fallback: 'I need to figure out what to build first.',
  },
  'pm.roleName': {
    translationKey: 'onboarding.preset.pm.roleName',
    fallback: 'Product Manager',
  },
  'pm.roleSlug': {
    translationKey: 'onboarding.preset.pm.roleSlug',
    fallback: 'product-manager',
  },
  'pm.workflowType': {
    translationKey: 'onboarding.preset.pm.workflowType',
    fallback: 'Product Manager',
  },
  'pm.pickupStatusName': {
    translationKey: 'onboarding.preset.pm.pickupStatusName',
    fallback: 'Todo',
  },
  'pm.finishStatusName': {
    translationKey: 'onboarding.preset.pm.finishStatusName',
    fallback: 'Done',
  },
  'pm.agentNameSuggestion': {
    translationKey: 'onboarding.preset.pm.agentNameSuggestion',
    fallback: 'product-manager-01',
  },
  'pm.exampleTicketTitle': {
    translationKey: 'onboarding.preset.pm.exampleTicketTitle',
    fallback: 'Draft the initial product requirements',
  },
  'pm.exampleTicketDescription': {
    translationKey: 'onboarding.preset.pm.exampleTicketDescription',
    fallback: 'Outline scope, goals, and the first set of acceptance criteria.',
  },
  'researcher.title': {
    translationKey: 'onboarding.preset.researcher.title',
    fallback: 'Explore ideas',
  },
  'researcher.subtitle': {
    translationKey: 'onboarding.preset.researcher.subtitle',
    fallback: "I'm not sure where to start yet.",
  },
  'researcher.roleName': {
    translationKey: 'onboarding.preset.researcher.roleName',
    fallback: 'Research Ideation',
  },
  'researcher.roleSlug': {
    translationKey: 'onboarding.preset.researcher.roleSlug',
    fallback: 'research-ideation',
  },
  'researcher.workflowType': {
    translationKey: 'onboarding.preset.researcher.workflowType',
    fallback: 'Research Ideation',
  },
  'researcher.pickupStatusName': {
    translationKey: 'onboarding.preset.researcher.pickupStatusName',
    fallback: 'Todo',
  },
  'researcher.finishStatusName': {
    translationKey: 'onboarding.preset.researcher.finishStatusName',
    fallback: 'Done',
  },
  'researcher.agentNameSuggestion': {
    translationKey: 'onboarding.preset.researcher.agentNameSuggestion',
    fallback: 'researcher-01',
  },
  'researcher.exampleTicketTitle': {
    translationKey: 'onboarding.preset.researcher.exampleTicketTitle',
    fallback: 'Explore options and recommend a direction',
  },
  'researcher.exampleTicketDescription': {
    translationKey: 'onboarding.preset.researcher.exampleTicketDescription',
    fallback: 'Compare two or three approaches and pick the most viable one.',
  },
}

function presetText(key: string, field: string) {
  const entry = presetCopy[`${key}.${field}`]
  return entry?.fallback ?? ''
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
