import { expect } from '@playwright/test'

import { test } from '../fixtures'
import { gotoProjectRoute, openSearchPalette } from './helpers'

test('mobile search can open the ticket drawer from the top bar', async ({ page }, testInfo) => {
  test.skip(
    testInfo.project.name === 'tablet-1024x768-landscape',
    'Covered by the other tablet project and slower in landscape.',
  )

  await gotoProjectRoute(page, '/orgs/org-e2e/projects/project-e2e')
  const searchInput = await openSearchPalette(page)
  await searchInput.fill('ASE-101')

  await expect(page.getByRole('group', { name: 'Tickets' })).toBeVisible()
  await page.getByRole('option', { name: /ASE-101 Improve machine management UX/i }).click()

  const drawer = page.getByRole('dialog', { name: 'ASE-101' })
  await expect(drawer).toBeVisible()
  await expect(drawer.getByText('Improve machine management UX')).toBeVisible()
})

test('mobile top bar can still open the new ticket dialog', async ({ page }, testInfo) => {
  test.skip(
    testInfo.project.name === 'tablet-1024x768-landscape',
    'Covered by the other tablet project and slower in landscape.',
  )

  await gotoProjectRoute(page, '/orgs/org-e2e/projects/project-e2e/tickets')

  const trigger = page.getByTestId('topbar-new-ticket-button')
  await expect(trigger).toBeVisible()
  await trigger.click()

  const dialog = page.getByRole('dialog', { name: 'Create Ticket' })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByLabel('Title')).toBeVisible()
  await expect(dialog.getByLabel('Description')).toBeVisible()
})
