<script lang="ts">
  import { cn } from '$lib/utils'
  import { projectPath } from '$lib/stores/app-context'
  import { appStore } from '$lib/stores/app.svelte'
  import { Badge } from '$ui/badge'
  import Button from '$ui/button/button.svelte'
  import { Skeleton } from '$ui/skeleton'
  import {
    Bot,
    ChevronDown,
    ChevronUp,
    GripVertical,
    PanelLeftClose,
    PanelLeftOpen,
    Settings,
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
  let issuesExpanded = $state(false)

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

  function handleDragStart(e: PointerEvent) {
    dragging = true
    dragStartX = e.clientX
    dragStartWidth = assistantWidth
    ;(e.target as HTMLElement).setPointerCapture(e.pointerId)
  }

  function handleDragMove(e: PointerEvent) {
    if (!dragging) return
    const delta = dragStartX - e.clientX
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
      {#if skillsSettingsHref}
        <a
          href={skillsSettingsHref}
          class="text-muted-foreground hover:text-foreground shrink-0 transition-colors"
          title="Manage skills library"
        >
          <Settings class="size-3.5" />
        </a>
      {/if}
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

  <!-- Editor + AI sidebar horizontal split -->
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
      <!-- Editor column -->
      <div class="flex min-w-0 flex-1 flex-col overflow-hidden">
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
      </div>

      <!-- AI right sidebar -->
      {#if showAssistant}
        <div class="border-border flex border-l" style="width: {assistantWidth}px">
          <!-- Drag handle -->
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
