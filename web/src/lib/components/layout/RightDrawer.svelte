<script lang="ts">
  import type { Snippet } from 'svelte'
  import { PanelRightOpen } from '@lucide/svelte'
  import ScrollPane from '$lib/components/layout/ScrollPane.svelte'
  import SurfacePanel from '$lib/components/layout/SurfacePanel.svelte'

  let {
    children,
    title = 'Workspace drawer',
    description = 'Keep project setup, workflow tuning, and harness editing off the route layer.',
  }: {
    children?: Snippet
    title?: string
    description?: string
  } = $props()
</script>

<SurfacePanel class="h-full">
  {#snippet header()}
    <div>
      <div class="flex items-center gap-2">
        <PanelRightOpen class="size-4" />
        <p class="text-sm font-semibold">{title}</p>
      </div>
      <p class="text-muted-foreground mt-1 text-xs leading-5">{description}</p>
    </div>
  {/snippet}

  <ScrollPane class="h-full px-4 py-4">
    <div class="space-y-4">
      {@render children?.()}
    </div>
  </ScrollPane>
</SurfacePanel>
