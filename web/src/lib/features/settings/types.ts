export const settingsSections = [
  'general',
  'repositories',
  'statuses',
  'agents',
  'notifications',
  'security',
] as const

export type SettingsSection = (typeof settingsSections)[number]
