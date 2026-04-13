import { i18nStore } from '$lib/i18n/store.svelte'
import { formatRelativeTime } from '$lib/utils'

function localizeRelativeTime(raw: string): string {
  if (i18nStore.locale !== 'zh') {
    return raw
  }
  if (raw === 'just now') {
    return '刚刚'
  }

  const match = raw.match(/^(\d+)([mhd]) ago$/)
  if (!match) {
    return raw
  }

  const [, value, unit] = match
  switch (unit) {
    case 'm':
      return `${value} 分钟前`
    case 'h':
      return `${value} 小时前`
    case 'd':
      return `${value} 天前`
    default:
      return raw
  }
}

export function formatMachineRelativeTime(date: string | Date): string {
  return localizeRelativeTime(formatRelativeTime(date))
}
