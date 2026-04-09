import type {
  BootstrapPresetKey,
  OnboardingData,
  OnboardingStep,
  OnboardingStepId,
  ProjectBootstrapPreset,
} from './types'

const stepDefinitions: Omit<OnboardingStep, 'status'>[] = [
  {
    id: 'github_token',
    label: 'Connect GitHub',
    description: 'Configure a GitHub token for repository access',
    purpose:
      'This lets OpenASE create repositories, inspect namespaces, and verify which GitHub identity the project will use for future automation.',
    configHighlights: [
      'The token is stored at project scope so later GitHub actions reuse the same identity.',
      'Verification checks both token health and the exact GitHub account OpenASE will act as.',
    ],
    skipRisks: [
      'Repository creation and repository search in onboarding will be blocked.',
      'Later steps may look configured, but agents still will not have a working code host identity.',
      'Using the wrong token can attach the project to the wrong GitHub account.',
    ],
  },
  {
    id: 'repo',
    label: 'Create or link a repository',
    description: 'Add at least one Git repository to the project',
    purpose:
      'This defines the codebase an agent will read from, write to, and scope tickets against during execution.',
    configHighlights: [
      'You can either create a fresh GitHub repository or link an existing codebase.',
      'The repository URL and default branch decide where new work starts from.',
    ],
    skipRisks: [
      'Agents have nowhere to clone from or open pull requests against.',
      'Tickets may be created without a repository scope, which weakens execution context.',
      'You will need extra manual setup before any coding workflow can start.',
    ],
  },
  {
    id: 'provider',
    label: 'Select and configure an AI provider',
    description: 'Configure at least one available AI execution engine',
    purpose:
      'This chooses the execution engine that actually runs agents, such as Codex, Claude Code, or Gemini CLI.',
    configHighlights: [
      'Availability checks confirm that the machine, model, and credentials are genuinely ready.',
      'The selected default provider is what onboarding uses when it creates the first agent.',
    ],
    skipRisks: [
      'OpenASE cannot launch agents if no healthy provider is available.',
      'Bootstrap may fail later because the default provider is missing or offline.',
      'Provider cost, capability, and auth tradeoffs remain unclear until you configure one.',
    ],
  },
  {
    id: 'agent_workflow',
    label: 'Create an agent and workflow',
    description: 'Automatically create the first working agent and workflow',
    purpose:
      'This wires a role, a provider, and ticket pickup rules together so the project can execute work automatically.',
    configHighlights: [
      'OpenASE picks a starter role from the project status and binds it to the default provider.',
      'The workflow decides which statuses the agent can pick up and what counts as finished.',
    ],
    skipRisks: [
      'Tickets can pile up with no agent configured to claim them.',
      'Even with repos and providers ready, orchestration will not start automatically.',
      'Manual workflow setup later is possible, but easier to misconfigure.',
    ],
  },
  {
    id: 'first_ticket',
    label: 'Create the first ticket',
    description: 'Submit the first task so an agent can start working',
    purpose:
      'This gives the orchestrator a real unit of work so you can verify the full setup end to end.',
    configHighlights: [
      'The initial status determines whether the workflow can auto-pick the ticket.',
      'An optional repository scope narrows the task to the most relevant codebase.',
    ],
    skipRisks: [
      'Setup may look complete, but nothing will execute because there is no queued work.',
      'A later ticket created with the wrong status may never be picked up.',
      'You lose the fastest way to confirm the automation pipeline is actually working.',
    ],
  },
  {
    id: 'ai_discovery',
    label: 'Try Project AI',
    description: 'Use Project AI to refine the next steps for the project',
    purpose:
      'This shows how Project AI can turn the current project context into follow-up tasks and planning help.',
    configHighlights: [
      'It uses the project, repositories, and tickets you just configured as context for suggestions.',
      'This is the fastest handoff from initial setup into ongoing planning and backlog expansion.',
    ],
    skipRisks: [
      'You miss the guided transition from setup into the next wave of project work.',
      'New users may underuse Project AI and do more manual triage than necessary.',
      'This step is lower risk than the execution steps above, but it still improves adoption.',
    ],
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

export const bootstrapPresets: ProjectBootstrapPreset[] = [
  {
    key: 'fullstack',
    title: 'Write code',
    subtitle: 'I have a codebase and want to ship features.',
    roleName: 'Fullstack Developer',
    roleSlug: 'fullstack-developer',
    workflowType: 'Fullstack Developer',
    pickupStatusName: 'Backlog',
    finishStatusName: 'Done',
    agentNameSuggestion: 'fullstack-dev-01',
    exampleTicketTitle: 'Implement user authentication',
    exampleTicketDescription: 'Add login/logout and protect routes that require a signed-in user.',
  },
  {
    key: 'pm',
    title: 'Plan the project',
    subtitle: 'I need to figure out what to build first.',
    roleName: 'Product Manager',
    roleSlug: 'product-manager',
    workflowType: 'Product Manager',
    pickupStatusName: 'Backlog',
    finishStatusName: 'Done',
    agentNameSuggestion: 'product-manager-01',
    exampleTicketTitle: 'Draft the initial product requirements',
    exampleTicketDescription: 'Outline scope, goals, and the first set of acceptance criteria.',
  },
  {
    key: 'researcher',
    title: 'Explore ideas',
    subtitle: "I'm not sure where to start yet.",
    roleName: 'Research Ideation',
    roleSlug: 'research-ideation',
    workflowType: 'Research Ideation',
    pickupStatusName: 'Backlog',
    finishStatusName: 'Done',
    agentNameSuggestion: 'researcher-01',
    exampleTicketTitle: 'Explore options and recommend a direction',
    exampleTicketDescription: 'Compare two or three approaches and pick the most viable one.',
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
