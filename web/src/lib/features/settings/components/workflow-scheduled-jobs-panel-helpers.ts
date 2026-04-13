import { ApiError } from '$lib/api/client'
import type { TranslationKey } from '$lib/i18n'
import { i18nStore } from '$lib/i18n/store.svelte'
import { toastStore } from '$lib/stores/toast.svelte'

export const scheduledJobToggleMessageKeys = {
  enabled: 'settings.workflowScheduledJobs.messages.enabled',
  disabled: 'settings.workflowScheduledJobs.messages.disabled',
} as const satisfies Record<'enabled' | 'disabled', TranslationKey>

export function showScheduledJobError(caughtError: unknown, fallbackKey: TranslationKey) {
  toastStore.error(caughtError instanceof ApiError ? caughtError.detail : i18nStore.t(fallbackKey))
}
