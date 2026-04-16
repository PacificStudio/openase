import { describe, expect, it } from 'vitest'

import type { MockState } from './constants'
import { createMachineRecord } from './entities'

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

describe('mock machine execution capabilities', () => {
  it('keeps reverse-connect runtimes aligned with the core remote capability set', () => {
    const machine = createMachineRecord(mockState(), {
      name: 'reverse-runtime',
      host: 'remote.internal',
      reachability_mode: 'reverse_connect',
      execution_mode: 'websocket',
    })

    expect(machine.execution_capabilities).toEqual([
      'probe',
      'workspace_prepare',
      'artifact_sync',
      'process_streaming',
    ])
  })
})
