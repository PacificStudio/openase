import { AlertTriangle, CircleCheck, CircleX } from '@lucide/svelte'
import type { ProjectUpdateStatus } from './types'
import { projectUpdatesT } from './i18n'

export const projectUpdateStatusConfig: Record<
  ProjectUpdateStatus,
  { icon: typeof CircleCheck; dotClass: string; textClass: string }
> = {
  on_track: {
    icon: CircleCheck,
    dotClass: 'text-emerald-500',
    textClass: 'text-emerald-700 dark:text-emerald-400',
  },
  at_risk: {
    icon: AlertTriangle,
    dotClass: 'text-amber-500',
    textClass: 'text-amber-700 dark:text-amber-400',
  },
  off_track: {
    icon: CircleX,
    dotClass: 'text-rose-500',
    textClass: 'text-rose-700 dark:text-rose-400',
  },
}

export const projectUpdateStatusOptions: Array<{
  value: ProjectUpdateStatus
  label: string
  icon: typeof CircleCheck
  textClass: string
}> = [
  {
    value: 'on_track',
    label: projectUpdatesT('projectUpdates.status.onTrack'),
    icon: CircleCheck,
    textClass: 'text-emerald-600',
  },
  {
    value: 'at_risk',
    label: projectUpdatesT('projectUpdates.status.atRisk'),
    icon: AlertTriangle,
    textClass: 'text-amber-600',
  },
  {
    value: 'off_track',
    label: projectUpdatesT('projectUpdates.status.offTrack'),
    icon: CircleX,
    textClass: 'text-rose-600',
  },
]
