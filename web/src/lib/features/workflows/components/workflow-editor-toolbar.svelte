<script lang="ts">
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import Button from '$ui/button/button.svelte'
  import { Bot, PanelLeftClose, PanelLeftOpen, Settings, Settings2 } from '@lucide/svelte'
  import type { SkillState } from '../model'
  import type { WorkflowSummary } from '../types'

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
