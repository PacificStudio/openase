<script lang="ts">
  import { Button } from '$ui/button'
  import { Plus } from '@lucide/svelte'

  let {
    title,
    description,
    onCreate,
  }: {
    title: string
    description: string
    onCreate?: () => void
  } = $props()
</script>

<div class="border-border flex shrink-0 items-center justify-between gap-3 border-b px-4 py-3">
  <div class="min-w-0">
    <h3 class="text-foreground truncate text-sm font-semibold">{title}</h3>
    <p class="text-muted-foreground mt-0.5 text-xs">{description}</p>
  </div>
  <Button variant="outline" size="sm" onclick={onCreate}>
    <Plus class="mr-1.5 size-3.5" />
    New job
  </Button>
</div>
