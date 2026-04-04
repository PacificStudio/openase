import type { ProjectUpdateStatus } from './types'

export const projectUpdateStatusOptions: Array<{ value: ProjectUpdateStatus; label: string }> = [
  { value: 'on_track', label: 'On track' },
  { value: 'at_risk', label: 'At risk' },
  { value: 'off_track', label: 'Off track' },
]

export function projectUpdateStatusLabel(status: ProjectUpdateStatus) {
  return projectUpdateStatusOptions.find((option) => option.value === status)?.label ?? status
}

export function projectUpdateStatusBadgeClass(status: ProjectUpdateStatus) {
  switch (status) {
    case 'on_track':
      return 'border-emerald-200 bg-emerald-50 text-emerald-700'
    case 'at_risk':
      return 'border-amber-200 bg-amber-50 text-amber-700'
    case 'off_track':
      return 'border-rose-200 bg-rose-50 text-rose-700'
  }
}
