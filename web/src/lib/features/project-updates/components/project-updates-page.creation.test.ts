import { fireEvent, render, waitFor } from '@testing-library/svelte'
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

  it('renders status labels in last-activity order and posts a new update', async () => {
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

    const { findByText, getByPlaceholderText, getByLabelText } = render(ProjectUpdatesPage)

    expect(await findByText('Migration watch')).toBeTruthy()
    await waitFor(() => {
      expect(document.body.textContent).toContain('At risk')
      expect(document.body.textContent).toContain('On track')
    })

    // The composer defaults to on_track; post an update with default status
    await fireEvent.input(getByPlaceholderText('Post an update...'), {
      target: { value: 'Hotfix hold' },
    })
    await fireEvent.click(getByLabelText('Post update'))

    expect(createProjectUpdateThread).toHaveBeenCalledWith('project-1', {
      status: 'on_track',
      title: 'Hotfix hold',
      body: 'Hotfix hold',
    })
    expect(await findByText('Update posted.')).toBeTruthy()
    expect(await findByText('Hotfix hold')).toBeTruthy()
  })
})
