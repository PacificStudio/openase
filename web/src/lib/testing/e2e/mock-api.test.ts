import { beforeEach, describe, expect, it } from 'vitest'

import { handleMockApi, resetMockState } from './mock-api'

const origin = 'http://127.0.0.1:4173'
const projectID = 'project-e2e'

describe('e2e mock api', () => {
  beforeEach(() => {
    resetMockState()
  })

  it('serves security settings with approval policy diagnostics', async () => {
    const request = new Request(`${origin}/api/v1/projects/${projectID}/security-settings`)
    const response = await handleMockApi(request, new URL(request.url))

    expect(response?.status).toBe(200)
    await expect(response?.json()).resolves.toMatchObject({
      security: {
        project_id: projectID,
        approval_policies: {
          status: 'reserved',
          rules_count: 0,
        },
        deferred: [{ key: 'github-device-flow' }],
      },
    })
  })

  it('keeps github credential mock routes stateful', async () => {
    const importRequest = new Request(
      `${origin}/api/v1/projects/${projectID}/security-settings/github-outbound-credential/import-gh-cli`,
      {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ scope: 'project' }),
      },
    )
    const importResponse = await handleMockApi(importRequest, new URL(importRequest.url))
    expect(importResponse?.status).toBe(200)
    await expect(importResponse?.json()).resolves.toMatchObject({
      security: {
        github: {
          effective: {
            scope: 'project',
            configured: true,
            source: 'gh_cli_import',
          },
          project_override: {
            configured: true,
          },
        },
      },
    })

    const deleteRequest = new Request(
      `${origin}/api/v1/projects/${projectID}/security-settings/github-outbound-credential?scope=project`,
      { method: 'DELETE' },
    )
    const deleteResponse = await handleMockApi(deleteRequest, new URL(deleteRequest.url))
    expect(deleteResponse?.status).toBe(200)
    await expect(deleteResponse?.json()).resolves.toMatchObject({
      security: {
        github: {
          effective: {
            configured: false,
          },
          project_override: {
            configured: false,
          },
        },
      },
    })
  })

  it('matches the real logout 204 contract', async () => {
    const request = new Request(`${origin}/api/v1/auth/logout`, { method: 'POST' })
    const response = await handleMockApi(request, new URL(request.url))

    expect(response?.status).toBe(204)
    await expect(response?.text()).resolves.toBe('')
  })
})
