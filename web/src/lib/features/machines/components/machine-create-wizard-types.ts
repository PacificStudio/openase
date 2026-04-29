import type { TranslationKey } from '$lib/i18n'
import type { Component } from 'svelte'

export type MachineWizardStrategy = 'direct-open' | 'ssh-install-listener' | 'reverse'
export type MachineWizardLocationAnswer = 'local' | 'remote'
export type MachineWizardStep =
  | 'location'
  | 'identity'
  | 'strategy'
  | 'credentials'
  | 'advertised-endpoint'
  | 'review'

export type MachineWizardOptionCard<T> = {
  value: T
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  icon: Component<any>
  titleKey: TranslationKey
  descKey: TranslationKey
}
