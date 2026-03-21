export const settingsSections = [
  'general',
  'repositories',
  'statuses',
  'workflows',
  'agents',
  'connectors',
  'notifications',
  'security',
] as const

export type SettingsSection = (typeof settingsSections)[number]
