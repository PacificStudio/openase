import { api, toErrorMessage } from './api'
import { defaultProjectForm } from './mappers'
import type { Organization, Project, ProjectForm } from './types'

type WorkspaceState = {
  orgBusy: boolean
  projectBusy: boolean
  errorMessage: string
  notice: string
  createOrgForm: { name: string; slug: string }
  editOrgForm: { name: string; slug: string }
  createProjectForm: ProjectForm
  editProjectForm: ProjectForm
  selectedOrg: Organization | null
  selectedProject: Project | null
}

type Dependencies = {
  state: WorkspaceState
  loadOrganizations: (preferredOrgId?: string) => Promise<void>
  loadProjects: (orgId: string, preferredProjectId?: string) => Promise<void>
}

export function createEntityMutationActions({
  state,
  loadOrganizations,
  loadProjects,
}: Dependencies) {
  async function createOrganization() {
    await runOrgMutation(async () => {
      const payload = await api<{ organization: Organization }>('/api/v1/orgs', {
        method: 'POST',
        body: JSON.stringify(state.createOrgForm),
      })
      state.createOrgForm = { name: '', slug: '' }
      state.notice = `Organization ${payload.organization.name} created`
      await loadOrganizations(payload.organization.id)
    })
  }

  async function updateOrganization() {
    const selectedOrg = state.selectedOrg
    if (!selectedOrg) {
      return
    }

    await runOrgMutation(async () => {
      await api(`/api/v1/orgs/${selectedOrg.id}`, {
        method: 'PATCH',
        body: JSON.stringify(state.editOrgForm),
      })
      state.notice = `Organization ${state.editOrgForm.name} updated`
      await loadOrganizations(selectedOrg.id)
    })
  }

  async function createProject() {
    const selectedOrg = state.selectedOrg
    if (!selectedOrg) {
      return
    }

    await runProjectMutation(async () => {
      const payload = await api<{ project: Project }>(`/api/v1/orgs/${selectedOrg.id}/projects`, {
        method: 'POST',
        body: JSON.stringify({
          name: state.createProjectForm.name,
          slug: state.createProjectForm.slug,
          description: state.createProjectForm.description,
          status: state.createProjectForm.status,
          max_concurrent_agents: state.createProjectForm.maxConcurrentAgents,
        }),
      })
      state.createProjectForm = defaultProjectForm()
      state.notice = `Project ${payload.project.name} created`
      await loadProjects(selectedOrg.id, payload.project.id)
    })
  }

  async function updateProject() {
    const selectedOrg = state.selectedOrg
    const selectedProject = state.selectedProject
    if (!selectedOrg || !selectedProject) {
      return
    }

    await runProjectMutation(async () => {
      await api(`/api/v1/projects/${selectedProject.id}`, {
        method: 'PATCH',
        body: JSON.stringify({
          name: state.editProjectForm.name,
          slug: state.editProjectForm.slug,
          description: state.editProjectForm.description,
          status: state.editProjectForm.status,
          max_concurrent_agents: state.editProjectForm.maxConcurrentAgents,
        }),
      })
      state.notice = `Project ${state.editProjectForm.name} updated`
      await loadProjects(selectedOrg.id, selectedProject.id)
    })
  }

  async function archiveProject() {
    const selectedOrg = state.selectedOrg
    const selectedProject = state.selectedProject
    if (!selectedOrg || !selectedProject) {
      return
    }

    await runProjectMutation(async () => {
      await api(`/api/v1/projects/${selectedProject.id}`, { method: 'DELETE' })
      state.notice = `Project ${selectedProject.name} archived`
      await loadProjects(selectedOrg.id, selectedProject.id)
    })
  }

  return {
    createOrganization,
    updateOrganization,
    createProject,
    updateProject,
    archiveProject,
  }

  async function runOrgMutation(mutation: () => Promise<void>) {
    state.orgBusy = true
    state.errorMessage = ''
    state.notice = ''
    try {
      await mutation()
    } catch (error) {
      state.errorMessage = toErrorMessage(error)
    } finally {
      state.orgBusy = false
    }
  }

  async function runProjectMutation(mutation: () => Promise<void>) {
    state.projectBusy = true
    state.errorMessage = ''
    state.notice = ''
    try {
      await mutation()
    } catch (error) {
      state.errorMessage = toErrorMessage(error)
    } finally {
      state.projectBusy = false
    }
  }
}
