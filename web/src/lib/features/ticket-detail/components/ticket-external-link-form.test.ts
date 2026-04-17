import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import TicketExternalLinkForm from './ticket-external-link-form.svelte'

describe('TicketExternalLinkForm', () => {
  afterEach(() => {
    cleanup()
  })

  it('renders type as optional text input and omits relation controls', () => {
    const { container, getByLabelText, getByPlaceholderText, queryByText } = render(
      TicketExternalLinkForm,
    )

    expect(getByLabelText(/Type/)).toBeTruthy()
    expect(getByPlaceholderText('Type')).toBeTruthy()
    expect(queryByText('Relation')).toBeNull()
    expect(container.textContent).toContain('None')
    expect(container.textContent).not.toContain('=>')
  })

  it('submits url and external id without requiring a type', async () => {
    const onCreate = vi.fn(async () => true)
    const { getByLabelText, getByRole } = render(TicketExternalLinkForm, {
      props: { onCreate },
    })

    await fireEvent.input(getByLabelText(/URL/), {
      target: { value: 'https://docs.example.com/spec' },
    })
    await fireEvent.input(getByLabelText(/External ID/), {
      target: { value: 'SPEC-1' },
    })
    await fireEvent.click(getByRole('button', { name: 'Add' }))

    expect(onCreate).toHaveBeenCalledWith({
      type: '',
      url: 'https://docs.example.com/spec',
      externalId: 'SPEC-1',
      title: '',
      status: '',
    })
  })
})
