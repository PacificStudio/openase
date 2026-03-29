import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { TicketDetail } from '../types'
import TicketRepoScopeCard from './ticket-repo-scope-card.svelte'

const scope: TicketDetail['repoScopes'][number] = {
  id: 'scope-1',
  repoId: 'repo-1',
  repoName: 'openase-web',
  branchName: 'fix/openase-329-read-first',
  prUrl: 'https://github.com/GrandCX/openase/pull/329',
  prStatus: 'open',
  ciStatus: 'pass',
  isPrimaryScope: true,
}

describe('TicketRepoScopeCard', () => {
  afterEach(() => {
    cleanup()
  })

  it('renders a compact summary by default and hides edit controls until requested', () => {
    const { getByRole, getByText, queryByLabelText, queryByText } = render(TicketRepoScopeCard, {
      props: { scope },
    })

    expect(getByText('openase-web')).toBeTruthy()
    expect(getByText('fix/openase-329-read-first')).toBeTruthy()
    expect(getByText('Open')).toBeTruthy()
    expect(getByText('Passing')).toBeTruthy()
    expect(getByText('Primary')).toBeTruthy()
    expect(getByRole('button', { name: 'Edit openase-web scope' })).toBeTruthy()
    expect(queryByLabelText('Branch')).toBeNull()
    expect(queryByText('Save scope')).toBeNull()
  })

  it('opens edit controls only after the explicit edit action and saves the current scope draft', async () => {
    const onSave = vi.fn()
    const { getByDisplayValue, getByLabelText, getByRole, getByText } = render(
      TicketRepoScopeCard,
      {
        props: { scope, onSave },
      },
    )

    await fireEvent.click(getByRole('button', { name: 'Edit openase-web scope' }))

    expect(getByLabelText('Branch')).toBeTruthy()
    expect(getByDisplayValue('fix/openase-329-read-first')).toBeTruthy()
    await fireEvent.click(getByText('Save scope'))

    expect(onSave).toHaveBeenCalledWith('scope-1', {
      branchName: 'fix/openase-329-read-first',
      pullRequestUrl: 'https://github.com/GrandCX/openase/pull/329',
      prStatus: 'open',
      ciStatus: 'pass',
      isPrimaryScope: true,
    })
  })
})
