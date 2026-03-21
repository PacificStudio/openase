import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it } from 'vitest'

import ConnectorsSettings from './connectors-settings.svelte'

describe('Connectors settings', () => {
  afterEach(() => {
    cleanup()
  })

  it('documents the deferred management boundary instead of rendering a placeholder', () => {
    const { getByText, queryByText } = render(ConnectorsSettings)

    expect(getByText('Current boundary')).toBeTruthy()
    expect(getByText('Management contract still deferred')).toBeTruthy()
    expect(getByText('/api/v1/webhooks/:connector/:provider')).toBeTruthy()
    expect(queryByText('This settings section has not been classified yet.')).toBeNull()
  })
})
