import { measureCompletion, measureFeedback, measureNavigation } from './perf'
import { expect, test } from './fixtures'

test('agents providers and registration remain responsive', async ({
  page,
  projectPath,
}, testInfo) => {
  await measureNavigation({
    page,
    scenario: 'agent_settings_page_ready',
    budgetMs: 800,
    ready: page.getByRole('button', { name: 'Configure' }).first(),
    testInfo,
    action: async () => {
      await page.goto(`${projectPath('settings')}#agents`)
    },
  })

  await expect(page).toHaveURL(/\/settings#agents$/)
  await expect(page.getByRole('button', { name: 'Configure' }).first()).toBeVisible()

  await measureFeedback({
    scenario: 'provider_config_drawer_open',
    budgetMs: 150,
    ready: page.getByTestId('provider-config-sheet'),
    testInfo,
    action: async () => {
      await page.getByRole('button', { name: 'Configure' }).first().click()
    },
  })

  await page.locator('#provider-name').fill('Codex Primary Updated')

  await measureCompletion({
    scenario: 'provider_config_save_complete',
    budgetMs: 1500,
    ready: page.getByText('Provider updated.'),
    testInfo,
    action: async () => {
      await page.getByRole('button', { name: 'Save changes' }).click({ noWaitAfter: true })
    },
  })

  await page.getByRole('button', { name: 'Cancel' }).click()
  await expect(page.getByTestId('provider-config-sheet')).not.toBeVisible()

  await page.goto(projectPath('agents'))

  await measureFeedback({
    scenario: 'agent_register_drawer_open',
    budgetMs: 150,
    ready: page.getByRole('heading', { name: 'Register agent' }),
    testInfo,
    action: async () => {
      await page.getByRole('button', { name: 'Register Agent' }).click()
    },
  })

  await page.locator('#agent-name').fill('e2e-agent')

  await measureCompletion({
    scenario: 'agent_register_complete',
    budgetMs: 1500,
    ready: page.getByText('Registered e2e-agent.'),
    testInfo,
    action: async () => {
      await page.getByRole('dialog').getByRole('button', { name: 'Register agent' }).click()
    },
  })

  await expect(page.getByText('e2e-agent', { exact: true })).toBeVisible()
})
