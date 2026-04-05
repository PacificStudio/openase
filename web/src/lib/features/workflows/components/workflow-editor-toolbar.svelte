<script lang="ts">
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import Button from '$ui/button/button.svelte'
  import { Bot, PanelLeftClose, PanelLeftOpen, Settings2 } from '@lucide/svelte'
  import type { SkillState } from '../model'
  import type { WorkflowSummary } from '../types'
  import WorkflowSkillsDropdown from './workflow-skills-dropdown.svelte'

  let {
    selectedWorkflow,
    skillStates,
    skillsSettingsHref,
    showList = true,
    isDirty = false,
    dictionarySize = 0,
    showAssistant = false,
    saving = false,
    validating = false,
    onToggleSkill,
    onToggleList,
    onToggleAssistant,
    onValidate,
    onSave,
    onToggleDetail,
  }: {
    selectedWorkflow?: WorkflowSummary
    skillStates: SkillState[]
    skillsSettingsHref: string | null
    showList?: boolean
    isDirty?: boolean
    dictionarySize?: number
    showAssistant?: boolean
    saving?: boolean
    validating?: boolean
    onToggleSkill?: (skill: SkillState) => void
    onToggleList?: () => void
    onToggleAssistant?: () => void
    onValidate?: () => void
    onSave?: () => void
    onToggleDetail?: () => void
  } = $props()
</script>

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

  {#if skillStates.length > 0}
    <WorkflowSkillsDropdown {skillStates} {skillsSettingsHref} {onToggleSkill} />
  {/if}

  <div class="ml-auto flex shrink-0 items-center gap-1.5">
    <Button
      variant="ghost"
      size="sm"
      onclick={onToggleAssistant}
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
