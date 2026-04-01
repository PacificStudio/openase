<script lang="ts">
  import { projectPath } from '$lib/stores/app-context'
  import { appStore } from '$lib/stores/app.svelte'
  import { Skeleton } from '$ui/skeleton'
  import { GripVertical } from '@lucide/svelte'
  import type { AgentProvider, HarnessValidationIssue } from '$lib/api/contracts'
  import type { SkillState } from '../model'
  import type { HarnessContent, HarnessVariableGroup, WorkflowSummary } from '../types'
  import HarnessAISidebar from './harness-ai-sidebar.svelte'
  import HarnessEditor from './harness-editor.svelte'
  import WorkflowEditorToolbar from './workflow-editor-toolbar.svelte'
  import WorkflowEditorValidationIssues from './workflow-editor-validation-issues.svelte'

  let {
    selectedWorkflow,
    projectId,
    providers = [],
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
    onApplyAssistantDraft,
    onSave,
    onValidate,
    onToggleSkill,
    onToggleList,
    onToggleDetail,
  }: {
    selectedWorkflow?: WorkflowSummary
    projectId?: string
    providers?: AgentProvider[]
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
    onApplyAssistantDraft?: (value: string) => void
    onSave?: () => void
    onValidate?: () => void
    onToggleSkill?: (skill: SkillState) => void
    onToggleList?: () => void
    onToggleDetail?: () => void
  } = $props()

  let showAssistant = $state(false)
  let assistantWidth = $state(340)
  let dragging = $state(false)
  let dragStartX = $state(0)
  let dragStartWidth = $state(0)

  const skillsSettingsHref = $derived.by(() => {
    const orgId = appStore.currentOrg?.id
    const projId = appStore.currentProject?.id
    if (!orgId || !projId) return null
    return `${projectPath(orgId, projId, 'settings')}#skills`
  })

  const dictionarySize = $derived(
    variableGroups.reduce((count, group) => count + group.variables.length, 0),
  )

  const MIN_SIDEBAR_WIDTH = 260
  const MAX_SIDEBAR_WIDTH = 560

  function handleDragStart(event: PointerEvent) {
    dragging = true
    dragStartX = event.clientX
    dragStartWidth = assistantWidth
    ;(event.target as HTMLElement).setPointerCapture(event.pointerId)
  }

  function handleDragMove(event: PointerEvent) {
    if (!dragging) return
    const delta = dragStartX - event.clientX
    assistantWidth = Math.min(
      MAX_SIDEBAR_WIDTH,
      Math.max(MIN_SIDEBAR_WIDTH, dragStartWidth + delta),
    )
  }

  function handleDragEnd() {
    dragging = false
  }
</script>

<div class="flex flex-1 flex-col overflow-hidden">
  <WorkflowEditorToolbar
    {selectedWorkflow}
    {skillStates}
    {skillsSettingsHref}
    {showList}
    {isDirty}
    {dictionarySize}
    {showAssistant}
    {saving}
    {validating}
    {onToggleSkill}
    {onToggleList}
    onToggleAssistant={() => (showAssistant = !showAssistant)}
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

      {#if showAssistant}
        <div class="border-border flex border-l" style="width: {assistantWidth}px">
          <div
            class="hover:bg-muted/50 flex w-1 cursor-col-resize items-center justify-center select-none"
            role="separator"
            tabindex="-1"
            onpointerdown={handleDragStart}
            onpointermove={handleDragMove}
            onpointerup={handleDragEnd}
            onpointercancel={handleDragEnd}
          >
            <GripVertical class="text-muted-foreground/40 size-3" />
          </div>
          <div class="min-w-0 flex-1 overflow-hidden">
            <HarnessAISidebar
              {projectId}
              {providers}
              workflowId={selectedWorkflow?.id}
              draftContent={harness.rawContent}
              onApplySuggestion={onApplyAssistantDraft}
            />
          </div>
        </div>
      {/if}
    {/if}
  </div>
</div>
