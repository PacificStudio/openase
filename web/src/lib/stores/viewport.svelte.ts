/**
 * Reactive viewport breakpoint detection for mobile adaptation.
 *
 * Breakpoints align with Tailwind defaults:
 * - mobile: < 768px  (below `md`)
 * - tablet: 768–1023px (`md` but below `lg`)
 * - desktop: >= 1024px (`lg`+)
 */

const MOBILE_MAX = 768
const DESKTOP_MIN = 1024

function createViewportStore() {
  let width = $state(typeof window !== 'undefined' ? window.innerWidth : DESKTOP_MIN)

  if (typeof window !== 'undefined') {
    const mqlMobile = window.matchMedia(`(max-width: ${MOBILE_MAX - 1}px)`)
    const mqlDesktop = window.matchMedia(`(min-width: ${DESKTOP_MIN}px)`)

    function sync() {
      width = window.innerWidth
    }

    mqlMobile.addEventListener('change', sync)
    mqlDesktop.addEventListener('change', sync)
  }

  return {
    get isMobile() {
      return width < MOBILE_MAX
    },
    get isTablet() {
      return width >= MOBILE_MAX && width < DESKTOP_MIN
    },
    get isDesktop() {
      return width >= DESKTOP_MIN
    },
    get width() {
      return width
    },
  }
}

export const viewport = createViewportStore()
