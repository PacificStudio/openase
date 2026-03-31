<script lang="ts">
  import type { EditableStage } from '$lib/features/stages/public'
  import { Button } from '$ui/button'
  import { Checkbox } from '$ui/checkbox'
  import { Input } from '$ui/input'
  import * as Select from '$ui/select'
  import Plus from '@lucide/svelte/icons/plus'
  import RotateCcw from '@lucide/svelte/icons/rotate-ccw'

  let {
    name = $bindable(''),
    color = $bindable('#94a3b8'),
    isDefault = $bindable(false),
    stageId = $bindable(''),
    stages = [],
    creating = false,
    loading = false,
    resetting = false,
    onCreate,
    onReset,
  }: {
    name?: string
    color?: string
    isDefault?: boolean
    stageId?: string
    stages?: EditableStage[]
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

  <div class="space-y-3">
    <div class="flex items-center gap-2">
      <input
        type="color"
        bind:value={color}
        class="size-9 shrink-0 rounded border-0 bg-transparent p-0"
      />
      <Input bind:value={name} class="h-9 flex-1 text-sm" placeholder="New status name" />
      <Select.Root
        type="single"
        value={stageId}
        onValueChange={(value) => (stageId = value || '')}
        disabled={creating || loading}
      >
        <Select.Trigger class="w-44 text-left text-sm">
          {stages.find((stage) => stage.id === stageId)?.name ?? 'Ungrouped'}
        </Select.Trigger>
        <Select.Content>
          <Select.Item value="">Ungrouped</Select.Item>
          {#each stages as stage (stage.id)}
            <Select.Item value={stage.id}>{stage.name}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
      <Button class="shrink-0" onclick={onCreate} disabled={creating || loading}>
        <Plus class="size-3.5" />
        {creating ? 'Adding…' : 'Add'}
      </Button>
    </div>
    <label class="flex items-center gap-2">
      <Checkbox bind:checked={isDefault} disabled={creating || loading} />
      <span class="text-sm font-medium">Create as default</span>
      <span class="text-muted-foreground text-xs">
        {isDefault
          ? 'The new status will replace the current default.'
          : 'Leave this off to keep the current default status.'}
      </span>
    </label>
  </div>
</div>
