import { measureCompletion, measureFeedback, measureNavigation } from './perf'
import { expect, test } from './fixtures'

test('machines drawer edit remains fast', async ({ page, projectPath }, testInfo) => {
  await measureNavigation({
    page,
    scenario: 'machines_page_ready',
    budgetMs: 800,
    ready: page.getByTestId('machine-card-machine-gpu'),
    testInfo,
    action: async () => {
      await page.goto(projectPath('machines'))
    },
  })

  await measureFeedback({
    scenario: 'machines_drawer_open',
    budgetMs: 150,
    ready: page.getByTestId('machine-editor-sheet'),
    testInfo,
    action: async () => {
      await page
        .getByTestId('machine-card-machine-gpu')
        .getByRole('button', { name: 'View details' })
        .click()
    },
  })

  await page.getByLabel('Description').fill('Updated from Playwright perf regression run')

  await measureCompletion({
    scenario: 'machines_save_complete',
    budgetMs: 1500,
    ready: page.getByText('Machine updated.'),
    testInfo,
    action: async () => {
      await page.getByTestId('machine-save-button').click({ noWaitAfter: true })
    },
  })
})

test('machines test connection keeps resource snapshot intact', async ({
  page,
  projectPath,
}, testInfo) => {
  await page.goto(projectPath('machines'))

  const gpuCard = page.getByTestId('machine-card-machine-gpu')
  const resources = page.getByTestId('machine-resources-machine-gpu')

  await expect(gpuCard).toBeVisible()
  await expect(resources).toContainText('46%')
  await expect(resources).toContainText('21.4 / 64.0 GB')

  await measureCompletion({
    scenario: 'machines_test_complete',
    budgetMs: 5000,
    ready: page.getByText('Connection test completed.'),
    testInfo,
    action: async () => {
      await gpuCard.getByRole('button', { name: 'More actions' }).click()
      await page.getByRole('menuitem', { name: 'Connection test' }).click()
    },
  })

  await expect(resources).toContainText('46%')
  await expect(resources).toContainText('21.4 / 64.0 GB')
  await expect(resources).not.toContainText('Pending')
})
