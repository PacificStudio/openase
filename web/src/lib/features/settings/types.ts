export const settingsSections = [
  'general',
  'repositories',
  'statuses',
  'agents',
  'notifications',
  'security',
  'archived',
] as const

export type SettingsSection = (typeof settingsSections)[number]
