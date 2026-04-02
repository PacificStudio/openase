import { measureCompletion, measureFeedback, measureNavigation } from './perf'
import { expect, test } from './fixtures'

test('skills page supports editing, disabling, and binding a skill', async ({
  page,
  projectPath,
}, testInfo) => {
  await measureNavigation({
    page,
    scenario: 'skills_page_ready',
    budgetMs: 800,
    ready: page.getByRole('heading', { name: 'Skills' }),
    testInfo,
    action: async () => {
      await page.goto(projectPath('skills'))
    },
  })

  await expect(page.getByText('deploy-openase')).toBeVisible()

  await measureFeedback({
    scenario: 'skills_detail_open',
    budgetMs: 400,
    ready: page.getByText('deploy-openase').last(),
    testInfo,
    action: async () => {
      await page.getByText('deploy-openase').first().click()
    },
  })

  const description = page.getByPlaceholder('Description...')
  await description.fill('Build and redeploy OpenASE with rollback checks.')

  const editor = page.locator('[data-testid="skill-editor-page"] textarea').first()
  await editor.fill(
    [
      '# Deploy OpenASE',
      '',
      'Build and redeploy OpenASE locally.',
      '',
      'Verify rollback steps first.',
    ].join('\n'),
  )

  await measureCompletion({
    scenario: 'skills_save_complete',
    budgetMs: 1500,
    ready: page.getByText('Saved deploy-openase.'),
    testInfo,
    action: async () => {
      await page.getByRole('button', { name: 'Save' }).click()
    },
  })

  await measureCompletion({
    scenario: 'skills_disable_complete',
    budgetMs: 800,
    ready: page.getByText('Disabled deploy-openase.'),
    testInfo,
    action: async () => {
      await page.getByTitle('Disable').click()
    },
  })
  await expect(page.getByTitle('Enable')).toBeVisible()

  await measureCompletion({
    scenario: 'skills_bind_complete',
    budgetMs: 800,
    ready: page.getByText('Bound deploy-openase to Coding Workflow.'),
    testInfo,
    action: async () => {
      await page.getByTitle('Bind to Coding Workflow').click()
    },
  })
})
