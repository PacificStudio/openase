<script lang="ts">
  import { Button } from '$ui/button'
  import { Plus, RefreshCw } from '@lucide/svelte'

  let {
    refreshing = false,
    onRefresh,
    onCreate,
  }: {
    refreshing?: boolean
    onRefresh?: () => void
    onCreate?: () => void
  } = $props()
</script>

<Button variant="outline" size="sm" class="gap-1.5" onclick={onRefresh}>
  <RefreshCw class="size-3.5" />
  {refreshing ? 'Refreshing…' : 'Refresh'}
</Button>
<Button size="sm" class="gap-1.5" onclick={onCreate}>
  <Plus class="size-3.5" />
  New machine
</Button>
