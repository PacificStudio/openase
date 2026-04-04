<script lang="ts">
  import { Button } from '$ui/button'
  import { Plus, RefreshCw } from '@lucide/svelte'

  let {
    refreshing = false,
    refreshDisabled = false,
    createDisabled = false,
    onRefresh,
    onCreate,
  }: {
    refreshing?: boolean
    refreshDisabled?: boolean
    createDisabled?: boolean
    onRefresh?: () => void
    onCreate?: () => void
  } = $props()
</script>

<Button variant="outline" size="sm" class="gap-1.5" onclick={onRefresh} disabled={refreshDisabled}>
  <RefreshCw class="size-3.5" />
  {refreshing ? 'Refreshing…' : 'Refresh'}
</Button>
<Button size="sm" class="gap-1.5" onclick={onCreate} disabled={createDisabled}>
  <Plus class="size-3.5" />
  New machine
</Button>
