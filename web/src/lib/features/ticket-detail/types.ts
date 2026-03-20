import type { Project } from '$lib/features/workspace'
import type {
  ProjectRepo as ContractProjectRepo,
  TicketDetailPayload as ContractTicketDetailPayload,
  TicketRepoScope as ContractTicketRepoScope,
} from '$lib/api/contracts'

export type ProjectRepo = ContractProjectRepo
export type TicketRepoScope = ContractTicketRepoScope
export type TicketDetailPayload = ContractTicketDetailPayload

export type TicketDetailData = {
  project: Project | null
  detail: TicketDetailPayload | null
}
