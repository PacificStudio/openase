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

export type RunSummarySectionDefinition = {
  key: RunSummarySectionKey
  title: string
  heading: string
  instruction: string
  description: string
}

export const runSummarySectionDefinitions: RunSummarySectionDefinition[] = [
  {
    key: 'overview',
    title: 'Overview',
    heading: '## Overview',
    instruction: 'Provide a compact high-level summary of the run.',
    description: 'Short overall summary for readers who do not need the full breakdown.',
  },
  {
    key: 'major_steps',
    title: 'Major Steps',
    heading: '## Major Steps',
    instruction: 'List the major steps in execution order.',
    description: 'Core execution path and milestone actions.',
  },
  {
    key: 'long_running_operations',
    title: 'Long-Running Operations',
    heading: '## Long-Running Operations',
    instruction: 'Call out operations that took unusually long.',
    description: 'Slow commands, waits, or long-running phases.',
  },
  {
    key: 'repeated_trial_and_error',
    title: 'Repeated Trial-and-Error',
    heading: '## Repeated Trial-and-Error',
    instruction: 'Call out repeated trial-and-error, retries, or churn.',
    description: 'Retries, failed loops, and other churn worth surfacing.',
  },
  {
    key: 'security_safety_risks',
    title: 'Security / Safety Risks',
    heading: '## Security / Safety Risks',
    instruction: 'Call out commands or file changes with elevated security or safety risk.',
    description: 'Potentially risky commands, approvals, or sensitive file changes.',
  },
  {
    key: 'files_touched',
    title: 'Files Touched',
    heading: '## Files Touched',
    instruction: 'Summarize the most important files or directories that were changed.',
    description: 'Useful when readers care about code and config surface area.',
  },
  {
    key: 'outcome',
    title: 'Outcome',
    heading: '## Outcome',
    instruction: 'Mention the main outcome and any important unresolved items.',
    description: 'Final result, status, and unresolved follow-ups.',
  },
  {
    key: 'commands_and_tooling',
    title: 'Commands and Tooling',
    heading: '## Commands and Tooling',
    instruction:
      'Highlight the most important commands, tools, and automation used during the run.',
    description: 'Good for operational and debugging transparency.',
  },
  {
    key: 'approvals_interruptions',
    title: 'Approvals / Interruptions',
    heading: '## Approvals / Interruptions',
    instruction:
      'Summarize approvals, interruptions, or operator intervention that affected the run.',
    description: 'Human intervention, approval gates, and interruptions.',
  },
  {
    key: 'run_metadata',
    title: 'Run Metadata',
    heading: '## Run Metadata',
    instruction:
      'Summarize relevant metadata such as duration, status, and token usage when it helps explain the run.',
    description: 'Timing, status, and token/cost context.',
  },
]

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
    'Summarize the overall work performed by the agent.',
    'Use concise Markdown.',
    'Include exactly these top-level sections in this order:',
    ...sections.flatMap((section) => [section.heading, section.instruction]),
    'If a selected section has no substantive content, write "None."',
  ]

  const trimmedCustomInstructions = customInstructions.trim()
  if (trimmedCustomInstructions !== '') {
    lines.push('', 'Additional instructions:', trimmedCustomInstructions)
  }

  return lines.join('\n')
}
