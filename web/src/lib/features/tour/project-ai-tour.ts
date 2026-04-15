import { driver, type DriveStep, type Config } from 'driver.js'
import 'driver.js/dist/driver.css'
import type { TranslationKey } from '$lib/i18n'
import { toursAllowed } from './runtime'

const STORAGE_KEY = 'openase.tour.projectAI.completed'

type TourStep = {
  tourId: string
  titleKey: TranslationKey
  descriptionKey: TranslationKey
  side?: 'top' | 'right' | 'bottom' | 'left'
  align?: 'start' | 'center' | 'end'
}

const STEPS: TourStep[] = [
  {
    tourId: 'project-ai-header',
    titleKey: 'tour.projectAI.header.title',
    descriptionKey: 'tour.projectAI.header.description',
    side: 'bottom',
    align: 'start',
  },
  {
    tourId: 'project-ai-isolation',
    titleKey: 'tour.projectAI.isolation.title',
    descriptionKey: 'tour.projectAI.isolation.description',
    side: 'left',
    align: 'start',
  },
  {
    tourId: 'project-ai-suggestions',
    titleKey: 'tour.projectAI.suggestions.title',
    descriptionKey: 'tour.projectAI.suggestions.description',
    side: 'left',
    align: 'start',
  },
  {
    tourId: 'project-ai-composer',
    titleKey: 'tour.projectAI.composer.title',
    descriptionKey: 'tour.projectAI.composer.description',
    side: 'top',
    align: 'start',
  },
  {
    tourId: 'project-ai-history',
    titleKey: 'tour.projectAI.history.title',
    descriptionKey: 'tour.projectAI.history.description',
    side: 'bottom',
    align: 'end',
  },
]

function readCompleted(): string[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function markCompleted(projectId: string) {
  try {
    const list = readCompleted()
    if (!list.includes(projectId)) {
      list.push(projectId)
      localStorage.setItem(STORAGE_KEY, JSON.stringify(list))
    }
  } catch {
    // storage unavailable
  }
}

export function hasProjectAITourBeenShown(projectId: string): boolean {
  return readCompleted().includes(projectId)
}

export function startProjectAITour(projectId: string, t: (key: TranslationKey) => string) {
  if (!toursAllowed(projectId)) return
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
        align: def.align ?? 'start',
      },
    })
  }
  if (steps.length === 0) return

  const driverObj = driver({
    showProgress: steps.length > 1,
    animate: true,
    allowClose: true,
    overlayColor: 'black',
    overlayOpacity: 0.5,
    stagePadding: 6,
    stageRadius: 8,
    popoverOffset: 10,
    nextBtnText: t('tour.btn.next'),
    prevBtnText: t('tour.btn.prev'),
    doneBtnText: t('tour.btn.done'),
    progressText: '{{current}} / {{total}}',
    steps,
    onDestroyed: () => {
      markCompleted(projectId)
    },
  } satisfies Config)

  driverObj.drive()
}
