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

export function resolveSettingsSectionHash(
  rawHash: string,
  fallback: SettingsSection,
): SettingsSection {
  const value = rawHash.startsWith('#') ? rawHash.slice(1) : rawHash

  // Preserve old project settings links after removing the legacy access tab.
  if (value === 'access') {
    return 'security'
  }

  return settingsSections.includes(value as SettingsSection) ? (value as SettingsSection) : fallback
}
