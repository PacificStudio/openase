import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import WorkflowTemplateGallery from './workflow-template-gallery.svelte'

const { listBuiltinRoles, getBuiltinRole } = vi.hoisted(() => ({
  listBuiltinRoles: vi.fn(),
  getBuiltinRole: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  listBuiltinRoles,
  getBuiltinRole,
}))

describe('WorkflowTemplateGallery', () => {
  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('loads the builtin template list and shows details for multiple templates', async () => {
    listBuiltinRoles.mockResolvedValue({
      roles: [
        {
          slug: 'fullstack-developer',
          name: 'Fullstack Developer',
          summary: 'Builds and ships product features.',
          workflow_type: 'coding',
          content: 'Base coding role content',
          workflow_content: '',
          harness_path: '.openase/harnesses/fullstack-developer.md',
        },
        {
          slug: 'test-engineer',
          name: 'Test Engineer',
          summary: 'Designs regression coverage and test plans.',
          workflow_type: 'test',
          content: 'Base test role content',
          workflow_content: '',
          harness_path: '.openase/harnesses/test-engineer.md',
        },
      ],
    })
    getBuiltinRole.mockImplementation(async (slug: string) => {
      if (slug === 'fullstack-developer') {
        return {
          role: {
            slug,
            name: 'Fullstack Developer',
            summary: 'Builds and ships product features.',
            workflow_type: 'coding',
            content: 'Base coding role content',
            workflow_content:
              'You are a fullstack developer.\nImplement features end to end and ship safely.',
            harness_path: '.openase/harnesses/fullstack-developer.md',
          },
        }
      }

      if (slug === 'test-engineer') {
        return {
          role: {
            slug,
            name: 'Test Engineer',
            summary: 'Designs regression coverage and test plans.',
            workflow_type: 'test',
            content: 'Base test role content',
            workflow_content:
              'You are a test engineer.\nWrite regression plans and verify critical paths.',
            harness_path: '.openase/harnesses/test-engineer.md',
          },
        }
      }

      throw new Error(`unexpected role ${slug}`)
    })

    const { findByText, findByTestId, findByRole, getByText, queryByText } = render(
      WorkflowTemplateGallery,
      {
        open: true,
      },
    )

    expect(await findByTestId('workflow-template-gallery')).toBeTruthy()

    await waitFor(() => {
      expect(listBuiltinRoles).toHaveBeenCalledTimes(1)
    })

    expect(await findByText('Fullstack Developer', { exact: false })).toBeTruthy()
    expect(await findByText('Test Engineer', { exact: false })).toBeTruthy()

    await fireEvent.click(getByText('Fullstack Developer', { exact: false }).closest('button')!)

    await waitFor(() => {
      expect(getBuiltinRole).toHaveBeenCalledWith('fullstack-developer')
    })
    expect(await findByText('Builds and ships product features.')).toBeTruthy()
    expect(
      await findByText(
        (_, element) =>
          element?.textContent ===
          'You are a fullstack developer.\nImplement features end to end and ship safely.',
      ),
    ).toBeTruthy()

    await fireEvent.click(await findByRole('button', { name: 'Back' }))

    await fireEvent.click(getByText('Test Engineer', { exact: false }).closest('button')!)

    await waitFor(() => {
      expect(getBuiltinRole).toHaveBeenCalledWith('test-engineer')
    })
    expect(await findByText('Designs regression coverage and test plans.')).toBeTruthy()
    expect(
      await findByText(
        (_, element) =>
          element?.textContent ===
          'You are a test engineer.\nWrite regression plans and verify critical paths.',
      ),
    ).toBeTruthy()
    expect(queryByText('You are a fullstack developer.')).toBeNull()
  })
})
