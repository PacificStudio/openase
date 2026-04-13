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

export function presetText(key: string, field: string) {
  const entry = presetCopy[`${key}.${field}`]
  return entry?.fallback ?? ''
}
