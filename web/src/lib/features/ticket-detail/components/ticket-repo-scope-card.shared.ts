import Circle from '@lucide/svelte/icons/circle'
import CircleCheck from '@lucide/svelte/icons/circle-check'
import CircleX from '@lucide/svelte/icons/circle-x'
import Loader from '@lucide/svelte/icons/loader'
import type { TranslationKey } from '$lib/i18n'
import { i18nStore } from '$lib/i18n/store.svelte'

export const prStatusConfig: Record<string, { class: string; labelKey: TranslationKey }> = {
  open: { class: 'text-green-400', labelKey: 'ticketDetail.repoScope.prStatus.open' },
  merged: { class: 'text-purple-400', labelKey: 'ticketDetail.repoScope.prStatus.merged' },
  closed: { class: 'text-red-400', labelKey: 'ticketDetail.repoScope.prStatus.closed' },
  draft: { class: 'text-muted-foreground', labelKey: 'ticketDetail.repoScope.prStatus.draft' },
}

export const ciStatusConfig: Record<
  string,
  { icon: typeof CircleCheck; class: string; labelKey: TranslationKey }
> = {
  pass: {
    icon: CircleCheck,
    class: 'text-green-400',
    labelKey: 'ticketDetail.repoScope.ciStatus.pass',
  },
  fail: {
    icon: CircleX,
    class: 'text-red-400',
    labelKey: 'ticketDetail.repoScope.ciStatus.fail',
  },
  running: {
    icon: Loader,
    class: 'text-yellow-400 animate-spin',
    labelKey: 'ticketDetail.repoScope.ciStatus.running',
  },
  pending: {
    icon: Circle,
    class: 'text-muted-foreground',
    labelKey: 'ticketDetail.repoScope.ciStatus.pending',
  },
}

const unsetStatusLabelKey: TranslationKey = 'ticketDetail.repoScope.status.unset'
const unknownStatusLabelKey: TranslationKey = 'ticketDetail.repoScope.status.unknown'

export function getPrStatusLabel(status?: string) {
  if (!status) {
    return i18nStore.t(unsetStatusLabelKey)
  }
  const labelKey = prStatusConfig[status]?.labelKey
  if (labelKey) {
    return i18nStore.t(labelKey)
  }
  return i18nStore.t(unknownStatusLabelKey, { status })
}

export function getCiStatusLabel(status?: string) {
  if (!status) {
    return i18nStore.t(unsetStatusLabelKey)
  }
  const labelKey = ciStatusConfig[status]?.labelKey
  if (labelKey) {
    return i18nStore.t(labelKey)
  }
  return i18nStore.t(unknownStatusLabelKey, { status })
}
