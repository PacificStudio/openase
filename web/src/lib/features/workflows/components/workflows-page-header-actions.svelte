<script lang="ts">
  import Button from '$ui/button/button.svelte'
  import { Plus } from '@lucide/svelte'

  let {
    canCreate,
    onCreate,
  }: {
    canCreate: boolean
    onCreate?: () => void
  } = $props()
</script>

<div class="flex items-center gap-2">
  <Button size="sm" onclick={() => onCreate?.()} disabled={!canCreate}>
    <Plus class="size-4" />
    New Workflow
  </Button>
</div>
