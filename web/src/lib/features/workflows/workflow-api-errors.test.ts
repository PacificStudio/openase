import { describe, expect, it } from 'vitest'

import { ApiError } from '$lib/api/client'

import { describeWorkflowApiError } from './workflow-api-errors'

describe('describeWorkflowApiError', () => {
  it('maps workflow name conflicts to a friendly create/update message', () => {
    expect(
      describeWorkflowApiError(
        new ApiError(
          409,
          'workflow name "Coding Workflow" is already used in this project',
          'WORKFLOW_NAME_CONFLICT',
        ),
        'Failed to save workflow.',
      ),
    ).toBe('A workflow with this name already exists in the project.')
  })

  it('maps harness path conflicts to a friendly message', () => {
    expect(
      describeWorkflowApiError(
        new ApiError(
          409,
          'harness_path ".openase/harnesses/coding.md" is already used by another workflow',
          'WORKFLOW_HARNESS_PATH_CONFLICT',
        ),
        'Failed to save workflow.',
      ),
    ).toBe('This harness path is already used by another workflow.')
  })

  it('maps overlapping workflow status bindings to a friendly message', () => {
    expect(
      describeWorkflowApiError(
        new ApiError(
          409,
          'workflow pickup and finish statuses must not overlap',
          'WORKFLOW_STATUS_BINDING_OVERLAP',
        ),
        'Failed to save workflow.',
      ),
    ).toBe('Pickup and finish statuses must be mutually exclusive.')
  })

  it('maps ticket references to a friendly delete message', () => {
    expect(
      describeWorkflowApiError(
        new ApiError(
          409,
          'tickets still reference this workflow',
          'WORKFLOW_REFERENCED_BY_TICKETS',
        ),
        'Failed to delete workflow.',
      ),
    ).toBe('This workflow cannot be deleted because tickets still reference it.')
  })

  it('maps scheduled job references to a friendly delete message', () => {
    expect(
      describeWorkflowApiError(
        new ApiError(
          409,
          'scheduled jobs still reference this workflow',
          'WORKFLOW_REFERENCED_BY_SCHEDULED_JOBS',
        ),
        'Failed to delete workflow.',
      ),
    ).toBe('This workflow cannot be deleted because scheduled jobs still reference it.')
  })

  it('falls back to backend detail when no specialized mapping exists', () => {
    expect(
      describeWorkflowApiError(new ApiError(409, 'workflow conflict'), 'Failed to save workflow.'),
    ).toBe('workflow conflict')
  })

  it('falls back to the provided default for non-api errors', () => {
    expect(describeWorkflowApiError(new Error('boom'), 'Failed to save workflow.')).toBe(
      'Failed to save workflow.',
    )
  })
})
