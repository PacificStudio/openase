import { driver, type DriveStep, type Config } from 'driver.js'
import 'driver.js/dist/driver.css'
import type { TranslationKey } from '$lib/i18n'
import { toursAllowed } from './runtime'
import type { ProjectSection } from '$lib/stores/app-context'
import { PAGE_TOURS, type PageSection, type PageTourStep } from './page-tours-data'

const PAGE_TOUR_STORAGE_KEY = 'openase.tour.pages.completed'

type CompletedMap = Record<string, string[]>

function readCompletedMap(): CompletedMap {
  try {
    const raw = localStorage.getItem(PAGE_TOUR_STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as CompletedMap
    }
    return {}
  } catch {
    return {}
  }
}

function writeCompletedMap(map: CompletedMap) {
  try {
    localStorage.setItem(PAGE_TOUR_STORAGE_KEY, JSON.stringify(map))
  } catch {
    // storage unavailable
  }
}

export function hasPageTourBeenShown(section: PageSection, projectId: string): boolean {
  const map = readCompletedMap()
  return (map[projectId] ?? []).includes(section)
}

function markPageTourShown(section: PageSection, projectId: string) {
  const map = readCompletedMap()
  const list = map[projectId] ?? []
  if (!list.includes(section)) {
    list.push(section)
    map[projectId] = list
    writeCompletedMap(map)
  }
}

function buildSteps(defs: PageTourStep[], t: (key: TranslationKey) => string): DriveStep[] {
  const steps: DriveStep[] = []
  for (const def of defs) {
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

export function startPageTour(
  section: PageSection,
  projectId: string,
  t: (key: TranslationKey) => string,
) {
  if (!toursAllowed(projectId)) return
  const defs = PAGE_TOURS[section]
  if (!defs || defs.length === 0) return
  const steps = buildSteps(defs, t)
  if (steps.length === 0) return

  const driverObj = driver({
    showProgress: steps.length > 1,
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
      markPageTourShown(section, projectId)
    },
  } satisfies Config)

  driverObj.drive()
}

export function isPageSection(section: ProjectSection): section is PageSection {
  return section !== 'dashboard'
}
