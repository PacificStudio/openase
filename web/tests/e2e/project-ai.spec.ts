import type { Page } from '@playwright/test'
import { expect, test } from './fixtures'
import { measureNavigation } from './perf'

const PROJECT_AI_SHORTCUT = 'Control+I'

async function expectProviderCatalog(page: Page) {
  const options = page.getByRole('option')
  await expect(options).toHaveCount(4)

  const optionTexts = await options.allTextContents()
  expect(
    optionTexts.some(
      (text) => text.includes('claude-opus-4-6') && text.includes('claude-code-cli'),
    ),
  ).toBeTruthy()
  expect(
    optionTexts.some((text) => text.includes('gemini-2.5-pro') && text.includes('gemini-cli')),
  ).toBeTruthy()
  expect(optionTexts.filter((text) => text.includes('codex-app-server')).length).toBe(2)
  expect(optionTexts.every((text) => text.includes('Ready'))).toBeTruthy()
}

test('project ai provider picker matches the available provider catalog', async ({
  page,
  projectPath,
}, testInfo) => {
  await measureNavigation({
    page,
    scenario: 'project_ai_workflows_ready',
    budgetMs: 800,
    ready: page.getByRole('button', { name: 'Validate' }),
    testInfo,
    action: async () => {
      await page.goto(projectPath('workflows'))
    },
  })

  await page.getByRole('button', { name: 'Ask AI' }).click()
  await expect(page.getByPlaceholder('Ask anything about this project…')).toBeVisible({
    timeout: 10_000,
  })

  await expect(page.getByText('No chat provider available.')).not.toBeVisible()

  const chatModelTrigger = page.getByLabel('Chat model')
  await expect(chatModelTrigger).toBeVisible()

  await chatModelTrigger.click()
  await expectProviderCatalog(page)
})

test('project ai creates a new conversation and streams the first reply', async ({
  page,
  projectPath,
}, testInfo) => {
  await measureNavigation({
    page,
    scenario: 'project_ai_first_turn_ready',
    budgetMs: 800,
    ready: page.getByRole('button', { name: 'Validate' }),
    testInfo,
    action: async () => {
      await page.goto(projectPath('workflows'))
    },
  })

  await page.getByRole('button', { name: 'Ask AI' }).click()
  await expect(page.getByPlaceholder('Ask anything about this project…')).toBeVisible({
    timeout: 10_000,
  })

  const prompt = page.getByPlaceholder('Ask anything about this project…')
  const sendButton = page.locator('button[aria-label="Send message"]:visible')

  await expect(page.getByText('No chat provider available.')).not.toBeVisible()
  await prompt.fill('What repo is connected?')
  await expect(sendButton).toBeEnabled({ timeout: 10_000 })
  await sendButton.click()

  await expect(
    page.locator('.break-words.whitespace-pre-wrap').filter({ hasText: 'What repo is connected?' }),
  ).toBeVisible()
  await expect(page.getByText('Mock assistant reply for: What repo is connected?')).toBeVisible({
    timeout: 10_000,
  })
  await expect(page.getByText('Thinking...')).not.toBeVisible({ timeout: 10_000 })
})

test('ticket drawer routes AI through ticket-focused Project AI with a complete capsule', async ({
  page,
  projectPath,
}) => {
  await page.goto(projectPath('tickets'))
  await expect(page.getByText('ASE-101')).toBeVisible({ timeout: 10_000 })

  await page.getByText('ASE-101').click()
  const drawer = page.getByRole('dialog', { name: 'ASE-101' })
  await expect(drawer).toBeVisible({ timeout: 10_000 })
  await page.keyboard.press(PROJECT_AI_SHORTCUT)
  await expect(page.getByPlaceholder('Ask anything about this project…')).toBeVisible({
    timeout: 10_000,
  })
  await expect(page.getByRole('heading', { name: 'Project AI' })).toBeVisible()
  await expect(page.getByText('ASE-101 / Improve machine management UX')).toBeVisible({
    timeout: 10_000,
  })
})

test('harness ai provider picker matches the available provider catalog', async ({
  page,
  projectPath,
}, testInfo) => {
  await measureNavigation({
    page,
    scenario: 'harness_ai_workflows_ready',
    budgetMs: 800,
    ready: page.getByRole('button', { name: 'AI', exact: true }),
    testInfo,
    action: async () => {
      await page.goto(projectPath('workflows'))
    },
  })

  await page.getByRole('button', { name: 'AI', exact: true }).click()
  await expect(page.getByPlaceholder('Ask AI to refine this harness…')).toBeVisible({
    timeout: 10_000,
  })

  await expect(page.getByText('No chat provider available.')).not.toBeVisible()

  const chatModelTrigger = page.getByLabel('Chat model')
  await expect(chatModelTrigger).toBeVisible()

  await chatModelTrigger.click()
  await expectProviderCatalog(page)
})
