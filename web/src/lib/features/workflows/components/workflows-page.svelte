<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { PageScaffold } from '$lib/components/layout'
  import WorkflowsPageBody from './workflows-page-body.svelte'
  import WorkflowsPageHeaderActions from './workflows-page-header-actions.svelte'
  import WorkflowTemplateGallery from './workflow-template-gallery.svelte'
  import { createWorkflowsPageController } from './workflows-page-controller.svelte'

  const controller = createWorkflowsPageController()
</script>

{#snippet actions()}
  <WorkflowsPageHeaderActions
    canCreate={controller.statuses.length > 0 && controller.agentOptions.length > 0}
    statusStageHref={controller.settingsHref ? `${controller.settingsHref}#statuses` : null}
    onCreate={controller.handleCreateWorkflow}
    onBrowseTemplates={() => (controller.showTemplateGallery = true)}
  />
{/snippet}

<PageScaffold
  title="Workflows"
  description="Edit published harnesses and manage workflow lifecycle settings."
  variant="workspace"
  {actions}
>
  <WorkflowsPageBody
    loading={controller.loading}
    loadingHarness={controller.loadingHarness}
    settingsHref={controller.settingsHref}
    loadError={controller.loadError}
    workflows={controller.workflows}
    selectedId={controller.selectedId}
    projectId={appStore.currentProject?.id ?? ''}
    providers={controller.providers}
    selectedWorkflow={controller.selectedWorkflow}
    harness={controller.harness}
    draftHarness={controller.draftHarness}
    variableGroups={controller.variableGroups}
    skillStates={controller.skillStates}
    validationIssues={controller.validationIssues}
    saving={controller.saving}
    validating={controller.validating}
    isDirty={controller.isDirty}
    bind:showDetail={controller.showDetail}
    bind:showCreateDialog={controller.showCreateDialog}
    bind:showList={controller.showList}
    statuses={controller.statuses}
    agentOptions={controller.agentOptions}
    scopeGroups={controller.scopeGroups}
    builtinRoleContent={controller.builtinRoleContent}
    templateDraft={controller.templateDraft}
    onSelectedIdChange={controller.handleSelectWorkflow}
    onDraftChange={(raw) => (controller.draftHarness = raw)}
    onApplyAssistantDraft={controller.handleApplyAssistantDraft}
    onSave={() => void controller.handleSave()}
    onValidate={() => void controller.handleValidate()}
    onToggleSkill={(skill) => void controller.handleToggleSkill(skill)}
    onWorkflowsChange={(nextWorkflows) => (controller.workflows = nextWorkflows)}
    onCreated={controller.handleCreated}
  />
</PageScaffold>

<WorkflowTemplateGallery
  bind:open={controller.showTemplateGallery}
  onUseTemplate={controller.handleUseTemplate}
/>
