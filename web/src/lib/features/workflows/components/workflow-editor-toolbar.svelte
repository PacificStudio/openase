<script lang="ts">
  import { Badge } from '$ui/badge'
  import Button from '$ui/button/button.svelte'
  import { PanelLeftClose, PanelLeftOpen, Settings2 } from '@lucide/svelte'
  import type { SkillState } from '../model'
  import type { WorkflowSummary } from '../types'
  import type { TranslationKey, TranslationParams } from '$lib/i18n/index'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import WorkflowSkillsDropdown from './workflow-skills-dropdown.svelte'

  let {
    selectedWorkflow,
    skillStates,
    skillsSettingsHref,
    showList = true,
    isDirty = false,
    dictionarySize = 0,
    saving = false,
    validating = false,
    onToggleSkill,
    onToggleList,
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
    saving?: boolean
    validating?: boolean
    onToggleSkill?: (skill: SkillState) => void
    onToggleList?: () => void
    onValidate?: () => void
    onSave?: () => void
    onToggleDetail?: () => void
  } = $props()

  function t(key: TranslationKey, params?: TranslationParams) {
    return i18nStore.t(key, params)
  }
</script>

<div class="border-border flex items-center gap-2 border-b px-3 py-2">
  <Button
    variant="ghost"
    size="icon-sm"
    onclick={onToggleList}
    title={showList
      ? t('workflows.editor.toolbar.actions.hideList')
      : t('workflows.editor.toolbar.actions.showList')}
  >
    {#if showList}
      <PanelLeftClose class="size-4" />
    {:else}
      <PanelLeftOpen class="size-4" />
    {/if}
  </Button>

  <div class="text-muted-foreground flex min-w-0 items-center gap-2 text-xs">
    <span class="truncate font-medium">
      {selectedWorkflow?.name ?? t('workflows.editor.toolbar.placeholders.noWorkflow')}
    </span>
    {#if isDirty}
      <Badge variant="outline" class="shrink-0 text-[10px]">
        {t('workflows.editor.toolbar.badge.unsaved')}
      </Badge>
    {/if}
    {#if dictionarySize > 0}
      <Badge variant="outline" class="shrink-0 text-[10px]">
        {t('workflows.editor.toolbar.badge.vars', { count: dictionarySize })}
      </Badge>
    {/if}
  </div>

  {#if skillStates.length > 0}
    <WorkflowSkillsDropdown {skillStates} {skillsSettingsHref} {onToggleSkill} />
  {/if}

  <div class="ml-auto flex shrink-0 items-center gap-1.5">
    <Button
      variant="outline"
      size="sm"
      onclick={onValidate}
      disabled={validating || !selectedWorkflow}
    >
      {validating
        ? t('workflows.editor.toolbar.actions.validating')
        : t('workflows.editor.toolbar.actions.validate')}
    </Button>
    <Button size="sm" onclick={onSave} disabled={!isDirty || saving || !selectedWorkflow}>
      {saving
        ? t('workflows.editor.toolbar.actions.saving')
        : t('workflows.editor.toolbar.actions.save')}
    </Button>
    <Button
      variant="ghost"
      size="icon-sm"
      onclick={onToggleDetail}
      title={t('workflows.editor.toolbar.actions.settings')}
    >
      <Settings2 class="size-4" />
    </Button>
  </div>
</div>
