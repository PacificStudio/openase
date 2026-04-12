<script lang="ts">
  import Button from '$ui/button/button.svelte'
  import { PanelRight, PanelRightClose, Plus } from '@lucide/svelte'
  import { t } from './i18n'

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

<div class="border-border flex items-center justify-between border-b px-4 py-2.5">
    <h1 class="text-foreground text-sm font-semibold">
      {t('workflows.page.toolbar.title')}
    </h1>
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
      {t('workflows.page.toolbar.actions.newWorkflow')}
    </Button>
  </div>
</div>
