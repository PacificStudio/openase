<script lang="ts">
  import Button from '$ui/button/button.svelte'
  import { Columns3, Plus } from '@lucide/svelte'

  let {
    canCreate,
    statusStageHref = null,
    onCreate,
  }: {
    canCreate: boolean
    statusStageHref?: string | null
    onCreate?: () => void
  } = $props()
</script>

<div class="flex items-center gap-2">
  {#if statusStageHref}
    <Button variant="outline" size="sm" href={statusStageHref}>
      <Columns3 class="size-4" />
      Stages & Statuses
    </Button>
  {/if}
  <Button size="sm" onclick={() => onCreate?.()} disabled={!canCreate}>
    <Plus class="size-4" />
    New Workflow
  </Button>
</div>
