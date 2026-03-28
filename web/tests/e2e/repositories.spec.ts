import { measureCompletion, measureFeedback, measureNavigation } from './perf'
import { expect, test } from './fixtures'

test('repositories settings use a responsive drawer workflow', async ({
  page,
  projectPath,
}, testInfo) => {
  await measureNavigation({
    page,
    scenario: 'repositories_section_ready',
    budgetMs: 800,
    ready: page.getByTestId('repository-card-repo-todo'),
    testInfo,
    action: async () => {
      await page.goto(`${projectPath('settings')}#repositories`)
    },
  })

  await expect(page).toHaveURL(/\/settings#repositories$/)

  await measureFeedback({
    scenario: 'repositories_drawer_open',
    budgetMs: 150,
    ready: page.getByTestId('repository-editor-sheet'),
    testInfo,
    action: async () => {
      await page.getByTestId('repository-open-repo-todo').click({ noWaitAfter: true })
    },
  })

  await page.getByLabel('Default branch').fill('trunk')

  await measureCompletion({
    scenario: 'repositories_save_complete',
    budgetMs: 1500,
    ready: page.getByText('Repository updated.'),
    testInfo,
    action: async () => {
      await page.getByTestId('repository-save-button').click({ noWaitAfter: true })
    },
  })

  await expect(page.getByTestId('repository-card-repo-todo')).toContainText('trunk')
})
