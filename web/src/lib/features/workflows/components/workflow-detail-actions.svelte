<script lang="ts">
  import { Button } from '$ui/button'
  import { Separator } from '$ui/separator'
  import { AlertCircle, CheckCircle2, Trash2 } from '@lucide/svelte'

  let {
    statusMessage = '',
    errorMessage = '',
    saving = false,
    deleting = false,
    isDirty = false,
    onDelete,
  }: {
    statusMessage?: string
    errorMessage?: string
    saving?: boolean
    deleting?: boolean
    isDirty?: boolean
    onDelete?: () => void | Promise<void>
  } = $props()
</script>

<Separator />

<div class="px-4 py-4">
  {#if statusMessage}
    <div class="mb-4 flex items-center gap-2 text-xs text-emerald-400">
      <CheckCircle2 class="size-3.5" />
      {statusMessage}
    </div>
  {/if}

  {#if errorMessage}
    <div class="text-destructive mb-4 flex items-center gap-2 text-xs">
      <AlertCircle class="size-3.5" />
      {errorMessage}
    </div>
  {/if}

  <div class="flex items-center justify-between gap-3">
    <Button
      type="button"
      variant="ghost"
      class="text-destructive hover:text-destructive"
      disabled={saving || deleting}
      onclick={() => void onDelete?.()}
    >
      <Trash2 class="size-4" />
      {deleting ? 'Deleting…' : 'Delete Workflow'}
    </Button>

    <Button type="submit" size="sm" disabled={!isDirty || saving || deleting}>
      {saving ? 'Saving…' : 'Save Changes'}
    </Button>
  </div>
</div>
