export type RunSummarySectionKey =
  | 'overview'
  | 'major_steps'
  | 'long_running_operations'
  | 'repeated_trial_and_error'
  | 'security_safety_risks'
  | 'files_touched'
  | 'outcome'
  | 'commands_and_tooling'
  | 'approvals_interruptions'
  | 'run_metadata'

import type { TranslationKey } from '$lib/i18n'
import { i18nStore } from '$lib/i18n/store.svelte'

export type RunSummarySectionDefinition = {
  key: RunSummarySectionKey
  title: string
  heading: string
  instruction: string
  description: string
}

function translateRaw(key: TranslationKey) {
  return i18nStore.t(key)
}

type RunSummarySectionTemplate = {
  key: RunSummarySectionKey
  titleKey: TranslationKey
  instructionKey: TranslationKey
  descriptionKey: TranslationKey
}

const runSummarySectionTemplates: RunSummarySectionTemplate[] = [
  {
    key: 'overview',
    titleKey: 'settings.runSummary.section.overview.title',
    instructionKey: 'settings.runSummary.section.overview.instruction',
    descriptionKey: 'settings.runSummary.section.overview.description',
  },
  {
    key: 'major_steps',
    titleKey: 'settings.runSummary.section.majorSteps.title',
    instructionKey: 'settings.runSummary.section.majorSteps.instruction',
    descriptionKey: 'settings.runSummary.section.majorSteps.description',
  },
  {
    key: 'long_running_operations',
    titleKey: 'settings.runSummary.section.longRunningOperations.title',
    instructionKey: 'settings.runSummary.section.longRunningOperations.instruction',
    descriptionKey: 'settings.runSummary.section.longRunningOperations.description',
  },
  {
    key: 'repeated_trial_and_error',
    titleKey: 'settings.runSummary.section.repeatedTrialAndError.title',
    instructionKey: 'settings.runSummary.section.repeatedTrialAndError.instruction',
    descriptionKey: 'settings.runSummary.section.repeatedTrialAndError.description',
  },
  {
    key: 'security_safety_risks',
    titleKey: 'settings.runSummary.section.securityAndSafetyRisks.title',
    instructionKey: 'settings.runSummary.section.securityAndSafetyRisks.instruction',
    descriptionKey: 'settings.runSummary.section.securityAndSafetyRisks.description',
  },
  {
    key: 'files_touched',
    titleKey: 'settings.runSummary.section.filesTouched.title',
    instructionKey: 'settings.runSummary.section.filesTouched.instruction',
    descriptionKey: 'settings.runSummary.section.filesTouched.description',
  },
  {
    key: 'outcome',
    titleKey: 'settings.runSummary.section.outcome.title',
    instructionKey: 'settings.runSummary.section.outcome.instruction',
    descriptionKey: 'settings.runSummary.section.outcome.description',
  },
  {
    key: 'commands_and_tooling',
    titleKey: 'settings.runSummary.section.commandsAndTooling.title',
    instructionKey: 'settings.runSummary.section.commandsAndTooling.instruction',
    descriptionKey: 'settings.runSummary.section.commandsAndTooling.description',
  },
  {
    key: 'approvals_interruptions',
    titleKey: 'settings.runSummary.section.approvalsInterruptions.title',
    instructionKey: 'settings.runSummary.section.approvalsInterruptions.instruction',
    descriptionKey: 'settings.runSummary.section.approvalsInterruptions.description',
  },
  {
    key: 'run_metadata',
    titleKey: 'settings.runSummary.section.runMetadata.title',
    instructionKey: 'settings.runSummary.section.runMetadata.instruction',
    descriptionKey: 'settings.runSummary.section.runMetadata.description',
  },
]

export const runSummarySectionDefinitions: RunSummarySectionDefinition[] = runSummarySectionTemplates.map(
  (template) => ({
    key: template.key,
    get title() {
      return translateRaw(template.titleKey)
    },
    get heading() {
      return `## ${translateRaw(template.titleKey)}`
    },
    get instruction() {
      return translateRaw(template.instructionKey)
    },
    get description() {
      return translateRaw(template.descriptionKey)
    },
  }),
)

export const defaultRunSummarySectionKeys: RunSummarySectionKey[] = [
  'major_steps',
  'long_running_operations',
  'repeated_trial_and_error',
  'security_safety_risks',
  'outcome',
]

export function buildRunSummaryPrompt(
  selectedKeys: RunSummarySectionKey[],
  customInstructions: string,
): string {
  const sections = selectedKeys
    .map((key) => runSummarySectionDefinitions.find((definition) => definition.key === key))
    .filter((definition): definition is RunSummarySectionDefinition => Boolean(definition))

  const lines = [
    translateRaw('settings.runSummary.prompt.overallInstruction'),
    translateRaw('settings.runSummary.prompt.conciseMarkdown'),
    translateRaw('settings.runSummary.prompt.includeSections'),
    ...sections.flatMap((section) => [section.heading, section.instruction]),
    translateRaw('settings.runSummary.prompt.emptySectionNotice'),
  ]

  const trimmedCustomInstructions = customInstructions.trim()
  if (trimmedCustomInstructions !== '') {
    lines.push('', translateRaw('settings.runSummary.prompt.additionalInstructions'), trimmedCustomInstructions)
  }

  return lines.join('\n')
}
