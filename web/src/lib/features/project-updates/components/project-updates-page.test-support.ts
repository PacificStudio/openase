import { cleanup } from '@testing-library/svelte'
import { afterEach, beforeEach, vi } from 'vitest'

import type {
  Project,
  ProjectUpdateCommentRecord,
  ProjectUpdateThreadRecord,
} from '$lib/api/contracts'
import { resetProjectUpdatesCacheForTests } from '../project-updates-cache'
import { appStore } from '$lib/stores/app.svelte'

const projectUpdatesPageMocks = vi.hoisted(() => ({
  subscribeProjectEvents: vi.fn(),
  createProjectUpdateComment: vi.fn(),
  createProjectUpdateThread: vi.fn(),
  deleteProjectUpdateComment: vi.fn(),
  deleteProjectUpdateThread: vi.fn(),
  listProjectUpdates: vi.fn(),
  updateProjectUpdateComment: vi.fn(),
  updateProjectUpdateThread: vi.fn(),
}))

export const subscribeProjectEvents = projectUpdatesPageMocks.subscribeProjectEvents
export const createProjectUpdateComment = projectUpdatesPageMocks.createProjectUpdateComment
export const createProjectUpdateThread = projectUpdatesPageMocks.createProjectUpdateThread
export const deleteProjectUpdateComment = projectUpdatesPageMocks.deleteProjectUpdateComment
export const deleteProjectUpdateThread = projectUpdatesPageMocks.deleteProjectUpdateThread
export const listProjectUpdates = projectUpdatesPageMocks.listProjectUpdates
export const updateProjectUpdateComment = projectUpdatesPageMocks.updateProjectUpdateComment
export const updateProjectUpdateThread = projectUpdatesPageMocks.updateProjectUpdateThread

vi.mock('$lib/api/openase', () => ({
  createProjectUpdateComment,
  createProjectUpdateThread,
  deleteProjectUpdateComment,
  deleteProjectUpdateThread,
  listProjectUpdates,
  updateProjectUpdateComment,
  updateProjectUpdateThread,
}))

vi.mock('$lib/features/project-events', async () => {
  const actual = await vi.importActual<typeof import('$lib/features/project-events')>(
    '$lib/features/project-events',
  )
  return {
    ...actual,
    isProjectUpdateEvent: (event: { topic?: string; type?: string }) =>
      event.topic === 'activity.events' &&
      typeof event.type === 'string' &&
      event.type.startsWith('project_update_'),
    subscribeProjectEvents: projectUpdatesPageMocks.subscribeProjectEvents,
  }
})

export const projectFixture: Project = {
  id: 'project-1',
  organization_id: 'org-1',
  name: 'OpenASE',
  slug: 'openase',
  description: 'Autonomous software engineering',
  status: 'active',
  default_agent_provider_id: null,
  accessible_machine_ids: [],
  max_concurrent_agents: 4,
}

export type StreamEventHandler = (event: {
  topic: string
  type: string
  payload: unknown
  publishedAt: string
}) => void

export function setupProjectUpdatesPageTest() {
  beforeEach(() => {
    vi.resetAllMocks()
    vi.stubGlobal(
      'confirm',
      vi.fn(() => true),
    )
    appStore.currentProject = projectFixture
    subscribeProjectEvents.mockReturnValue(() => {})
  })

  afterEach(() => {
    cleanup()
    resetProjectUpdatesCacheForTests()
    appStore.currentProject = null
    vi.unstubAllGlobals()
  })
}

export function makeCommentRecord(
  overrides: Partial<ProjectUpdateCommentRecord> = {},
): ProjectUpdateCommentRecord {
  return {
    id: 'comment-1',
    thread_id: 'thread-1',
    body_markdown: 'Need one more canary.',
    created_by: 'user:ops',
    created_at: '2026-04-01T10:32:00Z',
    updated_at: '2026-04-01T10:32:00Z',
    edited_at: null,
    edit_count: 0,
    last_edited_by: null,
    is_deleted: false,
    deleted_at: null,
    deleted_by: null,
    ...overrides,
  }
}

export function makeThreadRecord(
  overrides: Partial<ProjectUpdateThreadRecord> = {},
): ProjectUpdateThreadRecord {
  return {
    id: 'thread-1',
    project_id: 'project-1',
    status: 'on_track',
    title: 'Sprint 2 rollout',
    body_markdown: 'Launch is green.',
    created_by: 'user:codex',
    created_at: '2026-04-01T08:00:00Z',
    updated_at: '2026-04-01T08:15:00Z',
    edited_at: null,
    edit_count: 0,
    last_edited_by: null,
    is_deleted: false,
    deleted_at: null,
    deleted_by: null,
    last_activity_at: '2026-04-01T08:15:00Z',
    comment_count: 0,
    comments: [],
    ...overrides,
  }
}
