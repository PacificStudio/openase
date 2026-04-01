<script lang="ts">
  import type { WorkflowHookDraftValidation, WorkflowHooksDraft } from '../workflow-hooks'
  import WorkflowHooksEditor from './workflow-hooks-editor.svelte'

  let {
    draft,
    validation,
    disabled = false,
    onChange,
  }: {
    draft: WorkflowHooksDraft
    validation: WorkflowHookDraftValidation
    disabled?: boolean
    onChange?: (draft: WorkflowHooksDraft) => void
  } = $props()
</script>

<div class="space-y-4">
  <div class="space-y-1">
    <h3 class="text-sm font-semibold">Hooks</h3>
    <p class="text-muted-foreground text-xs">
      Configure workflow and ticket lifecycle hooks using structured rows instead of raw JSON.
    </p>
  </div>
  <WorkflowHooksEditor {draft} {validation} {disabled} {onChange} />
</div>
