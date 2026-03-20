import { loadOrganizationProjects, loadOrganizations } from '$lib/api/app-context'
import { organizationPath, projectPath, type ProjectSection } from '$lib/stores/app-context'
import { redirect, type LoadEvent } from '@sveltejs/kit'

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

  const projects = await loadOrganizationProjects(fetch, currentOrg.id)
  const currentProject = projects[0]

  if (!currentProject) {
    throw redirect(307, organizationPath(currentOrg.id))
  }

  throw redirect(307, projectPath(currentOrg.id, currentProject.id, section))
}
