import { measureFeedback, measureNavigation } from './perf'
import { expect, test } from './fixtures'

test.describe('workflow editor layout', () => {
  test('page loads and shows editor toolbar', async ({ page, projectPath }, testInfo) => {
    await measureNavigation({
      page,
      scenario: 'workflow_editor_page_ready',
      budgetMs: 800,
      ready: page.getByRole('button', { name: 'Validate' }),
      testInfo,
      action: async () => {
        await page.goto(projectPath('workflows'))
      },
    })

    // Toolbar buttons are visible
    await expect(page.getByRole('button', { name: 'Validate' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Save' })).toBeVisible()
    await expect(page.getByTitle('Workflow settings')).toBeVisible()

    // Workflow list is visible
    await expect(page.locator('.w-52')).toBeVisible()
  })

  test('workflow list collapses and expands', async ({ page, projectPath }, testInfo) => {
    await measureNavigation({
      page,
      scenario: 'workflow_editor_list_toggle_ready',
      budgetMs: 800,
      ready: page.getByRole('button', { name: 'Validate' }),
      testInfo,
      action: async () => {
        await page.goto(projectPath('workflows'))
      },
    })

    // List is visible initially
    const listContainer = page.locator('.w-52')
    await expect(listContainer).toBeVisible()

    // Collapse the list
    await page.getByTitle('Hide workflow list').click()
    await expect(listContainer).not.toBeVisible()

    // Expand the list
    await page.getByTitle('Show workflow list').click()
    await expect(listContainer).toBeVisible()
  })

  test('settings sheet opens and closes', async ({ page, projectPath }, testInfo) => {
    await measureNavigation({
      page,
      scenario: 'workflow_editor_settings_ready',
      budgetMs: 800,
      ready: page.getByRole('button', { name: 'Validate' }),
      testInfo,
      action: async () => {
        await page.goto(projectPath('workflows'))
      },
    })

    // Open the settings sheet
    await measureFeedback({
      scenario: 'workflow_settings_sheet_open',
      budgetMs: 300,
      ready: page.getByLabel('Workflow Name'),
      testInfo,
      action: async () => {
        await page.getByTitle('Workflow settings').click()
      },
    })

    // Sheet should show form fields
    await expect(page.getByLabel('Workflow Name')).toBeVisible()
    await expect(page.getByText('Bound Agent')).toBeVisible()
    await expect(page.getByText('Pickup Statuses')).toBeVisible()
    await expect(page.getByText('Finish Statuses')).toBeVisible()

    // Close the sheet
    await page.getByRole('button', { name: 'Close' }).click()
    await expect(page.getByLabel('Workflow Name')).not.toBeVisible()
  })

  test('AI drawer opens and has a drag handle', async ({ page, projectPath }, testInfo) => {
    await measureNavigation({
      page,
      scenario: 'workflow_editor_ai_drawer_ready',
      budgetMs: 800,
      ready: page.getByRole('button', { name: 'Validate' }),
      testInfo,
      action: async () => {
        await page.goto(projectPath('workflows'))
      },
    })

    // The drag handle (horizontal separator with cursor-row-resize) should not be visible initially
    const dragHandle = page.locator('[role="separator"].cursor-col-resize')
    await expect(dragHandle).not.toBeVisible()

    // Open the AI drawer
    await measureFeedback({
      scenario: 'workflow_ai_drawer_open',
      budgetMs: 300,
      ready: dragHandle,
      testInfo,
      action: async () => {
        await page.getByRole('button', { name: 'AI', exact: true }).click()
      },
    })

    // Drag handle should be visible
    await expect(dragHandle).toBeVisible()

    // Close the AI drawer
    await page.getByRole('button', { name: 'AI', exact: true }).click()
    await expect(dragHandle).not.toBeVisible()
  })

  test('skill pills render and can be toggled', async ({ page, projectPath }, testInfo) => {
    await measureNavigation({
      page,
      scenario: 'workflow_editor_skills_ready',
      budgetMs: 800,
      ready: page.getByRole('button', { name: 'Validate' }),
      testInfo,
      action: async () => {
        await page.goto(projectPath('workflows'))
      },
    })

    const skillsButton = page.locator('button').filter({ hasText: 'Skills' }).last()
    await expect(skillsButton).toBeVisible()
    await skillsButton.click()

    const commitRow = () =>
      page
        .getByRole('button', { name: /^commit\b/i })
        .filter({ has: page.getByTitle('Unbind skill') })
        .first()
    const deployRow = () =>
      page
        .getByRole('button', { name: /^deploy-openase\b/i })
        .filter({ has: page.getByTitle('Bind skill') })
        .first()

    await expect(commitRow()).toBeVisible()
    await expect(deployRow()).toBeVisible()

    // Bind the unbound skill
    await measureFeedback({
      scenario: 'workflow_skill_bind',
      budgetMs: 500,
      ready: page.getByText('Bound deploy-openase.'),
      testInfo,
      action: async () => {
        await deployRow().getByTitle('Bind skill').click()
      },
    })

    const commitToggle = page
      .getByRole('button', { name: /^commit\b/i })
      .filter({ has: page.getByTitle('Unbind skill') })
      .first()
      .getByTitle('Unbind skill')

    if (!(await commitToggle.isVisible())) {
      await skillsButton.click()
    }

    await expect(commitToggle).toBeVisible()

    // Unbind the bound skill
    await measureFeedback({
      scenario: 'workflow_skill_unbind',
      budgetMs: 500,
      ready: page.getByText('Unbound commit.'),
      testInfo,
      action: async () => {
        await commitToggle.click()
      },
    })
  })

  test('harness editor shows file path and copy button', async ({
    page,
    projectPath,
  }, testInfo) => {
    await measureNavigation({
      page,
      scenario: 'workflow_editor_harness_ready',
      budgetMs: 800,
      ready: page.getByRole('button', { name: 'Validate' }),
      testInfo,
      action: async () => {
        await page.goto(projectPath('workflows'))
      },
    })

    // The editor should display the harness file path (truncated)
    await expect(page.locator('[title*="harness"]')).toBeVisible()

    // Copy button should be visible
    await expect(page.getByText('Copy')).toBeVisible()
  })
})
