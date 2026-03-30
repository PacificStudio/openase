<script lang="ts">
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import Button from '$ui/button/button.svelte'
  import {
    Bot,
    ChevronDown,
    ChevronUp,
    GripHorizontal,
    PanelLeftClose,
    PanelLeftOpen,
    Settings2,
  } from '@lucide/svelte'
  import type { AgentProvider, HarnessValidationIssue } from '$lib/api/contracts'
  import type { HarnessContent, HarnessVariableGroup, WorkflowSummary } from '../types'
  import type { SkillState } from '../model'
  import HarnessEditor from './harness-editor.svelte'
  import HarnessAISidebar from './harness-ai-sidebar.svelte'

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
  let assistantHeight = $state(320)
  let dragging = $state(false)
  let dragStartY = $state(0)
  let dragStartHeight = $state(0)
  let issuesExpanded = $state(false)

  const dictionarySize = $derived(
    variableGroups.reduce((count, group) => count + group.variables.length, 0),
  )

  const MIN_DRAWER_HEIGHT = 200
  const MAX_DRAWER_HEIGHT = 600

  function handleDragStart(e: PointerEvent) {
    dragging = true
    dragStartY = e.clientY
    dragStartHeight = assistantHeight
    ;(e.target as HTMLElement).setPointerCapture(e.pointerId)
  }

  function handleDragMove(e: PointerEvent) {
    if (!dragging) return
    const delta = dragStartY - e.clientY
    assistantHeight = Math.min(
      MAX_DRAWER_HEIGHT,
      Math.max(MIN_DRAWER_HEIGHT, dragStartHeight + delta),
    )
  }

  function handleDragEnd() {
    dragging = false
  }
</script>

<div class="flex flex-1 flex-col overflow-hidden">
  <!-- Toolbar -->
  <div class="border-border flex items-center gap-2 border-b px-3 py-2">
    <Button
      variant="ghost"
      size="icon-sm"
      onclick={onToggleList}
      title={showList ? 'Hide workflow list' : 'Show workflow list'}
    >
      {#if showList}
        <PanelLeftClose class="size-4" />
      {:else}
        <PanelLeftOpen class="size-4" />
      {/if}
    </Button>

    <div class="text-muted-foreground flex min-w-0 items-center gap-2 text-xs">
      <span class="truncate font-medium">{selectedWorkflow?.name ?? 'No workflow selected'}</span>
      {#if isDirty}
        <Badge variant="outline" class="shrink-0 text-[10px]">Unsaved</Badge>
      {/if}
      {#if dictionarySize > 0}
        <Badge variant="outline" class="shrink-0 text-[10px]">{dictionarySize} vars</Badge>
      {/if}
    </div>

    <!-- Skill pills -->
    {#if skillStates.length > 0}
      <div class="border-border mx-1 h-4 w-px shrink-0 border-l"></div>
      <div class="flex min-w-0 flex-1 items-center gap-1.5 overflow-x-auto">
        {#each skillStates as skill (skill.path)}
          <button
            type="button"
            class={cn(
              'flex shrink-0 items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] transition-colors',
              skill.bound
                ? 'border-primary/40 bg-primary/10 text-foreground'
                : 'border-border text-muted-foreground hover:bg-muted',
            )}
            onclick={() => onToggleSkill?.(skill)}
            title={`${skill.bound ? 'Unbind' : 'Bind'} ${skill.name}: ${skill.description}`}
          >
            <span
              class={cn(
                'size-1.5 rounded-full',
                skill.bound ? 'bg-primary' : 'bg-muted-foreground/40',
              )}
            ></span>
            {skill.name}
          </button>
        {/each}
      </div>
    {/if}

    <div class="ml-auto flex shrink-0 items-center gap-1.5">
      <Button
        variant="ghost"
        size="sm"
        onclick={() => (showAssistant = !showAssistant)}
        disabled={!selectedWorkflow}
        class={cn(showAssistant && 'bg-primary/10 text-primary')}
      >
        <Bot class="size-4" />
        AI
      </Button>
      <Button
        variant="outline"
        size="sm"
        onclick={onValidate}
        disabled={validating || !selectedWorkflow}
      >
        {validating ? 'Validating…' : 'Validate'}
      </Button>
      <Button size="sm" onclick={onSave} disabled={!isDirty || saving || !selectedWorkflow}>
        {saving ? 'Saving…' : 'Save'}
      </Button>
      <Button variant="ghost" size="icon-sm" onclick={onToggleDetail} title="Workflow settings">
        <Settings2 class="size-4" />
      </Button>
    </div>
  </div>

  <!-- Editor + AI drawer vertical split -->
  <div class="flex min-h-0 flex-1 flex-col overflow-hidden">
    {#if harness}
      <!-- Editor area -->
      <div class="min-h-0 flex-1 overflow-hidden">
        <HarnessEditor
          content={harness}
          filePath={selectedWorkflow?.harnessPath ?? ''}
          version={selectedWorkflow?.version ?? 1}
          {variableGroups}
          onchange={onDraftChange}
        />
      </div>

      <!-- Validation issues bar -->
      {#if validationIssues.length > 0}
        <div class="border-border border-t">
          <button
            type="button"
            class="flex w-full items-center gap-2 px-3 py-1.5 text-xs text-amber-400 hover:bg-amber-500/5"
            onclick={() => (issuesExpanded = !issuesExpanded)}
          >
            {#if issuesExpanded}
              <ChevronDown class="size-3" />
            {:else}
              <ChevronUp class="size-3" />
            {/if}
            <span class="font-medium"
              >{validationIssues.length} validation issue{validationIssues.length > 1
                ? 's'
                : ''}</span
            >
            {#if !issuesExpanded}
              <span class="text-muted-foreground truncate">— {validationIssues[0].message}</span>
            {/if}
          </button>
          {#if issuesExpanded}
            <div
              class="max-h-32 space-y-1 overflow-y-auto border-t border-amber-500/20 bg-amber-500/5 px-3 py-2 text-xs text-amber-200"
            >
              {#each validationIssues as issue, index (index)}
                <div>
                  {issue.level?.toUpperCase() ?? 'ISSUE'}: {issue.message}
                  {#if issue.line}
                    at line {issue.line}
                  {/if}
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {/if}

      <!-- AI bottom drawer -->
      {#if showAssistant}
        <div class="border-border flex flex-col border-t" style="height: {assistantHeight}px">
          <!-- Drag handle -->
          <div
            class="border-border hover:bg-muted/50 flex cursor-row-resize items-center justify-center py-1 select-none"
            role="separator"
            tabindex="-1"
            onpointerdown={handleDragStart}
            onpointermove={handleDragMove}
            onpointerup={handleDragEnd}
            onpointercancel={handleDragEnd}
          >
            <GripHorizontal class="text-muted-foreground size-4" />
          </div>
          <div class="min-h-0 flex-1 overflow-hidden">
            <HarnessAISidebar
              {projectId}
              {providers}
              workflowId={selectedWorkflow?.id}
              workflowName={selectedWorkflow?.name}
              draftContent={harness.rawContent}
              onApplySuggestion={onApplyAssistantDraft}
            />
          </div>
        </div>
      {/if}
    {/if}
  </div>
</div>
