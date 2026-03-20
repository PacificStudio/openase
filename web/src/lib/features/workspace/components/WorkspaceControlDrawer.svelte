<script lang="ts">
  import type { createWorkspaceController } from '$lib/features/workspace/controller.svelte'
  import OrganizationPanel from './OrganizationPanel.svelte'
  import ProjectPanel from './ProjectPanel.svelte'
  import WorkflowPanel from './WorkflowPanel.svelte'
  import SkillPanel from './SkillPanel.svelte'
  import HarnessPanel from './HarnessPanel.svelte'

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

  <WorkflowPanel
    selectedProject={controller.state.selectedProject}
    selectedWorkflowId={controller.state.selectedWorkflowId}
    selectedWorkflow={controller.state.selectedWorkflow}
    selectedBuiltinRoleSlug={controller.state.selectedBuiltinRoleSlug}
    builtinRoles={controller.state.builtinRoles}
    workflows={controller.state.workflows}
    ticketStatuses={controller.board.statuses}
    createForm={controller.state.createWorkflowForm}
    editForm={controller.state.editWorkflowForm}
    hrAdvisor={controller.dashboard.hrAdvisor}
    busy={controller.state.workflowBusy}
    onSelectRole={controller.selectBuiltinRole}
    onClearRole={controller.clearBuiltinRoleSelection}
    onSelectWorkflow={controller.selectWorkflow}
    onCreate={controller.createWorkflow}
    onUpdate={controller.updateWorkflow}
    onDelete={controller.deleteWorkflow}
    onLoadRecommendedRole={controller.loadRecommendedRole}
  />

  <SkillPanel
    skills={controller.state.skills}
    selectedWorkflow={controller.state.selectedWorkflow}
    busy={controller.state.skillBusy}
    pendingSkillName={controller.state.pendingSkillName}
    harnessDirty={controller.harnessDirty()}
    onToggleSkill={controller.toggleSkillBinding}
  />

  <HarnessPanel
    selectedWorkflow={controller.state.selectedWorkflow}
    harnessPath={controller.state.harnessPath}
    harnessVersion={controller.state.harnessVersion}
    bind:harnessDraft={controller.state.harnessDraft}
    harnessIssues={controller.state.harnessIssues}
    validationBusy={controller.state.validationBusy}
    harnessBusy={controller.state.harnessBusy}
    harnessDirty={controller.harnessDirty()}
    onDraftChange={controller.setHarnessDraft}
    onValidate={controller.validateHarnessNow}
    onSave={controller.saveHarness}
  />
</div>
