import { expect } from '@playwright/test'

import { test } from '../fixtures'
import { projectPageMobilePolicies, isResponsiveRoutePolicy } from './policies.js'
import {
  assertDescriptorsDoNotOverlap,
  assertDescriptorsReachable,
  assertNoUnexpectedHorizontalScroll,
  buildPolicyUrl,
  policyAppliesToProject,
} from './helpers'

for (const policy of projectPageMobilePolicies.filter(isResponsiveRoutePolicy)) {
  test(`${policy.title} mobile smoke keeps key controls reachable`, async ({
    page,
    projectPath,
  }, testInfo) => {
    test.skip(
      !policyAppliesToProject(policy, testInfo.project.name),
      `${policy.routeId} is not required on ${testInfo.project.name}`,
    )

    if (!policy.smoke) {
      throw new Error(`Missing smoke config for ${policy.routeId}`)
    }

    await page.goto(buildPolicyUrl(projectPath, policy.routeId, policy.interaction?.hash))

    await expect(
      page.getByRole('heading', { name: policy.smoke.heading, exact: true }),
    ).toBeVisible()
    await assertNoUnexpectedHorizontalScroll(page)
    await assertDescriptorsReachable(page, policy.smoke.criticalControls)
    await assertDescriptorsDoNotOverlap(page, policy.smoke.criticalControls)
  })
}
