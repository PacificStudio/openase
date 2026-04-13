import type { TranslationKey } from '$lib/i18n'
import type { ScopedSecret, ScopedSecretBinding, Ticket, Workflow } from '$lib/api/contracts'

export type SecretBindingDraft = {
  bindingKey: string
  scope: 'workflow' | 'ticket'
  scopeResourceId: string
  secretId: string
}

export const scopeOptions: { value: SecretBindingDraft['scope']; labelKey: TranslationKey }[] = [
  { value: 'workflow', labelKey: 'settings.secretBindings.scope.workflow' },
  { value: 'ticket', labelKey: 'settings.secretBindings.scope.ticket' },
]

export function scopeLabelKey(scope: SecretBindingDraft['scope']): TranslationKey {
  return scope === 'ticket'
    ? 'settings.secretBindings.scope.ticket'
    : 'settings.secretBindings.scope.workflow'
}

export function secretScopeLabel(secret: Pick<ScopedSecret, 'scope'>) {
  return secret.scope === 'organization'
    ? 'settings.secretBindings.scopeLabel.organization'
    : 'settings.secretBindings.scopeLabel.project'
}

export function targetLabel(target: Workflow | Ticket) {
  if ('identifier' in target) {
    return `${target.identifier} - ${target.title}`
  }
  return target.name
}

export function bindingTargetLabel(binding: ScopedSecretBinding) {
  if (binding.target.identifier) {
    return `${binding.target.identifier} - ${binding.target.name}`
  }
  return binding.target.name
}
