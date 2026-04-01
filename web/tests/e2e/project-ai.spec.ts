import { expect, test } from './fixtures'

const expectedProjectAIProviders = [
  {
    detail: 'Claude Code · claude-code-cli',
  },
  {
    detail: 'Fake Codex Validation Provider · codex-app-server',
  },
  {
    detail: 'Gemini CLI · gemini-cli',
  },
  {
    detail: 'OpenAI Codex · codex-app-server',
  },
]

test('project ai provider picker matches the available provider catalog', async ({
  page,
  projectPath,
}) => {
  await page.goto(projectPath('workflows'))
  await page.waitForTimeout(2000)
  await expect(page.getByRole('heading', { name: 'Workflows' })).toBeVisible({ timeout: 10_000 })

  await page.getByRole('button', { name: 'Ask AI' }).click()
  await expect(page.getByLabel('Chat model')).toBeVisible({ timeout: 10_000 })

  await expect(page.getByText('No chat provider available.')).not.toBeVisible()

  const chatModelTrigger = page.getByLabel('Chat model')
  await expect(chatModelTrigger).toBeVisible()

  await chatModelTrigger.click()

  for (const provider of expectedProjectAIProviders) {
    await expect(page.getByText(provider.detail, { exact: true })).toBeVisible()
  }
})

test('harness ai provider picker matches the available provider catalog', async ({
  page,
  projectPath,
}) => {
  await page.goto(projectPath('workflows'))
  await page.waitForTimeout(2000)
  await expect(page.getByRole('heading', { name: 'Workflows' })).toBeVisible({ timeout: 10_000 })

  await page.getByRole('button', { name: 'AI', exact: true }).click()
  await expect(page.getByText('Harness AI')).toBeVisible({ timeout: 10_000 })

  await expect(page.getByText('No chat provider available.')).not.toBeVisible()

  const chatModelTrigger = page.getByLabel('Chat model')
  await expect(chatModelTrigger).toBeVisible()

  await chatModelTrigger.click()

  for (const provider of expectedProjectAIProviders) {
    await expect(page.getByText(provider.detail, { exact: true })).toBeVisible()
  }
})
