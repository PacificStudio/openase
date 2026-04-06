import { expect, type Locator, type Page } from '@playwright/test'

export type LocatorDescriptor = {
  kind: 'button' | 'heading' | 'label' | 'link' | 'placeholder' | 'test-id' | 'text'
  value: string
  exact?: boolean
}

export type ResponsivePolicy = {
  routeId: string
  support: 'mobile-supported' | 'tablet-supported' | 'desktop-only'
  smoke?: {
    heading: string
    criticalControls: LocatorDescriptor[]
  }
}

export function buildPolicyUrl(
  projectPath: (section: string) => string,
  routeId: string,
  hash = '',
) {
  const base = routeId === 'dashboard' ? '/orgs/org-e2e/projects/project-e2e' : projectPath(routeId)
  return hash ? `${base}${hash}` : base
}

export function policyAppliesToProject(policy: ResponsivePolicy, projectName: string) {
  return isTabletProject(projectName)
    ? policy.support === 'mobile-supported' || policy.support === 'tablet-supported'
    : policy.support === 'mobile-supported'
}

export function isTabletProject(projectName: string) {
  return projectName.startsWith('tablet-')
}

export function locatorFromDescriptor(page: Page, descriptor: LocatorDescriptor): Locator {
  switch (descriptor.kind) {
    case 'button':
      return page.getByRole('button', { name: descriptor.value, exact: descriptor.exact })
    case 'heading':
      return page.getByRole('heading', { name: descriptor.value, exact: descriptor.exact })
    case 'label':
      return page.getByLabel(descriptor.value, { exact: descriptor.exact })
    case 'link':
      return page.getByRole('link', { name: descriptor.value, exact: descriptor.exact })
    case 'placeholder':
      return page.getByPlaceholder(descriptor.value, { exact: descriptor.exact })
    case 'test-id':
      return page.getByTestId(descriptor.value)
    case 'text':
      return page.getByText(descriptor.value, { exact: descriptor.exact })
    default: {
      const exhaustive: never = descriptor.kind
      throw new Error(`Unsupported locator kind: ${exhaustive}`)
    }
  }
}

export async function assertNoUnexpectedHorizontalScroll(page: Page) {
  const metrics = await page.evaluate(() => ({
    rootClientWidth: document.documentElement.clientWidth,
    rootScrollWidth: document.documentElement.scrollWidth,
    bodyScrollWidth: document.body.scrollWidth,
  }))

  expect(metrics.rootScrollWidth).toBeLessThanOrEqual(metrics.rootClientWidth + 1)
  expect(metrics.bodyScrollWidth).toBeLessThanOrEqual(metrics.rootClientWidth + 1)
}

export async function assertDescriptorsReachable(page: Page, descriptors: LocatorDescriptor[]) {
  for (const descriptor of descriptors) {
    await assertDescriptorReachable(page, descriptor)
  }
}

export async function assertDescriptorReachable(page: Page, descriptor: LocatorDescriptor) {
  const locator = locatorFromDescriptor(page, descriptor).first()
  await expect(locator).toBeVisible()

  const box = await locator.boundingBox()
  expect(box, `${descriptor.kind}:${descriptor.value} should render a bounding box`).not.toBeNull()
  if (!box) {
    return
  }

  const viewport = page.viewportSize()
  expect(viewport).not.toBeNull()
  if (!viewport) {
    return
  }

  expect(box.x).toBeGreaterThanOrEqual(0)
  expect(box.y).toBeGreaterThanOrEqual(0)
  expect(box.x + box.width).toBeLessThanOrEqual(viewport.width + 1)
  expect(box.y + box.height).toBeLessThanOrEqual(viewport.height + 1)
}

export async function assertDescriptorsDoNotOverlap(page: Page, descriptors: LocatorDescriptor[]) {
  const visibleLocators = await Promise.all(
    descriptors.map(async (descriptor) => ({
      descriptor,
      box: await locatorFromDescriptor(page, descriptor).first().boundingBox(),
    })),
  )

  const boxes = visibleLocators.filter(
    (entry): entry is { descriptor: LocatorDescriptor; box: NonNullable<typeof entry.box> } =>
      entry.box !== null,
  )

  for (let leftIndex = 0; leftIndex < boxes.length; leftIndex += 1) {
    for (let rightIndex = leftIndex + 1; rightIndex < boxes.length; rightIndex += 1) {
      const left = boxes[leftIndex]
      const right = boxes[rightIndex]
      const overlapWidth = Math.max(
        0,
        Math.min(left.box.x + left.box.width, right.box.x + right.box.width) -
          Math.max(left.box.x, right.box.x),
      )
      const overlapHeight = Math.max(
        0,
        Math.min(left.box.y + left.box.height, right.box.y + right.box.height) -
          Math.max(left.box.y, right.box.y),
      )
      expect(
        overlapWidth * overlapHeight,
        `${left.descriptor.value} should not overlap ${right.descriptor.value}`,
      ).toBe(0)
    }
  }
}

export async function openSearchPalette(page: Page) {
  await page.getByRole('button', { name: /Search/i }).click()
  const input = page.getByPlaceholder('Search pages, tickets, workflows, agents, and commands...')
  await expect(input).toBeVisible()
  return input
}

export async function openAndAssertSurface(
  page: Page,
  opener: LocatorDescriptor,
  ready: LocatorDescriptor,
) {
  await locatorFromDescriptor(page, opener).first().click({ noWaitAfter: true })
  const readyLocator = locatorFromDescriptor(page, ready).first()
  await expect(readyLocator).toBeVisible()
  return readyLocator
}
