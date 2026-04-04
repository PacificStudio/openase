import { measureNavigation } from './perf'
import { expect, test } from './fixtures'

test('project sidebar navigation stays responsive', async ({ page, projectPath }, testInfo) => {
  await measureNavigation({
    page,
    scenario: 'machines_route_initial_load',
    budgetMs: 800,
    ready: page.getByTestId('machines-workspace'),
    testInfo,
    action: async () => {
      await page.goto(projectPath('machines'))
    },
  })

  await measureNavigation({
    page,
    scenario: 'sidebar_nav_agents',
    budgetMs: 800,
    ready: page.getByRole('heading', { name: 'Agents' }),
    testInfo,
    action: async () => {
      await page.getByRole('link', { name: 'Agents' }).click()
    },
  })

  await measureNavigation({
    page,
    scenario: 'sidebar_nav_scheduled_jobs',
    budgetMs: 800,
    ready: page.getByRole('heading', { name: 'Scheduled Jobs' }),
    testInfo,
    action: async () => {
      await page.getByRole('link', { name: 'Scheduled Jobs' }).click()
    },
  })

  await measureNavigation({
    page,
    scenario: 'sidebar_nav_workflows',
    budgetMs: 800,
    ready: page.getByRole('heading', { name: 'Workflows' }),
    testInfo,
    action: async () => {
      await page.getByRole('link', { name: 'Workflows' }).click()
    },
  })

  await expect(page.getByRole('heading', { name: 'Workflows' })).toBeVisible()
})
