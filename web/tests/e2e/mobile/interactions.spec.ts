import { expect } from '@playwright/test'

import { test } from '../fixtures'
import { projectPageMobilePolicies, isResponsiveRoutePolicy } from './policies.js'
import {
  buildPolicyUrl,
  gotoProjectRoute,
  locatorFromDescriptor,
  openAndAssertSurface,
  policyAppliesToProject,
} from './helpers'

for (const policy of projectPageMobilePolicies.filter(isResponsiveRoutePolicy)) {
  test(`${policy.title} mobile interaction flow remains usable`, async ({
    page,
    projectPath,
  }, testInfo) => {
    test.skip(
      !policyAppliesToProject(policy, testInfo.project.name),
      `${policy.routeId} is not required on ${testInfo.project.name}`,
    )

    const interaction = policy.interaction
    if (!interaction) {
      throw new Error(`Missing interaction config for ${policy.routeId}`)
    }

    await gotoProjectRoute(
      page,
      buildPolicyUrl(projectPath, interaction.route ?? policy.routeId, interaction.hash),
    )

    switch (interaction.kind) {
      case 'ticket-drawer': {
        if (!interaction.opener) {
          throw new Error('Ticket drawer policy is incomplete.')
        }

        await locatorFromDescriptor(page, interaction.opener).first().click({ noWaitAfter: true })
        const drawer = page.getByRole('dialog')
        await expect(drawer).toBeVisible()
        await expect(drawer).toHaveAccessibleName('ASE-101', {
          timeout: 10_000,
        })
        await expect(drawer.getByText('Improve machine management UX')).toBeVisible()
        break
      }

      case 'settings-repositories': {
        if (
          !interaction.opener ||
          !interaction.ready ||
          !interaction.input ||
          !interaction.submit
        ) {
          throw new Error('Settings repositories policy is incomplete.')
        }

        await openAndAssertSurface(page, interaction.opener, interaction.ready)
        await locatorFromDescriptor(page, interaction.input).fill('trunk-mobile')
        await locatorFromDescriptor(page, interaction.submit).click({ noWaitAfter: true })
        await expect(
          page.getByText(interaction.expectedText ?? 'Repository updated.'),
        ).toBeVisible()
        await expect(page.getByTestId('repository-card-repo-todo')).toContainText('trunk-mobile')
        break
      }

      case 'activity-filters': {
        if (
          !interaction.input ||
          !interaction.searchValue ||
          !interaction.filterOption ||
          !interaction.expectedButton
        ) {
          throw new Error('Activity filters policy is incomplete.')
        }

        await locatorFromDescriptor(page, interaction.input).fill(interaction.searchValue)
        await expect(
          page.getByText(interaction.expectedText ?? 'coding-main started work.'),
        ).toBeVisible()
        await page.getByRole('button', { name: 'All' }).click()
        await page.getByRole('option', { name: interaction.filterOption }).click()
        await expect(page.getByRole('button', { name: interaction.expectedButton })).toBeVisible()
        await expect(
          page.getByText(interaction.expectedText ?? 'coding-main started work.'),
        ).toBeVisible()
        break
      }

      case 'updates-composer': {
        if (!interaction.input || !interaction.submit || !interaction.expectedText) {
          throw new Error('Updates composer policy is incomplete.')
        }

        await locatorFromDescriptor(page, interaction.input).fill(interaction.expectedText)
        await locatorFromDescriptor(page, interaction.submit).click({ noWaitAfter: true })
        await expect(page.getByText(interaction.expectedText)).toBeVisible()
        break
      }

      case 'machines-sheet': {
        if (
          !interaction.opener ||
          !interaction.ready ||
          !interaction.input ||
          !interaction.submit
        ) {
          throw new Error('Machines sheet policy is incomplete.')
        }

        await openAndAssertSurface(page, interaction.opener, interaction.ready)
        await locatorFromDescriptor(page, interaction.input).fill(
          'Updated from mobile regression coverage',
        )
        await locatorFromDescriptor(page, interaction.submit).click({ noWaitAfter: true })
        await expect(page.getByText(interaction.expectedText ?? 'Machine updated.')).toBeVisible()
        break
      }

      case 'agents-register': {
        if (
          !interaction.opener ||
          !interaction.ready ||
          !interaction.input ||
          !interaction.submit
        ) {
          throw new Error('Agents register policy is incomplete.')
        }

        const agentName = `mobile-agent-${testInfo.project.name}-${testInfo.retry}`
        await openAndAssertSurface(page, interaction.opener, interaction.ready)
        const registrationSheet = page.getByRole('dialog', { name: 'Register agent' })
        await expect(registrationSheet).toBeVisible()
        await registrationSheet.getByLabel(interaction.input.value).fill(agentName)
        await registrationSheet
          .getByRole('button', { name: interaction.submit.value, exact: true })
          .click({ noWaitAfter: true })
        await expect(page.getByText(`Registered ${agentName}.`)).toBeVisible()
        await expect(
          page.getByRole('button', { name: agentName, exact: true }).first(),
        ).toBeVisible()
        break
      }

      default: {
        const exhaustive: never = interaction.kind
        throw new Error(`Unhandled interaction kind: ${exhaustive}`)
      }
    }
  })
}
