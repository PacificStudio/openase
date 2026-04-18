import type { MockState } from './constants'
import { DEFAULT_WORKFLOW_ID, PROJECT_ID, nowIso } from './constants'
import { createDefaultSecuritySettings } from './security'
import {
  createSeedActivityEvents,
  createSeedAgentRuns,
  createSeedAgents,
  createSeedMachines,
  createSeedOrganizations,
  createSeedProjects,
  createSeedProviders,
} from './initial-state-seed-core'
import {
  createSeedBuiltinRoles,
  createSeedCounters,
  createSeedHarnessByWorkflowId,
  createSeedHarnessVariables,
  createSeedRepos,
  createSeedScheduledJobs,
  createSeedSkills,
  createSeedStatuses,
  createSeedTickets,
  createSeedWorkflows,
} from './initial-state-seed-project'
import { createMockTicketRecord } from './ticket-data'
import { asNumber, asString } from './helpers'

export function createInitialState(): MockState {
  return {
    organizations: createSeedOrganizations(),
    projects: createSeedProjects(),
    machines: createSeedMachines(),
    providers: createSeedProviders(),
    agents: createSeedAgents(),
    agentRuns: createSeedAgentRuns(),
    activityEvents: createSeedActivityEvents(),
    projectUpdates: [],
    tickets: createSeedTickets(),
    statuses: createSeedStatuses(),
    repos: createSeedRepos(),
    workflows: createSeedWorkflows(),
    harnessByWorkflowId: createSeedHarnessByWorkflowId(),
    scheduledJobs: createSeedScheduledJobs(),
    projectConversations: [],
    projectConversationEntries: [],
    skills: createSeedSkills(),
    builtinRoles: createSeedBuiltinRoles(),
    securitySettingsByProjectId: {
      [PROJECT_ID]: createDefaultSecuritySettings(PROJECT_ID),
    },
    harnessVariables: createSeedHarnessVariables(),
    counters: createSeedCounters(),
  }
}

export function seedBoardState(state: MockState, countsByStatusID: Record<string, number>) {
  const statusNameByID = new Map(
    state.statuses.map((status) => [asString(status.id) ?? '', asString(status.name) ?? 'Todo']),
  )
  const seededTickets: Record<string, unknown>[] = []
  let sequence = 0

  for (const status of state.statuses) {
    const statusId = asString(status.id)
    if (!statusId) {
      continue
    }

    const count = Math.max(0, asNumber(countsByStatusID[statusId]) ?? 0)
    for (let index = 0; index < count; index += 1) {
      sequence += 1
      seededTickets.push(
        createMockTicketRecord({
          id: `ticket-seeded-${sequence}`,
          identifier: `ASE-${100 + sequence}`,
          title: `Seeded ticket ${sequence}`,
          statusId,
          statusName: statusNameByID.get(statusId) ?? 'Todo',
          workflowId: DEFAULT_WORKFLOW_ID,
          createdAt: new Date(Date.parse(nowIso) + sequence * 60_000).toISOString(),
        }),
      )
    }
  }

  state.tickets = seededTickets
  state.activityEvents = []
}
