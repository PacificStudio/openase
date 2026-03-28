import { test as base, expect } from '@playwright/test'
type E2EFixtures = {
  projectPath: (section: string) => string
}

export const test = base.extend<E2EFixtures>({
  projectPath: async ({ page: _page }, use) => {
    const builder = (section: string) => `/orgs/org-e2e/projects/project-e2e/${section}`
    await use(builder)
  },
})

test.beforeEach(async ({ request, page }) => {
  const response = await request.post('/api/v1/__e2e__/reset')
  expect(response.ok()).toBeTruthy()

  await page
    .addStyleTag({
      content: `
      *,
      *::before,
      *::after {
        transition-duration: 0ms !important;
        animation-duration: 0ms !important;
        animation-delay: 0ms !important;
        scroll-behavior: auto !important;
      }
    `,
    })
    .catch(() => {})
})

export { expect } from '@playwright/test'
