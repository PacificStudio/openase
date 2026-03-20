<script lang="ts">
  import {
    createWorkspaceController,
    HarnessPanel,
    SkillPanel,
    WorkflowPanel,
  } from '$lib/features/workspace'
  import WorkflowAssistantPane from './WorkflowAssistantPane.svelte'
  import WorkflowPreviewPane from './WorkflowPreviewPane.svelte'
  import WorkflowVariablesPane from './WorkflowVariablesPane.svelte'

  let {
    controller,
  }: {
    controller: ReturnType<typeof createWorkspaceController>
  } = $props()
</script>

<svelte:head>
  <title>Workflows · OpenASE</title>
</svelte:head>

<div class="space-y-6">
  <div class="grid gap-6 xl:grid-cols-[24rem_minmax(0,1fr)]">
    <div class="space-y-6">
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

      <WorkflowAssistantPane
        builtinRoles={controller.state.builtinRoles}
        hrAdvisor={controller.dashboard.hrAdvisor}
        onLoadRecommendedRole={controller.loadRecommendedRole}
      />
    </div>

    <div class="space-y-6">
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

      <div class="grid gap-6 xl:grid-cols-[minmax(0,1.15fr)_20rem]">
        <WorkflowPreviewPane
          selectedWorkflow={controller.state.selectedWorkflow}
          statuses={controller.board.statuses}
        />
        <WorkflowVariablesPane />
      </div>

      <SkillPanel
        skills={controller.state.skills}
        selectedWorkflow={controller.state.selectedWorkflow}
        busy={controller.state.skillBusy}
        pendingSkillName={controller.state.pendingSkillName}
        harnessDirty={controller.harnessDirty()}
        onToggleSkill={controller.toggleSkillBinding}
      />
    </div>
  </div>
</div>
