import type { ActivityEvent, Project, Ticket } from '$lib/features/workspace'
import type {
  ProjectRepo as ContractProjectRepo,
  TicketDetailPayload as ContractTicketDetailPayload,
  TicketRepoScope as ContractTicketRepoScope,
} from '$lib/api/contracts'

export type ProjectRepo = ContractProjectRepo
export type TicketRepoScope = ContractTicketRepoScope
export type TicketDetailPayload = ContractTicketDetailPayload & {
  ticket: Ticket
  activity: ActivityEvent[]
  hook_history: ActivityEvent[]
}

export type TicketDetailData = {
  project: Project | null
  detail: TicketDetailPayload | null
}
