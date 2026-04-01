import { fireEvent, render } from '@testing-library/svelte'
import { describe, expect, it } from 'vitest'

import {
  createProjectUpdateThread,
  listProjectUpdates,
  makeThreadRecord,
  setupProjectUpdatesPageTest,
} from './project-updates-page.test-support'
import { ProjectUpdatesPage } from '..'

describe('ProjectUpdatesPage creation flow', () => {
  setupProjectUpdatesPageTest()

  it('renders status badges in last-activity order and posts a new update', async () => {
    const migrationWatch = makeThreadRecord({
      id: 'thread-late',
      status: 'at_risk',
      title: 'Migration watch',
      body_markdown: 'Database cleanup is running late.',
      created_by: 'user:ops',
      created_at: '2026-04-01T09:30:00Z',
      updated_at: '2026-04-01T09:35:00Z',
      last_activity_at: '2026-04-01T10:00:00Z',
    })
    const sprintRollout = makeThreadRecord()
    const hotfixHold = makeThreadRecord({
      id: 'thread-new',
      status: 'off_track',
      title: 'Hotfix hold',
      body_markdown: 'Release paused pending rollback validation.',
      created_by: 'user:ops',
      created_at: '2026-04-01T10:31:00Z',
      updated_at: '2026-04-01T10:31:00Z',
      last_activity_at: '2026-04-01T10:31:00Z',
    })

    listProjectUpdates
      .mockResolvedValueOnce({ threads: [sprintRollout, migrationWatch] })
      .mockResolvedValueOnce({ threads: [hotfixHold, migrationWatch, sprintRollout] })
    createProjectUpdateThread.mockResolvedValue({ thread: { id: 'thread-new' } })

    const { findByText, getByPlaceholderText, getAllByRole, getByRole } = render(ProjectUpdatesPage)

    expect(await findByText('Migration watch')).toBeTruthy()
    expect(await findByText('At risk', { selector: 'span' })).toBeTruthy()
    expect(await findByText('On track', { selector: 'span' })).toBeTruthy()
    expect(
      getAllByRole('heading', { level: 2 })
        .map((node) => node.textContent)
        .filter((text) => text !== 'Post a project update'),
    ).toEqual(['Migration watch', 'Sprint 2 rollout'])

    await fireEvent.change(getByRole('combobox', { name: 'New update status' }), {
      target: { value: 'off_track' },
    })
    await fireEvent.input(getByPlaceholderText('Sprint 2 rollout'), {
      target: { value: 'Hotfix hold' },
    })
    await fireEvent.input(
      getByPlaceholderText('Summarize the latest delivery signal, risks, and next checkpoint.'),
      { target: { value: 'Release paused pending rollback validation.' } },
    )
    await fireEvent.click(getByRole('button', { name: 'Post update' }))

    expect(createProjectUpdateThread).toHaveBeenCalledWith('project-1', {
      status: 'off_track',
      title: 'Hotfix hold',
      body: 'Release paused pending rollback validation.',
    })
    expect(await findByText('Update posted.')).toBeTruthy()
    expect(await findByText('Hotfix hold')).toBeTruthy()
    expect(await findByText('Off track', { selector: 'span' })).toBeTruthy()
  })
})
