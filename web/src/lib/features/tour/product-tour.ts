import { driver, type DriveStep, type Config } from 'driver.js'
import 'driver.js/dist/driver.css'
import type { TranslationKey } from '$lib/i18n'

export const TOUR_STORAGE_KEY = 'openase.tour.completed'

export function hasTourBeenShown(projectId: string): boolean {
  try {
    const raw = localStorage.getItem(TOUR_STORAGE_KEY)
    if (!raw) return false
    const shown: string[] = JSON.parse(raw)
    return shown.includes(projectId)
  } catch {
    return false
  }
}

function markTourShown(projectId: string) {
  try {
    const raw = localStorage.getItem(TOUR_STORAGE_KEY)
    const shown: string[] = raw ? JSON.parse(raw) : []
    if (!shown.includes(projectId)) {
      shown.push(projectId)
    }
    localStorage.setItem(TOUR_STORAGE_KEY, JSON.stringify(shown))
  } catch {
    // storage unavailable
  }
}

type TourStepDef = {
  tourId: string
  titleKey: TranslationKey
  descriptionKey: TranslationKey
  side?: 'top' | 'right' | 'bottom' | 'left'
}

const DASHBOARD_STEPS: TourStepDef[] = [
  {
    tourId: 'dashboard-stats',
    titleKey: 'tour.dashboardStats.title',
    descriptionKey: 'tour.dashboardStats.description',
    side: 'bottom',
  },
]

const TOPBAR_STEPS: TourStepDef[] = [
  {
    tourId: 'topbar-search',
    titleKey: 'tour.topbarSearch.title',
    descriptionKey: 'tour.topbarSearch.description',
    side: 'bottom',
  },
  {
    tourId: 'topbar-new-ticket',
    titleKey: 'tour.topbarNewTicket.title',
    descriptionKey: 'tour.topbarNewTicket.description',
    side: 'bottom',
  },
  {
    tourId: 'topbar-sse-status',
    titleKey: 'tour.topbarSseStatus.title',
    descriptionKey: 'tour.topbarSseStatus.description',
    side: 'bottom',
  },
]

const SIDEBAR_STEPS: TourStepDef[] = [
  {
    tourId: 'sidebar-ai-assistant',
    titleKey: 'tour.sidebarAiAssistant.title',
    descriptionKey: 'tour.sidebarAiAssistant.description',
    side: 'right',
  },
  {
    tourId: 'nav-tickets',
    titleKey: 'tour.navTickets.title',
    descriptionKey: 'tour.navTickets.description',
    side: 'right',
  },
  {
    tourId: 'nav-agents',
    titleKey: 'tour.navAgents.title',
    descriptionKey: 'tour.navAgents.description',
    side: 'right',
  },
  {
    tourId: 'nav-machines',
    titleKey: 'tour.navMachines.title',
    descriptionKey: 'tour.navMachines.description',
    side: 'right',
  },
  {
    tourId: 'nav-updates',
    titleKey: 'tour.navUpdates.title',
    descriptionKey: 'tour.navUpdates.description',
    side: 'right',
  },
  {
    tourId: 'nav-activity',
    titleKey: 'tour.navActivity.title',
    descriptionKey: 'tour.navActivity.description',
    side: 'right',
  },
  {
    tourId: 'nav-workflows',
    titleKey: 'tour.navWorkflows.title',
    descriptionKey: 'tour.navWorkflows.description',
    side: 'right',
  },
  {
    tourId: 'nav-skills',
    titleKey: 'tour.navSkills.title',
    descriptionKey: 'tour.navSkills.description',
    side: 'right',
  },
  {
    tourId: 'nav-scheduled-jobs',
    titleKey: 'tour.navScheduledJobs.title',
    descriptionKey: 'tour.navScheduledJobs.description',
    side: 'right',
  },
  {
    tourId: 'nav-settings',
    titleKey: 'tour.navSettings.title',
    descriptionKey: 'tour.navSettings.description',
    side: 'right',
  },
]

const ALL_STEPS: TourStepDef[] = [...DASHBOARD_STEPS, ...TOPBAR_STEPS, ...SIDEBAR_STEPS]

function buildSteps(t: (key: TranslationKey) => string): DriveStep[] {
  const steps: DriveStep[] = []

  for (const def of ALL_STEPS) {
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

export function startProductTour(projectId: string, t: (key: TranslationKey) => string) {
  const steps = buildSteps(t)
  if (steps.length === 0) return

  const driverObj = driver({
    showProgress: true,
    animate: true,
    allowClose: true,
    overlayColor: 'black',
    overlayOpacity: 0.5,
    stagePadding: 8,
    stageRadius: 8,
    popoverOffset: 12,
    nextBtnText: t('tour.btn.next'),
    prevBtnText: t('tour.btn.prev'),
    doneBtnText: t('tour.btn.done'),
    progressText: '{{current}} / {{total}}',
    steps,
    onDestroyed: () => {
      markTourShown(projectId)
    },
  } satisfies Config)

  driverObj.drive()
}
