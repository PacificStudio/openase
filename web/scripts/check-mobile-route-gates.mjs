import { readdir } from 'node:fs/promises'
import path from 'node:path'

import {
  mobileInteractionKinds,
  mobileSupportModes,
  projectPageMobilePolicies,
  routePatternForRouteId,
} from '../tests/e2e/mobile/policies.js'

const routesRoot = path.resolve('src/routes/(app)/orgs/[orgId]/projects/[projectId]')
const routeFileName = '+page.svelte'

async function main() {
  const routeIds = await collectProjectRouteIds(routesRoot)
  const policyMap = buildPolicyMap(projectPageMobilePolicies)
  const errors = []

  const policyIds = [...policyMap.keys()].sort()
  const sortedRouteIds = [...routeIds].sort()

  for (const routeId of sortedRouteIds) {
    if (!policyMap.has(routeId)) {
      errors.push(`Missing mobile route policy for ${routeId}.`)
    }
  }

  for (const routeId of policyIds) {
    if (!routeIds.has(routeId)) {
      errors.push(`Mobile route policy references unknown route ${routeId}.`)
    }
  }

  for (const policy of projectPageMobilePolicies) {
    validatePolicy(policy, errors)
  }

  if (errors.length > 0) {
    console.error('Mobile regression gate policy check failed:')
    for (const error of errors) {
      console.error(`- ${error}`)
    }
    process.exitCode = 1
    return
  }

  console.log(
    `Mobile regression gate policy check passed for ${sortedRouteIds.length} project routes and ${projectPageMobilePolicies.length} policy entries.`,
  )
}

/**
 * @returns {Promise<Set<string>>}
 */
async function collectProjectRouteIds(root) {
  const routeIds = new Set()
  await walkRouteTree(root, '', routeIds)
  return routeIds
}

/**
 * @param {string} absoluteDir
 * @param {string} relativeDir
 * @param {Set<string>} routeIds
 */
async function walkRouteTree(absoluteDir, relativeDir, routeIds) {
  const entries = await readdir(absoluteDir, { withFileTypes: true })

  for (const entry of entries) {
    if (entry.isDirectory()) {
      const nextRelative = relativeDir ? path.posix.join(relativeDir, entry.name) : entry.name
      await walkRouteTree(path.join(absoluteDir, entry.name), nextRelative, routeIds)
      continue
    }

    if (!entry.isFile() || entry.name !== routeFileName) {
      continue
    }

    routeIds.add(relativeDir === '' ? 'dashboard' : relativeDir)
  }
}

/**
 * @param {readonly Array<{ routeId: string }>} policies
 */
function buildPolicyMap(policies) {
  const policyMap = new Map()
  for (const policy of policies) {
    if (policyMap.has(policy.routeId)) {
      throw new Error(`Duplicate mobile policy route id: ${policy.routeId}`)
    }
    policyMap.set(policy.routeId, policy)
  }
  return policyMap
}

/**
 * @param {{
 *   routeId: string
 *   routePattern: string
 *   support: string
 *   reason?: string
 *   smoke?: { heading: string; criticalControls: Array<{ kind: string; value: string }> }
 *   interaction?: { kind: string }
 * }} policy
 * @param {string[]} errors
 */
function validatePolicy(policy, errors) {
  if (!mobileSupportModes.includes(policy.support)) {
    errors.push(`Route ${policy.routeId} has unsupported support mode ${policy.support}.`)
  }

  const expectedPattern = routePatternForRouteId(policy.routeId)
  if (policy.routePattern !== expectedPattern) {
    errors.push(
      `Route ${policy.routeId} declares routePattern ${policy.routePattern}, expected ${expectedPattern}.`,
    )
  }

  if (policy.support === 'desktop-only') {
    if (!policy.reason || policy.reason.trim() === '') {
      errors.push(`Desktop-only route ${policy.routeId} must include a non-empty reason.`)
    }
    return
  }

  if (!policy.smoke || policy.smoke.heading.trim() === '') {
    errors.push(`Responsive route ${policy.routeId} must declare a smoke heading.`)
  }

  if (!policy.smoke || policy.smoke.criticalControls.length === 0) {
    errors.push(`Responsive route ${policy.routeId} must declare at least one critical control.`)
  }

  for (const [index, control] of (policy.smoke?.criticalControls ?? []).entries()) {
    if (!control.kind || !control.value || control.value.trim() === '') {
      errors.push(`Route ${policy.routeId} has an invalid critical control at index ${index}.`)
    }
  }

  if (!policy.interaction) {
    errors.push(`Responsive route ${policy.routeId} must declare an interaction template.`)
    return
  }

  if (!mobileInteractionKinds.includes(policy.interaction.kind)) {
    errors.push(
      `Route ${policy.routeId} has unsupported interaction kind ${policy.interaction.kind}.`,
    )
  }
}

await main()
