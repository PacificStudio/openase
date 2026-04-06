import { expect, test } from './fixtures'

test('updates page keeps scrolling inside page content when Project AI is open', async ({
  page,
  projectPath,
}) => {
  await page.setViewportSize({ width: 1280, height: 460 })
  await page.goto(projectPath('updates'))
  await page.waitForLoadState('domcontentloaded')
  await page.waitForTimeout(500)

  await page.getByRole('button', { name: 'Ask AI' }).click()
  await expect(page.getByPlaceholder('Ask anything about this project…')).toBeVisible({
    timeout: 10_000,
  })

  const main = page.locator('main')
  const pageContent = page.locator('[data-testid="page-scaffold-content"]')

  await expect(pageContent).toBeVisible()

  const [mainMetrics, contentMetrics, documentMetrics] = await Promise.all([
    main.evaluate((node) => {
      const element = node as HTMLElement
      return {
        clientHeight: element.clientHeight,
        scrollHeight: element.scrollHeight,
      }
    }),
    pageContent.evaluate((node) => {
      const element = node as HTMLElement
      return {
        clientHeight: element.clientHeight,
        scrollHeight: element.scrollHeight,
      }
    }),
    page.evaluate(() => ({
      clientHeight: document.documentElement.clientHeight,
      scrollHeight: document.documentElement.scrollHeight,
      bodyScrollHeight: document.body.scrollHeight,
    })),
  ])

  expect(mainMetrics.scrollHeight).toBeLessThanOrEqual(mainMetrics.clientHeight + 1)
  expect(contentMetrics.scrollHeight).toBeGreaterThan(contentMetrics.clientHeight + 40)
  expect(documentMetrics.scrollHeight).toBe(documentMetrics.clientHeight)
  expect(documentMetrics.bodyScrollHeight).toBe(documentMetrics.clientHeight)
})
