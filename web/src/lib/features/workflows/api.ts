import { api } from '$lib/api/client'
import type {
  HarnessDocument,
  HarnessPayload,
  HarnessVariableDictionaryPayload,
  OrganizationPayload,
  ProjectPayload,
  StatusPayload,
  WorkflowListPayload,
} from '$lib/api/contracts'
import type { HarnessContent, HarnessVariableGroup, WorkflowSummary, WorkflowType } from './types'

export type WorkflowPageData = {
  orgName: string | null
  projectName: string | null
  workflows: WorkflowSummary[]
  harnessDocuments: Record<string, HarnessContent>
  variableGroups: HarnessVariableGroup[]
}

const harnessDocumentPattern = /^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/

export function splitHarnessContent(rawContent: string): HarnessContent {
  const match = rawContent.match(harnessDocumentPattern)
  if (!match) {
    return {
      frontmatter: '',
      body: rawContent,
      rawContent,
    }
  }

  return {
    frontmatter: match[1] ?? '',
    body: match[2] ?? '',
    rawContent,
  }
}

export function toHarnessContent(document: HarnessDocument): HarnessContent {
  return splitHarnessContent(document.content)
}

function mapWorkflowSummary(
  workflow: WorkflowListPayload['workflows'][number],
  statusNamesByID: Map<string, string>,
): WorkflowSummary {
  return {
    id: workflow.id,
    name: workflow.name,
    type: workflow.type as WorkflowType,
    harnessPath: workflow.harness_path,
    pickupStatus: statusNamesByID.get(workflow.pickup_status_id) ?? workflow.pickup_status_id,
    finishStatus: workflow.finish_status_id
      ? (statusNamesByID.get(workflow.finish_status_id) ?? workflow.finish_status_id)
      : 'Unassigned',
    maxConcurrent: workflow.max_concurrent,
    maxRetry: workflow.max_retry_attempts,
    timeoutMinutes: workflow.timeout_minutes,
    stallTimeoutMinutes: workflow.stall_timeout_minutes,
    isActive: workflow.is_active,
    version: workflow.version,
  }
}

export async function loadWorkflowPageData(signal?: AbortSignal): Promise<WorkflowPageData> {
  const orgResponse = await api.get<OrganizationPayload>('/api/v1/orgs', { signal })
  const organization = orgResponse.organizations[0] ?? null
  if (!organization) {
    return {
      orgName: null,
      projectName: null,
      workflows: [],
      harnessDocuments: {},
      variableGroups: [],
    }
  }

  const projectResponse = await api.get<ProjectPayload>(
    `/api/v1/orgs/${organization.id}/projects`,
    { signal },
  )
  const project = projectResponse.projects[0] ?? null
  if (!project) {
    return {
      orgName: organization.name,
      projectName: null,
      workflows: [],
      harnessDocuments: {},
      variableGroups: [],
    }
  }

  const [workflowResponse, statusResponse, variableResponse] = await Promise.all([
    api.get<WorkflowListPayload>(`/api/v1/projects/${project.id}/workflows`, { signal }),
    api.get<StatusPayload>(`/api/v1/projects/${project.id}/statuses`, { signal }),
    api.get<HarnessVariableDictionaryPayload>('/api/v1/harness/variables', { signal }),
  ])

  const statusNamesByID = new Map(statusResponse.statuses.map((status) => [status.id, status.name]))
  const workflows = workflowResponse.workflows.map((workflow) =>
    mapWorkflowSummary(workflow, statusNamesByID),
  )

  const harnessEntries = await Promise.all(
    workflows.map(async (workflow) => {
      const response = await api.get<HarnessPayload>(`/api/v1/workflows/${workflow.id}/harness`, {
        signal,
      })
      return [workflow.id, toHarnessContent(response.harness)] as const
    }),
  )

  return {
    orgName: organization.name,
    projectName: project.name,
    workflows,
    harnessDocuments: Object.fromEntries(harnessEntries),
    variableGroups: variableResponse.groups,
  }
}

export async function saveWorkflowHarness(
  workflowID: string,
  content: string,
  signal?: AbortSignal,
): Promise<HarnessDocument> {
  const response = await api.put<HarnessPayload>(`/api/v1/workflows/${workflowID}/harness`, {
    body: { content },
    signal,
  })
  return response.harness
}
