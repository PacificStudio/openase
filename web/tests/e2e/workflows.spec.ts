import { measureCompletion, measureFeedback, measureNavigation } from './perf'
import { expect, test } from './fixtures'

test('workflow creation dialog stays responsive', async ({ page, projectPath }, testInfo) => {
  const workflowName = `Workflow E2E ${testInfo.parallelIndex}-${testInfo.retry}`

  await measureNavigation({
    page,
    scenario: 'workflows_page_ready',
    budgetMs: 800,
    ready: page.getByRole('heading', { name: 'Workflows' }),
    testInfo,
    action: async () => {
      await page.goto(projectPath('workflows'))
    },
  })

  await measureFeedback({
    scenario: 'workflow_create_dialog_open',
    budgetMs: 150,
    ready: page.getByRole('heading', { name: 'Create Workflow' }),
    testInfo,
    action: async () => {
      await page.getByRole('button', { name: 'New Workflow' }).click()
    },
  })

  const dialog = page.getByRole('dialog', { name: 'Create Workflow' })
  await page.locator('#workflow-create-name').fill(workflowName)
  await dialog
    .getByRole('group', { name: 'Finish Statuses' })
    .getByRole('button', { name: 'Done' })
    .click()

  await measureCompletion({
    scenario: 'workflow_create_complete',
    budgetMs: 1500,
    ready: page.getByText(workflowName).first(),
    testInfo,
    action: async () => {
      await dialog.getByRole('button', { name: 'Create workflow' }).click()
    },
  })

  await expect(page.getByText(workflowName).first()).toBeVisible()
})

test('scheduled jobs creation remains responsive', async ({ page, projectPath }, testInfo) => {
  await measureNavigation({
    page,
    scenario: 'scheduled_jobs_page_ready',
    budgetMs: 800,
    ready: page.getByRole('heading', { name: 'Scheduled Jobs' }),
    testInfo,
    action: async () => {
      await page.goto(projectPath('scheduled-jobs'))
    },
  })

  await page.getByRole('button', { name: 'New job' }).click()
  await page.getByLabel('Job name').fill('Morning sync')
  await page.getByRole('button', { name: 'Manual input' }).click()
  await page.getByPlaceholder('0 2 * * *').fill('0 9 * * 1-5')
  await page.getByRole('button', { name: 'Ticket template' }).click()
  await page.getByLabel('Title').fill('Run morning sync')
  await page.getByLabel('Created by').fill('playwright')

  await measureCompletion({
    scenario: 'scheduled_job_create_complete',
    budgetMs: 1500,
    ready: page.getByText('Scheduled job created.'),
    testInfo,
    action: async () => {
      await page.getByRole('button', { name: 'Create' }).click()
    },
  })

  await expect(page.getByRole('button', { name: 'Morning sync 0 9 * * 1-5' }).first()).toBeVisible()
})
