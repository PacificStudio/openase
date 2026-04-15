import { driver, type DriveStep, type Config } from 'driver.js'
import 'driver.js/dist/driver.css'
import type { TranslationKey } from '$lib/i18n'
import { toursAllowed } from '$lib/features/tour/runtime'

const TOUR_STORAGE_KEY = 'openase.tour.workflowCreation.completed'

type TourStepDef = {
  tourId: string
  titleKey: TranslationKey
  descriptionKey: TranslationKey
  side?: 'top' | 'right' | 'bottom' | 'left'
}

const STEPS: TourStepDef[] = [
  {
    tourId: 'workflow-create-name',
    titleKey: 'tour.workflowCreation.name.title',
    descriptionKey: 'tour.workflowCreation.name.description',
    side: 'bottom',
  },
  {
    tourId: 'workflow-create-type',
    titleKey: 'tour.workflowCreation.type.title',
    descriptionKey: 'tour.workflowCreation.type.description',
    side: 'bottom',
  },
  {
    tourId: 'workflow-create-agent',
    titleKey: 'tour.workflowCreation.agent.title',
    descriptionKey: 'tour.workflowCreation.agent.description',
    side: 'bottom',
  },
  {
    tourId: 'workflow-create-pickup',
    titleKey: 'tour.workflowCreation.pickup.title',
    descriptionKey: 'tour.workflowCreation.pickup.description',
    side: 'top',
  },
  {
    tourId: 'workflow-create-finish',
    titleKey: 'tour.workflowCreation.finish.title',
    descriptionKey: 'tour.workflowCreation.finish.description',
    side: 'top',
  },
  {
    tourId: 'workflow-create-stages-explainer',
    titleKey: 'tour.workflowCreation.stages.title',
    descriptionKey: 'tour.workflowCreation.stages.description',
    side: 'top',
  },
]

function hasBeenShown(projectId: string): boolean {
  try {
    const raw = localStorage.getItem(TOUR_STORAGE_KEY)
    if (!raw) return false
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) return parsed.includes(projectId)
    return false
  } catch {
    return false
  }
}

function markShown(projectId: string) {
  try {
    const raw = localStorage.getItem(TOUR_STORAGE_KEY)
    const list: string[] = raw ? (JSON.parse(raw) ?? []) : []
    if (!list.includes(projectId)) {
      list.push(projectId)
      localStorage.setItem(TOUR_STORAGE_KEY, JSON.stringify(list))
    }
  } catch {
    // storage unavailable
  }
}

function buildSteps(t: (key: TranslationKey) => string): DriveStep[] {
  const steps: DriveStep[] = []
  for (const def of STEPS) {
    const el = document.querySelector(`[data-tour="${def.tourId}"]`)
    if (!el) continue
    steps.push({
      element: `[data-tour="${def.tourId}"]`,
      popover: {
        title: t(def.titleKey),
        description: t(def.descriptionKey),
        side: def.side ?? 'bottom',
        align: 'start',
      },
    })
  }
  return steps
}

export function maybeStartWorkflowCreationTour(
  projectId: string,
  t: (key: TranslationKey) => string,
  options: { force?: boolean } = {},
) {
  if (!toursAllowed(projectId)) return
  if (!options.force && hasBeenShown(projectId)) return

  const steps = buildSteps(t)
  if (steps.length === 0) return

  const driverObj = driver({
    showProgress: steps.length > 1,
    animate: true,
    allowClose: true,
    overlayColor: 'black',
    overlayOpacity: 0.5,
    stagePadding: 6,
    stageRadius: 8,
    popoverOffset: 12,
    nextBtnText: t('tour.btn.next'),
    prevBtnText: t('tour.btn.prev'),
    doneBtnText: t('tour.btn.done'),
    progressText: '{{current}} / {{total}}',
    steps,
    onDestroyed: () => {
      markShown(projectId)
    },
  } satisfies Config)

  driverObj.drive()
}
