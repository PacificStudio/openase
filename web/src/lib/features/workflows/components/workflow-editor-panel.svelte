<script lang="ts">
  import { projectPath } from '$lib/stores/app-context'
  import { appStore } from '$lib/stores/app.svelte'
  import { Skeleton } from '$ui/skeleton'
  import type { HarnessValidationIssue } from '$lib/api/contracts'
  import type { SkillState } from '../model'
  import type { HarnessContent, HarnessVariableGroup, WorkflowSummary } from '../types'
  import HarnessEditor from './harness-editor.svelte'
  import WorkflowEditorToolbar from './workflow-editor-toolbar.svelte'
  import WorkflowEditorValidationIssues from './workflow-editor-validation-issues.svelte'

  let {
    selectedWorkflow,
    harness,
    variableGroups = [],
    skillStates,
    validationIssues,
    saving = false,
    validating = false,
    isDirty = false,
    loadingHarness = false,
    showList = true,
    onDraftChange,
    onSave,
    onValidate,
    onToggleSkill,
    onToggleList,
    onToggleDetail,
  }: {
    selectedWorkflow?: WorkflowSummary
    harness: HarnessContent | null
    variableGroups?: HarnessVariableGroup[]
    skillStates: SkillState[]
    validationIssues: HarnessValidationIssue[]
    saving?: boolean
    validating?: boolean
    isDirty?: boolean
    loadingHarness?: boolean
    showList?: boolean
    onDraftChange?: (value: string) => void
    onSave?: () => void
    onValidate?: () => void
    onToggleSkill?: (skill: SkillState) => void
    onToggleList?: () => void
    onToggleDetail?: () => void
  } = $props()

  const skillsSettingsHref = $derived.by(() => {
    const orgId = appStore.currentOrg?.id
    const projId = appStore.currentProject?.id
    if (!orgId || !projId) return null
    return `${projectPath(orgId, projId, 'settings')}#skills`
  })

  const dictionarySize = $derived(
    variableGroups.reduce((count, group) => count + group.variables.length, 0),
  )
</script>

<div class="flex flex-1 flex-col overflow-hidden">
  <WorkflowEditorToolbar
    {selectedWorkflow}
    {skillStates}
    {skillsSettingsHref}
    {showList}
    {isDirty}
    {dictionarySize}
    {saving}
    {validating}
    {onToggleSkill}
    {onToggleList}
    {onValidate}
    {onSave}
    {onToggleDetail}
  />

  <div class="flex min-h-0 flex-1 overflow-hidden">
    {#if loadingHarness && !harness}
      <div class="flex-1 space-y-2.5 p-4">
        <Skeleton class="h-3.5 w-[30%]" />
        <Skeleton class="h-3.5 w-[65%]" />
        <Skeleton class="h-3.5 w-[45%]" />
        <Skeleton class="h-3.5 w-[80%]" />
        <Skeleton class="h-3.5 w-[35%]" />
        <div class="h-2"></div>
        <Skeleton class="h-3.5 w-[50%]" />
        <Skeleton class="h-3.5 w-[70%]" />
        <Skeleton class="h-3.5 w-[55%]" />
        <Skeleton class="h-3.5 w-[40%]" />
        <Skeleton class="h-3.5 w-[75%]" />
        <div class="h-2"></div>
        <Skeleton class="h-3.5 w-[60%]" />
        <Skeleton class="h-3.5 w-[85%]" />
        <Skeleton class="h-3.5 w-[42%]" />
      </div>
    {:else if harness}
      <div class="flex min-w-0 flex-1 flex-col overflow-hidden">
        <div class="min-h-0 flex-1 overflow-hidden">
          <HarnessEditor
            content={harness}
            filePath={selectedWorkflow?.harnessPath ?? ''}
            version={selectedWorkflow?.version ?? 1}
            {variableGroups}
            onchange={onDraftChange}
          />
        </div>

        {#if validationIssues.length > 0}
          <WorkflowEditorValidationIssues {validationIssues} />
        {/if}
      </div>
    {/if}
  </div>
</div>
