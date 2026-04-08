<script lang="ts">
  import { toHarnessContent } from '../model'
  import type {
    HarnessVariableGroup,
    ScopeGroup,
    WorkflowAgentOption,
    WorkflowStatusOption,
    WorkflowSummary,
    WorkflowTemplateDraft,
  } from '../types'
  import type { HarnessValidationIssue } from '$lib/api/contracts'
  import type { SkillState } from '../model'
  import { Button } from '$ui/button'
  import { GitBranch } from '@lucide/svelte'
  import * as Sheet from '$ui/sheet'
  import WorkflowCreationDialog from './workflow-creation-dialog.svelte'
  import WorkflowEditorPanel from './workflow-editor-panel.svelte'
  import WorkflowLifecycleSidebar from './workflow-lifecycle-sidebar.svelte'
  import WorkflowList from './workflow-list.svelte'
  import WorkflowsPageState from './workflows-page-state.svelte'

  let {
    loading = false,
    loadingHarness = false,
    settingsHref = null,
    loadError = '',
    workflows,
    selectedId,
    projectId = '',
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
    scopeGroups = [],
    builtinRoleContent = '',
    templateDraft = null as WorkflowTemplateDraft | null,
    onSelectedIdChange,
    onDraftChange,
    onSave,
    onValidate,
    onToggleSkill,
    onWorkflowsChange,
    onCreated,
  }: {
    loading?: boolean
    loadingHarness?: boolean
    settingsHref?: string | null
    loadError?: string
    workflows: WorkflowSummary[]
    selectedId: string
    projectId?: string
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
    scopeGroups?: ScopeGroup[]
    builtinRoleContent?: string
    templateDraft?: WorkflowTemplateDraft | null
    onSelectedIdChange?: (id: string) => void
    onDraftChange?: (raw: string) => void
    onSave?: () => void
    onValidate?: () => void
    onToggleSkill?: (skill: SkillState) => void
    onWorkflowsChange?: (workflows: WorkflowSummary[]) => void
    onCreated?: (payload: { workflow: WorkflowSummary; selectedId: string }) => void
  } = $props()
</script>

{#if loading || (loadError && workflows.length === 0)}
  <WorkflowsPageState {loading} loadError={workflows.length === 0 ? loadError : ''} />
  {#if settingsHref}
    <div class="px-4 pb-4">
      <a class="text-muted-foreground text-xs underline" href={settingsHref}>
        Open workflow settings details
      </a>
    </div>
  {/if}
{:else if workflows.length === 0}
  <div
    class="border-border bg-card animate-fade-in-up flex flex-1 flex-col items-center justify-center rounded-xl border border-dashed px-4 py-14 text-center"
  >
    <div class="bg-muted/60 mb-4 flex size-12 items-center justify-center rounded-full">
      <GitBranch class="text-muted-foreground size-5" />
    </div>
    <p class="text-foreground text-sm font-medium">No workflows yet</p>
    <p class="text-muted-foreground mt-1 max-w-sm text-sm">
      Workflows define how agents process tickets — sequencing steps, branching on conditions, and
      calling skills. Create one to automate a repeatable process.
    </p>
    <Button variant="outline" size="sm" class="mt-4" onclick={() => (showCreateDialog = true)}>
      Create workflow
    </Button>
  </div>
{:else}
  <div class="border-border/60 bg-card/60 flex min-h-0 flex-1 overflow-hidden rounded-xl border">
    {#if showList}
      <div class="w-52 shrink-0">
        <WorkflowList {workflows} {selectedId} onselect={(id) => onSelectedIdChange?.(id)} />
      </div>
    {/if}
    <WorkflowEditorPanel
      selectedWorkflow={selectedWorkflow ?? undefined}
      harness={harness ? toHarnessContent(draftHarness) : null}
      {variableGroups}
      {skillStates}
      {validationIssues}
      {saving}
      {validating}
      {isDirty}
      {loadingHarness}
      {showList}
      onDraftChange={(raw) => onDraftChange?.(raw)}
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
          {scopeGroups}
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
  {workflows}
  existingCount={workflows.length}
  {builtinRoleContent}
  {templateDraft}
  {onCreated}
/>
