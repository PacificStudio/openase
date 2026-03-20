<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import Plus from '@lucide/svelte/icons/plus'
  import RotateCcw from '@lucide/svelte/icons/rotate-ccw'

  let {
    name = $bindable(''),
    color = $bindable('#94a3b8'),
    isDefault = $bindable(false),
    creating = false,
    loading = false,
    resetting = false,
    onCreate,
    onReset,
  }: {
    name?: string
    color?: string
    isDefault?: boolean
    creating?: boolean
    loading?: boolean
    resetting?: boolean
    onCreate: () => Promise<void> | void
    onReset: () => Promise<void> | void
  } = $props()
</script>

<div class="border-border bg-card space-y-4 rounded-md border p-4">
  <div class="flex items-center justify-between gap-3">
    <div>
      <h3 class="text-foreground text-sm font-medium">Add Status</h3>
      <p class="text-muted-foreground text-xs">
        Create statuses, set the default lane, and keep board order aligned with product flow.
      </p>
    </div>
    <Button variant="outline" size="sm" disabled={resetting || loading} onclick={onReset}>
      <RotateCcw class="size-3.5" />
      {resetting ? 'Resetting…' : 'Reset'}
    </Button>
  </div>

  <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
    <Input bind:value={name} class="h-9 flex-1 text-sm" placeholder="New status name" />
    <input
      type="color"
      bind:value={color}
      class="size-9 shrink-0 rounded border-0 bg-transparent p-0"
    />
    <label class="text-muted-foreground flex items-center gap-2 text-xs font-medium">
      <input type="checkbox" bind:checked={isDefault} />
      Default
    </label>
    <Button onclick={onCreate} disabled={creating || loading}>
      <Plus class="size-3.5" />
      {creating ? 'Adding…' : 'Add'}
    </Button>
  </div>
</div>
