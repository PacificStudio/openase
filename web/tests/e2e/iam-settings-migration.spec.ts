import { expect, test } from './fixtures'

test('project settings route admins to /admin and org admin surfaces', async ({
  page,
  projectPath,
}) => {
  await page.goto(projectPath('settings'))

  await expect(page.getByRole('heading', { name: 'Settings' })).toBeVisible()

  await page.getByRole('button', { name: 'Security', exact: true }).click()
  await expect(page.getByText('Compatibility notice')).toBeVisible()
  await expect(
    page.getByText('Project-owned credentials, webhook boundaries, and runtime token policies.'),
  ).toBeVisible()
  await expect(page.getByRole('link', { name: 'Open /admin/auth' })).toBeVisible()

  await page.getByRole('button', { name: 'Access', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Access' })).toBeVisible()
  await expect(page.getByText('Disabled-mode project access')).toBeVisible()
  await expect(page.getByText('Current surface')).toBeVisible()

  await page.getByRole('link', { name: 'Open /admin/auth' }).click()
  await expect(page).toHaveURL('/admin/auth')
  await expect(page.getByRole('heading', { name: 'Admin Auth' })).toBeVisible()
  await expect(page.getByText('Instance-level authentication, OIDC rollout')).toBeVisible()
  await expect(page.getByText('OIDC configuration')).toBeVisible()

  await page.goto(projectPath('settings'))
  await page.getByRole('button', { name: 'Access', exact: true }).click()
  await page.getByRole('link', { name: 'Open org admin' }).click()
  await expect(page).toHaveURL('/orgs/org-e2e/admin/members')
  await expect(page.getByRole('heading', { name: 'Organization admin' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Members', exact: true })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Invitations', exact: true })).toBeVisible()
})
