import type { Organization, Project } from '$lib/api/contracts'
import { organizationPath, projectPath, type ProjectSection } from '$lib/stores/app-context'
import { redirect, type LoadEvent } from '@sveltejs/kit'

type OrgResponse = {
  organizations?: Organization[]
}

type ProjectResponse = {
  projects?: Project[]
}

type AppFetch = LoadEvent['fetch']

export async function redirectToDefaultOrganization(fetch: AppFetch) {
  const organizations = await loadOrganizations(fetch)
  const currentOrg = organizations[0]

  if (currentOrg) {
    throw redirect(307, organizationPath(currentOrg.id))
  }

  return { missingContext: true }
}

export async function redirectToDefaultProject(fetch: AppFetch, section: ProjectSection) {
  const organizations = await loadOrganizations(fetch)
  const currentOrg = organizations[0]

  if (!currentOrg) {
    return { missingContext: true }
  }

  const projects = await loadProjects(fetch, currentOrg.id)
  const currentProject = projects[0]

  if (!currentProject) {
    throw redirect(307, organizationPath(currentOrg.id))
  }

  throw redirect(307, projectPath(currentOrg.id, currentProject.id, section))
}

async function loadOrganizations(fetch: AppFetch) {
  const orgResponse = await fetch('/api/v1/orgs')
  if (!orgResponse.ok) {
    return []
  }

  const orgData = (await orgResponse.json()) as OrgResponse
  return orgData.organizations ?? []
}

async function loadProjects(fetch: AppFetch, orgId: string) {
  const projectResponse = await fetch(`/api/v1/orgs/${orgId}/projects`)
  if (!projectResponse.ok) {
    return []
  }

  const projectData = (await projectResponse.json()) as ProjectResponse
  return projectData.projects ?? []
}
