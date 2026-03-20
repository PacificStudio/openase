<script lang="ts">
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import Button from '$ui/button/button.svelte'
  import { AlertCircle, CheckCircle2 } from '@lucide/svelte'
  import type { HarnessValidationIssue } from '$lib/api/contracts'
  import type { HarnessContent, WorkflowSummary } from '../types'
  import HarnessEditor from './harness-editor.svelte'

  type SkillState = {
    name: string
    description: string
    path: string
    bound: boolean
  }

  let {
    selectedWorkflow,
    harness,
    skillStates,
    validationIssues,
    statusMessage,
    error,
    saving = false,
    validating = false,
    isDirty = false,
    onDraftChange,
    onSave,
    onValidate,
    onToggleSkill,
  }: {
    selectedWorkflow?: WorkflowSummary
    harness: HarnessContent | null
    skillStates: SkillState[]
    validationIssues: HarnessValidationIssue[]
    statusMessage: string
    error: string
    saving?: boolean
    validating?: boolean
    isDirty?: boolean
    onDraftChange?: (value: string) => void
    onSave?: () => void
    onValidate?: () => void
    onToggleSkill?: (skill: SkillState) => void
  } = $props()
</script>

<div class="flex flex-1 flex-col overflow-hidden">
  <div class="flex items-center justify-between border-b border-border px-4 py-2">
    <div class="flex items-center gap-2 text-xs text-muted-foreground">
      <span>{selectedWorkflow?.name ?? 'No workflow selected'}</span>
      {#if isDirty}
        <Badge variant="outline" class="text-[10px]">Unsaved</Badge>
      {/if}
    </div>
    <div class="flex items-center gap-2">
      <Button variant="outline" size="sm" onclick={onValidate} disabled={validating || !selectedWorkflow}>
        {validating ? 'Validating…' : 'Validate'}
      </Button>
      <Button size="sm" onclick={onSave} disabled={!isDirty || saving || !selectedWorkflow}>
        {saving ? 'Saving…' : 'Save Harness'}
      </Button>
    </div>
  </div>

  <div class="flex-1 overflow-hidden">
    {#if harness}
      <HarnessEditor
        content={harness}
        filePath={selectedWorkflow ? `harness/${selectedWorkflow.id}.md` : ''}
        version={selectedWorkflow?.version ?? 1}
        onchange={onDraftChange}
      />
    {/if}
  </div>

  <div class="border-t border-border px-4 py-3">
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
          {skill.bound ? 'Unbind' : 'Bind'} {skill.name}
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
      <div class="mt-3 flex items-center gap-2 text-xs text-destructive">
        <AlertCircle class="size-3.5" />
        {error}
      </div>
    {/if}

    {#if validationIssues.length > 0}
      <div class="mt-3 space-y-1 rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-100">
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
