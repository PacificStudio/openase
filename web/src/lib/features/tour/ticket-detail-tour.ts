import { driver, type DriveStep, type Config } from 'driver.js'
import 'driver.js/dist/driver.css'
import type { TranslationKey } from '$lib/i18n'
import { toursAllowed } from './runtime'

const STORAGE_KEY = 'openase.tour.ticketDetail.completed'

type TourStep = {
  tourId: string
  titleKey: TranslationKey
  descriptionKey: TranslationKey
  side?: 'top' | 'right' | 'bottom' | 'left'
  align?: 'start' | 'center' | 'end'
}

const STEPS: TourStep[] = [
  {
    tourId: 'ticket-detail-header',
    titleKey: 'tour.ticketDetail.header.title',
    descriptionKey: 'tour.ticketDetail.header.description',
    side: 'bottom',
    align: 'start',
  },
  {
    tourId: 'ticket-detail-sidebar',
    titleKey: 'tour.ticketDetail.sidebar.title',
    descriptionKey: 'tour.ticketDetail.sidebar.description',
    side: 'left',
    align: 'start',
  },
  {
    tourId: 'ticket-detail-tab-discussion',
    titleKey: 'tour.ticketDetail.discussion.title',
    descriptionKey: 'tour.ticketDetail.discussion.description',
    side: 'bottom',
    align: 'start',
  },
  {
    tourId: 'ticket-detail-tab-runs',
    titleKey: 'tour.ticketDetail.runs.title',
    descriptionKey: 'tour.ticketDetail.runs.description',
    side: 'bottom',
    align: 'start',
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

export function hasTicketDetailTourBeenShown(projectId: string): boolean {
  return readCompleted().includes(projectId)
}

export function startTicketDetailTour(projectId: string, t: (key: TranslationKey) => string) {
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
