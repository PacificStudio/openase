import type { Page } from '@playwright/test'
import { expect, test } from './fixtures'

async function openGlobalSearch(page: Page) {
  await page.getByRole('button', { name: /Search/i }).click()
  await expect(
    page.getByPlaceholder('Search pages, tickets, workflows, agents, and commands...'),
  ).toBeVisible()
}

async function searchFor(page: Page, query: string) {
  const input = page.getByPlaceholder('Search pages, tickets, workflows, agents, and commands...')
  await input.fill(query)
  return input
}

test('global search finds a ticket and opens its detail drawer', async ({ page }) => {
  await page.goto('/orgs/org-e2e/projects/project-e2e')

  await openGlobalSearch(page)
  await searchFor(page, 'ASE-101')

  await expect(page.getByRole('group', { name: 'Tickets' })).toBeVisible()
  await expect(
    page.getByRole('option', { name: /ASE-101 Improve machine management UX/i }),
  ).toBeVisible()

  await page.getByRole('option', { name: /ASE-101 Improve machine management UX/i }).click()

  const ticketDrawer = page.getByRole('dialog', { name: /ASE-101/i })

  await expect(ticketDrawer).toBeVisible()
  await expect(ticketDrawer.getByText('Improve machine management UX')).toBeVisible()
})

test('global search finds workflows and navigates to the workflows page', async ({ page }) => {
  await page.goto('/orgs/org-e2e/projects/project-e2e')

  await openGlobalSearch(page)
  await searchFor(page, 'Coding Workflow')

  const workflowGroup = page.getByRole('group', { name: 'Workflows' })
  const workflowOption = workflowGroup.getByRole('option', { name: /Coding Workflow/i })

  await expect(workflowGroup).toBeVisible()
  await expect(workflowOption).toBeVisible()

  await workflowOption.click()

  await expect(page).toHaveURL(/\/workflows$/)
  await expect(page.getByRole('heading', { name: 'Workflows' })).toBeVisible()
  await expect(page.getByRole('button', { name: /Coding Workflow/i }).first()).toBeVisible()
})

test('global search supports agent and command results from the top-bar entrypoint', async ({
  page,
}) => {
  await page.goto('/orgs/org-e2e/projects/project-e2e')

  await openGlobalSearch(page)

  await searchFor(page, 'coding-main')
  const agentGroup = page.getByRole('group', { name: 'Agents' })
  const agentOption = agentGroup.getByRole('option', { name: /coding-main/i })

  await expect(agentGroup).toBeVisible()
  await expect(agentOption).toBeVisible()

  await agentOption.click()

  await expect(page).toHaveURL(/\/agents$/)
  await expect(page.getByRole('heading', { name: 'Agents' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'coding-main' }).first()).toBeVisible()

  await openGlobalSearch(page)
  await searchFor(page, 'Ask AI')

  const commandGroup = page.getByRole('group', { name: 'Commands' })
  const askAiOption = commandGroup.getByRole('option', { name: /Ask AI/i })

  await expect(commandGroup).toBeVisible()
  await expect(askAiOption).toBeVisible()

  await askAiOption.click()

  await expect(page.getByPlaceholder('Ask anything about this project…')).toBeVisible()
})
