import type { ActivityEvent, Project, Ticket } from '$lib/features/workspace'

export type ProjectRepo = {
  id: string
  project_id: string
  name: string
  repository_url: string
  default_branch: string
  clone_path?: string | null
  is_primary: boolean
  labels: string[]
}

export type TicketRepoScope = {
  id: string
  ticket_id: string
  repo_id: string
  repo?: ProjectRepo | null
  branch_name: string
  pull_request_url?: string | null
  pr_status: string
  ci_status: string
  is_primary_scope: boolean
}

export type TicketDetailPayload = {
  ticket: Ticket
  repo_scopes: TicketRepoScope[]
  activity: ActivityEvent[]
  hook_history: ActivityEvent[]
}

export type TicketDetailData = {
  project: Project | null
  detail: TicketDetailPayload | null
}
