<script lang="ts">
  import { toHarnessContent } from '../model'
  import type {
    HarnessVariableGroup,
    WorkflowAgentOption,
    WorkflowStatusOption,
    WorkflowSummary,
  } from '../types'
  import type { AgentProvider, HarnessValidationIssue } from '$lib/api/contracts'
  import type { SkillState } from '../model'
  import type { WorkflowRepositoryPrerequisite } from '../repository-prerequisite'
  import * as Sheet from '$ui/sheet'
  import WorkflowCreationDialog from './workflow-creation-dialog.svelte'
  import WorkflowEditorPanel from './workflow-editor-panel.svelte'
  import WorkflowLifecycleSidebar from './workflow-lifecycle-sidebar.svelte'
  import WorkflowList from './workflow-list.svelte'
  import WorkflowsPageState from './workflows-page-state.svelte'

  let {
    loading = false,
    prerequisite = null,
    settingsHref = null,
    loadError = '',
    workflows,
    selectedId,
    projectId = '',
    providers,
    selectedWorkflow = null,
    harness,
    draftHarness,
    variableGroups,
    skillStates,
    validationIssues,
    saving = false,
    validating = false,
    isDirty = false,
    showDetail = $bindable(false),
    showCreateDialog = $bindable(false),
    showList = $bindable(true),
    statuses,
    agentOptions,
    builtinRoleContent = '',
    onSelectedIdChange,
    onDraftChange,
    onApplyAssistantDraft,
    onSave,
    onValidate,
    onToggleSkill,
    onWorkflowsChange,
    onCreated,
  }: {
    loading?: boolean
    prerequisite?: WorkflowRepositoryPrerequisite | null
    settingsHref?: string | null
    loadError?: string
    workflows: WorkflowSummary[]
    selectedId: string
    projectId?: string
    providers: AgentProvider[]
    selectedWorkflow?: WorkflowSummary | null
    harness: ReturnType<typeof toHarnessContent> | null
    draftHarness: string
    variableGroups: HarnessVariableGroup[]
    skillStates: SkillState[]
    validationIssues: HarnessValidationIssue[]
    saving?: boolean
    validating?: boolean
    isDirty?: boolean
    showDetail?: boolean
    showCreateDialog?: boolean
    showList?: boolean
    statuses: WorkflowStatusOption[]
    agentOptions: WorkflowAgentOption[]
    builtinRoleContent?: string
    onSelectedIdChange?: (id: string) => void
    onDraftChange?: (raw: string) => void
    onApplyAssistantDraft?: (content: string) => void
    onSave?: () => void
    onValidate?: () => void
    onToggleSkill?: (skill: SkillState) => void
    onWorkflowsChange?: (workflows: WorkflowSummary[]) => void
    onCreated?: (payload: { workflow: WorkflowSummary; selectedId: string }) => void
  } = $props()
</script>

{#if loading || (prerequisite && prerequisite.kind !== 'ready') || (loadError && workflows.length === 0)}
  <WorkflowsPageState
    {loading}
    {prerequisite}
    loadError={workflows.length === 0 ? loadError : ''}
  />
{:else}
  <div class="border-border/60 bg-card/60 flex min-h-0 flex-1 overflow-hidden rounded-xl border">
    {#if showList}
      <div class="w-52 shrink-0">
        <WorkflowList {workflows} {selectedId} onselect={(id) => onSelectedIdChange?.(id)} />
      </div>
    {/if}
    <WorkflowEditorPanel
      projectId={projectId || undefined}
      {providers}
      selectedWorkflow={selectedWorkflow ?? undefined}
      harness={harness ? toHarnessContent(draftHarness) : null}
      {variableGroups}
      {skillStates}
      {validationIssues}
      {saving}
      {validating}
      {isDirty}
      {showList}
      onDraftChange={(raw) => onDraftChange?.(raw)}
      {onApplyAssistantDraft}
      {onSave}
      {onValidate}
      onToggleSkill={(skill) => onToggleSkill?.(skill)}
      onToggleList={() => (showList = !showList)}
      onToggleDetail={() => (showDetail = !showDetail)}
    />
  </div>

  <Sheet.Root bind:open={showDetail}>
    <Sheet.Content side="right" class="w-[60%] overflow-y-auto p-0 sm:max-w-[60%]">
      <Sheet.Header class="sr-only">
        <Sheet.Title>Workflow Settings</Sheet.Title>
        <Sheet.Description>Configure workflow lifecycle settings.</Sheet.Description>
      </Sheet.Header>
      {#if selectedWorkflow}
        <WorkflowLifecycleSidebar
          workflow={selectedWorkflow}
          {workflows}
          {statuses}
          {agentOptions}
          onWorkflowsChange={(nextWorkflows) => onWorkflowsChange?.(nextWorkflows)}
          onSelectedIdChange={(nextSelectedId) => onSelectedIdChange?.(nextSelectedId)}
        />
      {/if}
    </Sheet.Content>
  </Sheet.Root>
{/if}

<WorkflowCreationDialog
  bind:open={showCreateDialog}
  {projectId}
  {statuses}
  {agentOptions}
  existingCount={workflows.length}
  {builtinRoleContent}
  {onCreated}
/>
