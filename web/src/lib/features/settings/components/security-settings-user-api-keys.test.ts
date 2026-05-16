import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

import { authStore } from '$lib/stores/auth.svelte'
import { appStore } from '$lib/stores/app.svelte'
import { configuredSecurity, currentOrg, currentProject } from './security-settings.test-helpers'
import SecuritySettingsUserAPIKeys from './security-settings-user-api-keys.svelte'

const {
  createProjectUserAPIKey,
  deleteProjectUserAPIKey,
  disableProjectUserAPIKey,
  listProjectUserAPIKeys,
  rotateProjectUserAPIKey,
} = vi.hoisted(() => ({
  createProjectUserAPIKey: vi.fn(),
  deleteProjectUserAPIKey: vi.fn(),
  disableProjectUserAPIKey: vi.fn(),
  listProjectUserAPIKeys: vi.fn(),
  rotateProjectUserAPIKey: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  createProjectUserAPIKey,
  deleteProjectUserAPIKey,
  disableProjectUserAPIKey,
  listProjectUserAPIKeys,
  rotateProjectUserAPIKey,
}))

describe('SecuritySettingsUserAPIKeys', () => {
  beforeAll(() => {
    HTMLElement.prototype.scrollIntoView ??= vi.fn()
    Object.assign(globalThis.navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    })
  })

  afterEach(() => {
    cleanup()
    authStore.clear()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('shows plaintext API key once after creating a project user API key', async () => {
    authStore.hydrate({ authenticated: true })
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    listProjectUserAPIKeys.mockResolvedValue({ api_keys: [] })
    createProjectUserAPIKey.mockResolvedValue({
      api_key: {
        id: 'key-1',
        name: 'Buildkite',
        token_hint: 'ase_pat_abc...1234',
        scopes: ['tickets.list'],
        status: 'active',
        expires_at: null,
        last_used_at: null,
        created_at: '2026-04-16T12:00:00Z',
        disabled_at: null,
        revoked_at: null,
      },
      plain_text_token: 'ase_pat_plain_text_token',
    })

    const { findByLabelText, findAllByRole, findByText, queryByText } = render(
      SecuritySettingsUserAPIKeys,
      { security: configuredSecurity() },
    )

    await fireEvent.click((await findAllByRole('button', { name: 'Create key' }))[0])
    await fireEvent.input(await findByLabelText('Name'), { target: { value: 'Buildkite' } })
    await fireEvent.click(await findByText('tickets'))
    await fireEvent.click(await findByText('list'))
    await fireEvent.click((await findAllByRole('button', { name: 'Create key' }))[1])

    await waitFor(() => {
      expect(createProjectUserAPIKey).toHaveBeenCalledWith(appStore.currentProject?.id, {
        name: 'Buildkite',
        scopes: ['tickets.list'],
        expires_at: null,
      })
    })

    expect(await findByText('Copy this API key now')).toBeTruthy()
    expect(await findByText('ase_pat_plain_text_token')).toBeTruthy()
    await fireEvent.click(await findByText('Done'))
    await waitFor(() => {
      expect(queryByText('ase_pat_plain_text_token')).toBeNull()
    })
  })

  it('uses the loaded security project id for API calls', async () => {
    authStore.hydrate({ authenticated: true })
    appStore.currentOrg = currentOrg()
    appStore.currentProject = { ...currentProject(), id: 'atlas' }
    listProjectUserAPIKeys.mockResolvedValue({ api_keys: [] })

    render(SecuritySettingsUserAPIKeys, { security: configuredSecurity() })

    await waitFor(() => {
      expect(listProjectUserAPIKeys).toHaveBeenCalledWith(configuredSecurity().project_id)
    })
  })

  it('shows a sign-in message instead of calling the API without a human session', async () => {
    authStore.hydrate({
      authenticated: false,
      loginRequired: true,
      authCapabilities: { availableAuthMethods: ['oidc'], currentAuthMethod: 'oidc' },
    })
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()

    const { findByText } = render(SecuritySettingsUserAPIKeys, { security: configuredSecurity() })

    expect(
      await findByText(
        'Sign in to list, create, rotate, disable, or delete user API keys for this project.',
      ),
    ).toBeTruthy()
    expect(listProjectUserAPIKeys).not.toHaveBeenCalled()
  })
})
