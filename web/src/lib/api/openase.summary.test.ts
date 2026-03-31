import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, patch, put, del } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
}))

vi.mock('./client', () => ({
  api: {
    get,
    post,
    patch,
    put,
    delete: del,
  },
}))

import { getOrganizationSummary, getWorkspaceSummary } from './openase'

describe('workspace summary helpers', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('calls the workspace summary endpoint with request options', async () => {
    const signal = new AbortController().signal
    get.mockResolvedValue({ workspace: {}, organizations: [] })

    await getWorkspaceSummary({ signal })

    expect(get).toHaveBeenCalledWith('/api/v1/workspace/summary', { signal })
  })

  it('calls the organization summary endpoint with the org id', async () => {
    const signal = new AbortController().signal
    get.mockResolvedValue({ organization: {}, projects: [] })

    await getOrganizationSummary('org-123', { signal })

    expect(get).toHaveBeenCalledWith('/api/v1/orgs/org-123/summary', { signal })
  })
})
