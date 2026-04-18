import { describe, expect, it } from 'vitest'

import type { MockState } from './constants'
import {
  createProjectUpdateCommentRecord,
  createProjectUpdateThreadRecord,
  deleteProjectUpdateCommentRecord,
  deleteProjectUpdateThreadRecord,
  parseProjectUpdateStatus,
  summarizeProjectUpdateTitle,
  updateProjectUpdateCommentRecord,
  updateProjectUpdateThreadRecord,
} from './project-update-records'

function mockState(): MockState {
  return {
    organizations: [],
    projects: [],
    machines: [],
    providers: [],
    agents: [],
    agentRuns: [],
    activityEvents: [],
    projectUpdates: [],
    tickets: [],
    statuses: [],
    repos: [],
    workflows: [],
    harnessByWorkflowId: {},
    scheduledJobs: [],
    projectConversations: [],
    projectConversationEntries: [],
    skills: [],
    builtinRoles: [],
    securitySettingsByProjectId: {},
    harnessVariables: { groups: [] },
    counters: {
      machine: 0,
      repo: 0,
      workflow: 0,
      agent: 0,
      skill: 0,
      scheduledJob: 0,
      projectUpdateThread: 0,
      projectUpdateComment: 0,
      projectConversation: 0,
      projectConversationEntry: 0,
      projectConversationTurn: 0,
    },
  }
}

describe('project update records', () => {
  it('normalizes thread status and titles from the latest body', () => {
    const thread = createProjectUpdateThreadRecord(mockState(), 'project-1', {
      status: 'unknown',
      body: 'First line\nSecond line',
    })

    expect(thread.status).toBe('on_track')
    expect(thread.title).toBe('First line')
    expect(summarizeProjectUpdateTitle('')).toBe('Update')
    expect(parseProjectUpdateStatus('off_track')).toBe('off_track')
  })

  it('tracks comment and thread lifecycle edits', () => {
    const state = mockState()
    const thread = createProjectUpdateThreadRecord(state, 'project-1', { body: 'Initial' })
    updateProjectUpdateThreadRecord(thread, { body: 'Updated body', edited_by: 'user:test' })
    deleteProjectUpdateThreadRecord(thread)

    const comment = createProjectUpdateCommentRecord(state, 'thread-1', { body: 'Hello' })
    updateProjectUpdateCommentRecord(comment, { body: 'Updated comment', edited_by: 'user:test' })
    deleteProjectUpdateCommentRecord(comment)

    expect(thread.body_markdown).toBe('Updated body')
    expect(thread.edit_count).toBe(1)
    expect(thread.is_deleted).toBe(true)
    expect(comment.body_markdown).toBe('Updated comment')
    expect(comment.edit_count).toBe(1)
    expect(comment.is_deleted).toBe(true)
  })
})
