import { expect, test } from './fixtures'

for (const route of ['/', '/orgs', '/orgs/org-e2e']) {
  test(`route ${route} keeps long content inside its page scroll container`, async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 460 })
    await page.goto(route)
    await page.waitForLoadState('domcontentloaded')

    const main = page.locator('main')
    const scrollContainer = page.getByTestId('route-scroll-container')

    await expect(scrollContainer).toBeVisible()

    await scrollContainer.evaluate((node) => {
      const host = node.firstElementChild ?? node
      const filler = document.createElement('div')
      filler.setAttribute('data-testid', 'route-scroll-filler')
      filler.style.height = '1200px'
      filler.style.pointerEvents = 'none'
      host.appendChild(filler)
    })

    const [mainMetrics, containerMetrics, documentMetrics] = await Promise.all([
      main.evaluate((node) => {
        const element = node as HTMLElement
        return {
          clientHeight: element.clientHeight,
          scrollHeight: element.scrollHeight,
        }
      }),
      scrollContainer.evaluate((node) => {
        const element = node as HTMLElement
        element.scrollTop = element.scrollHeight
        return {
          clientHeight: element.clientHeight,
          scrollHeight: element.scrollHeight,
          scrollTop: element.scrollTop,
        }
      }),
      page.evaluate(() => ({
        clientHeight: document.documentElement.clientHeight,
        scrollHeight: document.documentElement.scrollHeight,
        bodyScrollHeight: document.body.scrollHeight,
      })),
    ])

    expect(mainMetrics.scrollHeight).toBeLessThanOrEqual(mainMetrics.clientHeight + 1)
    expect(containerMetrics.scrollHeight).toBeGreaterThan(containerMetrics.clientHeight + 200)
    expect(containerMetrics.scrollTop).toBeGreaterThan(0)
    expect(documentMetrics.scrollHeight).toBe(documentMetrics.clientHeight)
    expect(documentMetrics.bodyScrollHeight).toBe(documentMetrics.clientHeight)
  })
}
