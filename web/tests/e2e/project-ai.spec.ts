import type { Page } from '@playwright/test'
import { expect, test } from './fixtures'

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
}) => {
  await page.goto(projectPath('workflows'))
  await page.waitForTimeout(2000)
  await expect(page.getByRole('heading', { name: 'Workflows' })).toBeVisible({ timeout: 10_000 })

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
}) => {
  await page.goto(projectPath('workflows'))
  await expect(page.getByRole('heading', { name: 'Workflows' })).toBeVisible({ timeout: 10_000 })

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

test('harness ai provider picker matches the available provider catalog', async ({
  page,
  projectPath,
}) => {
  await page.goto(projectPath('workflows'))
  await page.waitForTimeout(2000)
  await expect(page.getByRole('heading', { name: 'Workflows' })).toBeVisible({ timeout: 10_000 })

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
