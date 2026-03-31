export const settingsSections = [
  'general',
  'skills',
  'repositories',
  'statuses',
  'workflows',
  'agents',
  'connectors',
  'notifications',
  'security',
] as const

export type SettingsSection = (typeof settingsSections)[number]
