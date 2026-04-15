import { translate, type AppLocale, type TranslationKey } from '$lib/i18n'
import type { OnboardingStepId } from './types'

export type StepCopyEntry = {
  fallback: string | string[]
  translationKey?: TranslationKey | TranslationKey[]
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

export function stepText(locale: AppLocale, stepId: OnboardingStepId, key: string) {
  const entry = stepCopy[`${stepId}.${key}`]
  if (!entry || typeof entry.fallback !== 'string') {
    return ''
  }
  if (typeof entry.translationKey === 'string') {
    return translate(locale, entry.translationKey)
  }
  return entry.fallback
}

export function stepList(locale: AppLocale, stepId: OnboardingStepId, key: string) {
  const entry = stepCopy[`${stepId}.${key}`]
  if (!entry || !Array.isArray(entry.fallback)) {
    return []
  }
  if (Array.isArray(entry.translationKey)) {
    return entry.translationKey.map((translationKey) => translate(locale, translationKey))
  }
  return entry.fallback
}
