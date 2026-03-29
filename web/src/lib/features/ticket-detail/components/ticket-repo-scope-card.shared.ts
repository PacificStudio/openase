import Circle from '@lucide/svelte/icons/circle'
import CircleCheck from '@lucide/svelte/icons/circle-check'
import CircleX from '@lucide/svelte/icons/circle-x'
import Loader from '@lucide/svelte/icons/loader'

export const prStatusConfig: Record<string, { class: string; label: string }> = {
  open: { class: 'text-green-400', label: 'Open' },
  merged: { class: 'text-purple-400', label: 'Merged' },
  closed: { class: 'text-red-400', label: 'Closed' },
  draft: { class: 'text-muted-foreground', label: 'Draft' },
}

export const ciStatusConfig: Record<
  string,
  { icon: typeof CircleCheck; class: string; label: string }
> = {
  pass: { icon: CircleCheck, class: 'text-green-400', label: 'Passing' },
  fail: { icon: CircleX, class: 'text-red-400', label: 'Failing' },
  running: { icon: Loader, class: 'text-yellow-400 animate-spin', label: 'Running' },
  pending: { icon: Circle, class: 'text-muted-foreground', label: 'Pending' },
}

export function getPrStatusLabel(status?: string) {
  return status ? (prStatusConfig[status]?.label ?? status) : 'Unset'
}

export function getCiStatusLabel(status?: string) {
  return status ? (ciStatusConfig[status]?.label ?? status) : 'Unset'
}
