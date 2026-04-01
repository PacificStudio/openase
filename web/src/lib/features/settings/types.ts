export const settingsSections = [
  'general',
  'repositories',
  'statuses',
  'agents',
  'connectors',
  'notifications',
  'security',
] as const

export type SettingsSection = (typeof settingsSections)[number]
