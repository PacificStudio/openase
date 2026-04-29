<script lang="ts">
  import * as Collapsible from '$ui/collapsible'
  import { ChevronRight } from '@lucide/svelte'
  import { t } from './i18n'

  import type { WorkflowHookDraftValidation, WorkflowHooksDraft } from '../workflow-hooks'
  import WorkflowHooksEditor from './workflow-hooks-editor.svelte'

  let {
    open = $bindable(false),
    draft,
    validation,
    disabled = false,
    error = '',
    onChange,
  }: {
    open?: boolean
    draft: WorkflowHooksDraft
    validation: WorkflowHookDraftValidation
    disabled?: boolean
    error?: string
    onChange?: (nextDraft: WorkflowHooksDraft) => void
  } = $props()
</script>

<Collapsible.Root bind:open>
  <Collapsible.Trigger>
    {#snippet child({ props })}
      <button
        {...props}
        type="button"
        class="text-muted-foreground hover:text-foreground flex items-center gap-1 text-sm transition-colors"
      >
        <ChevronRight class="size-4 transition-transform {open ? 'rotate-90' : ''}" />
        {t('workflows.creation.advancedSection.toggle')}
      </button>
    {/snippet}
  </Collapsible.Trigger>
  <Collapsible.Content>
    <div class="mt-3 space-y-4">
      <div class="space-y-1">
        <div class="text-sm font-medium">
          {t('workflows.creation.advancedSection.hooksTitle')}
        </div>
        <p class="text-muted-foreground text-xs">
          {t('workflows.creation.advancedSection.hooksDescription')}
        </p>
      </div>

      <WorkflowHooksEditor
        {draft}
        {validation}
        {disabled}
        onChange={(nextDraft) => onChange?.(nextDraft)}
      />

      {#if error}
        <p class="text-destructive text-xs">{error}</p>
      {/if}
    </div>
  </Collapsible.Content>
</Collapsible.Root>
