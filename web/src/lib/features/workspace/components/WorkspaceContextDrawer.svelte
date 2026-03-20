<script lang="ts">
  import type { createWorkspaceController } from '$lib/features/workspace/controller.svelte'
  import OrganizationPanel from './OrganizationPanel.svelte'
  import ProjectPanel from './ProjectPanel.svelte'

  let {
    controller,
  }: {
    controller: ReturnType<typeof createWorkspaceController>
  } = $props()
</script>

<div class="space-y-6">
  <OrganizationPanel
    selectedOrg={controller.state.selectedOrg}
    createForm={controller.state.createOrgForm}
    editForm={controller.state.editOrgForm}
    busy={controller.state.orgBusy}
    onCreate={controller.createOrganization}
    onUpdate={controller.updateOrganization}
  />

  <ProjectPanel
    selectedOrg={controller.state.selectedOrg}
    selectedProject={controller.state.selectedProject}
    createForm={controller.state.createProjectForm}
    editForm={controller.state.editProjectForm}
    busy={controller.state.projectBusy}
    onCreate={controller.createProject}
    onUpdate={controller.updateProject}
    onArchive={controller.archiveProject}
  />
</div>
