<script lang="ts">
  import Button from '$ui/button/button.svelte'
  import { PanelRight, PanelRightClose, Plus } from '@lucide/svelte'

  let {
    showDetail,
    canCreate,
    onToggleDetail,
    onCreate,
  }: {
    showDetail: boolean
    canCreate: boolean
    onToggleDetail?: () => void
    onCreate?: () => void
  } = $props()
</script>

<div class="flex items-center gap-2">
  <Button variant="ghost" size="sm" onclick={() => onToggleDetail?.()}>
    {#if showDetail}
      <PanelRightClose class="size-4" />
    {:else}
      <PanelRight class="size-4" />
    {/if}
  </Button>
  <Button size="sm" onclick={() => onCreate?.()} disabled={!canCreate}>
    <Plus class="size-4" />
    New Workflow
  </Button>
</div>
