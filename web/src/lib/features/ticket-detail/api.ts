import { api } from '$lib/features/workspace'
import type { Project } from '$lib/features/workspace'
import type { TicketDetailData, TicketDetailPayload } from './types'

export async function loadTicketDetailData(
  projectId: string,
  ticketId: string,
): Promise<TicketDetailData> {
  const [detailPayload, projectPayload] = await Promise.all([
    api<TicketDetailPayload>(`/api/v1/projects/${projectId}/tickets/${ticketId}/detail`),
    api<{ project: Project }>(`/api/v1/projects/${projectId}`),
  ])

  return {
    detail: detailPayload,
    project: projectPayload.project,
  }
}
