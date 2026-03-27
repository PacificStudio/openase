<script lang="ts">
  import { Button } from '$ui/button'
  import { Separator } from '$ui/separator'
  import { AlertCircle, Trash2 } from '@lucide/svelte'

  let {
    errorMessage = '',
    saving = false,
    deleting = false,
    isDirty = false,
    onDelete,
  }: {
    errorMessage?: string
    saving?: boolean
    deleting?: boolean
    isDirty?: boolean
    onDelete?: () => void | Promise<void>
  } = $props()
</script>

<Separator />

<div class="px-4 py-4">
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
