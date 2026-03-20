<script lang="ts">
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import Button from '$ui/button/button.svelte'
  import { AlertCircle, CheckCircle2, Bot, PanelRightClose, PanelRightOpen } from '@lucide/svelte'
  import type { HarnessValidationIssue } from '$lib/api/contracts'
  import type { HarnessContent, WorkflowSummary } from '../types'
  import HarnessEditor from './harness-editor.svelte'
  import HarnessAISidebar from './harness-ai-sidebar.svelte'

  type SkillState = {
    name: string
    description: string
    path: string
    bound: boolean
  }

  let {
    selectedWorkflow,
    projectId,
    harness,
    skillStates,
    validationIssues,
    statusMessage,
    error,
    saving = false,
    validating = false,
    isDirty = false,
    onDraftChange,
    onApplyAssistantDraft,
    onSave,
    onValidate,
    onToggleSkill,
  }: {
    selectedWorkflow?: WorkflowSummary
    projectId?: string
    harness: HarnessContent | null
    skillStates: SkillState[]
    validationIssues: HarnessValidationIssue[]
    statusMessage: string
    error: string
    saving?: boolean
    validating?: boolean
    isDirty?: boolean
    onDraftChange?: (value: string) => void
    onApplyAssistantDraft?: (value: string) => void
    onSave?: () => void
    onValidate?: () => void
    onToggleSkill?: (skill: SkillState) => void
  } = $props()

  let showAssistant = $state(true)
</script>

<div class="flex flex-1 flex-col overflow-hidden">
  <div class="border-border flex items-center justify-between border-b px-4 py-2">
    <div class="text-muted-foreground flex items-center gap-2 text-xs">
      <span>{selectedWorkflow?.name ?? 'No workflow selected'}</span>
      {#if isDirty}
        <Badge variant="outline" class="text-[10px]">Unsaved</Badge>
      {/if}
    </div>
    <div class="flex items-center gap-2">
      <Button
        variant="ghost"
        size="sm"
        onclick={() => (showAssistant = !showAssistant)}
        disabled={!selectedWorkflow}
      >
        {#if showAssistant}
          <PanelRightClose class="size-4" />
        {:else}
          <PanelRightOpen class="size-4" />
        {/if}
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
        {saving ? 'Saving…' : 'Save Harness'}
      </Button>
    </div>
  </div>

  <div class="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
    {#if harness}
      <div class="min-h-0 flex-1 overflow-hidden">
        <HarnessEditor
          content={harness}
          filePath={selectedWorkflow ? `harness/${selectedWorkflow.id}.md` : ''}
          version={selectedWorkflow?.version ?? 1}
          onchange={onDraftChange}
        />
      </div>
      {#if showAssistant}
        <div
          class="border-border h-[32rem] shrink-0 border-t lg:h-auto lg:w-[28rem] lg:border-t-0 lg:border-l"
        >
          <HarnessAISidebar
            {projectId}
            workflowId={selectedWorkflow?.id}
            workflowName={selectedWorkflow?.name}
            draftContent={harness.rawContent}
            onApplySuggestion={onApplyAssistantDraft}
          />
        </div>
      {/if}
    {/if}
  </div>

  <div class="border-border border-t px-4 py-3">
    <div class="flex flex-wrap items-center gap-2">
      {#each skillStates as skill (skill.path)}
        <button
          type="button"
          class={cn(
            'rounded-full border px-2.5 py-1 text-xs transition-colors',
            skill.bound
              ? 'border-primary/40 bg-primary/10 text-foreground'
              : 'border-border text-muted-foreground hover:bg-muted',
          )}
          onclick={() => onToggleSkill?.(skill)}
          title={skill.description}
        >
          {skill.bound ? 'Unbind' : 'Bind'}
          {skill.name}
        </button>
      {/each}
    </div>

    {#if statusMessage}
      <div class="mt-3 flex items-center gap-2 text-xs text-emerald-400">
        <CheckCircle2 class="size-3.5" />
        {statusMessage}
      </div>
    {/if}

    {#if error}
      <div class="text-destructive mt-3 flex items-center gap-2 text-xs">
        <AlertCircle class="size-3.5" />
        {error}
      </div>
    {/if}

    {#if validationIssues.length > 0}
      <div
        class="mt-3 space-y-1 rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-100"
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
</div>
