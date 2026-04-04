export type DraftParseResult<T> =
  | {
      ok: true
      value: T
    }
  | {
      ok: false
      error: string
    }

export type OrganizationCreationDraft = {
  name: string
  slug: string
}

export type ProjectCreationDraft = {
  name: string
  slug: string
  description: string
  status: string
  maxConcurrentAgents: string
  defaultAgentProviderId: string
}

export const projectStatusOptions = [
  'Backlog',
  'Planned',
  'In Progress',
  'Completed',
  'Canceled',
  'Archived',
] as const

export function createOrganizationDraft(): OrganizationCreationDraft {
  return {
    name: '',
    slug: '',
  }
}

export function createProjectDraft(
  defaultAgentProviderId: string | null = null,
): ProjectCreationDraft {
  return {
    name: '',
    slug: '',
    description: '',
    status: 'Planned',
    maxConcurrentAgents: '',
    defaultAgentProviderId: defaultAgentProviderId ?? '',
  }
}

export function parseOrganizationDraft(draft: OrganizationCreationDraft): DraftParseResult<{
  name: string
  slug: string
}> {
  const name = draft.name.trim()
  if (!name) {
    return { ok: false, error: 'Organization name is required.' }
  }

  const slug = parseSlug(draft.slug)
  if (!slug.ok) {
    return slug
  }

  return {
    ok: true,
    value: {
      name,
      slug: slug.value,
    },
  }
}

export function parseProjectDraft(draft: ProjectCreationDraft): DraftParseResult<{
  name: string
  slug: string
  description: string
  status: string
  max_concurrent_agents?: number
  default_agent_provider_id?: string | null
}> {
  const name = draft.name.trim()
  if (!name) {
    return { ok: false, error: 'Project name is required.' }
  }

  const slug = parseSlug(draft.slug)
  if (!slug.ok) {
    return slug
  }

  const status = draft.status
  if (!projectStatusOptions.includes(status as (typeof projectStatusOptions)[number])) {
    return {
      ok: false,
      error: `Project status must be one of ${projectStatusOptions.join(', ')}.`,
    }
  }

  const maxConcurrentAgents = parseOptionalPositiveInteger(
    'Max concurrent agents',
    draft.maxConcurrentAgents,
  )
  if (!maxConcurrentAgents.ok) {
    return maxConcurrentAgents
  }

  const defaultAgentProviderId = draft.defaultAgentProviderId.trim()

  return {
    ok: true,
    value: {
      name,
      slug: slug.value,
      description: draft.description.trim(),
      status,
      max_concurrent_agents: maxConcurrentAgents.value,
      default_agent_provider_id: defaultAgentProviderId || undefined,
    },
  }
}

export function slugFromName(raw: string) {
  return raw
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .replace(/-{2,}/g, '-')
}

function parseSlug(raw: string): DraftParseResult<string> {
  const slug = raw.trim().toLowerCase()
  if (!slug) {
    return { ok: false, error: 'Slug is required.' }
  }

  if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(slug)) {
    return {
      ok: false,
      error: 'Slug must use lowercase letters, numbers, and single hyphens.',
    }
  }

  return { ok: true, value: slug }
}

function parseOptionalPositiveInteger(
  label: string,
  raw: string,
): DraftParseResult<number | undefined> {
  const value = raw.trim()
  if (!value) {
    return { ok: true, value: undefined }
  }

  const parsed = Number(value)
  if (!Number.isInteger(parsed) || parsed < 1) {
    return { ok: false, error: `${label} must be a positive integer.` }
  }

  return { ok: true, value: parsed }
}
