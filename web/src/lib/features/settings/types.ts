export const settingsSections = [
  'general',
  'repositories',
  'statuses',
  'agents',
  'notifications',
  'access',
  'security',
  'archived',
] as const

export type SettingsSection = (typeof settingsSections)[number]
