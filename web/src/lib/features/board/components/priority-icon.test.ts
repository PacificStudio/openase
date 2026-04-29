import { render } from '@testing-library/svelte'
import { describe, expect, it } from 'vitest'

import { i18nStore } from '$lib/i18n/store.svelte'
import PriorityIcon from './priority-icon.svelte'

describe('PriorityIcon', () => {
  it('renders no-priority as a horizontal minus glyph', () => {
    const { container, getByLabelText } = render(PriorityIcon, {
      props: { priority: '' },
    })

    expect(getByLabelText('Priority: unset')).toBeTruthy()
    expect(container.querySelectorAll('rect')).toHaveLength(1)
  })

  it('renders stacked bars for ranked priorities', () => {
    const { container, getByLabelText } = render(PriorityIcon, {
      props: { priority: 'high' },
    })

    expect(getByLabelText('Priority: high')).toBeTruthy()
    expect(container.querySelectorAll('rect')).toHaveLength(3)
  })

  it('localizes the aria label with the active locale', () => {
    i18nStore.setLocale('zh')
    const { getByLabelText } = render(PriorityIcon, {
      props: { priority: 'high' },
    })

    expect(getByLabelText('优先级: 高')).toBeTruthy()
    i18nStore.setLocale('en')
  })
})
