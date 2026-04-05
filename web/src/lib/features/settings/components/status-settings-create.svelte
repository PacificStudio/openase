<script lang="ts">
  import { ticketStatusStageOptions, type TicketStatusStage } from '$lib/features/statuses/public'
  import { Button } from '$ui/button'
  import { Checkbox } from '$ui/checkbox'
  import { ColorPicker } from '$ui/color-picker'
  import { Input } from '$ui/input'
  import * as Select from '$ui/select'
  import Plus from '@lucide/svelte/icons/plus'
  import RotateCcw from '@lucide/svelte/icons/rotate-ccw'

  let {
    name = $bindable(''),
    stage = $bindable('unstarted'),
    color = $bindable('#94a3b8'),
    isDefault = $bindable(false),
    maxActiveRuns = $bindable(''),
    creating = false,
    loading = false,
    resetting = false,
    onCreate,
    onReset,
  }: {
    name?: string
    stage?: TicketStatusStage
    color?: string
    isDefault?: boolean
    maxActiveRuns?: string
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
      <ColorPicker bind:value={color} />
      <Input bind:value={name} class="h-9 flex-1 text-sm" placeholder="New status name" />
      <Select.Root
        type="single"
        value={stage}
        disabled={creating || loading}
        onValueChange={(value) => (stage = (value as TicketStatusStage) || 'unstarted')}
      >
        <Select.Trigger class="h-9 w-40"
          >{ticketStatusStageOptions.find((option) => option.value === stage)?.label ??
            'Stage'}</Select.Trigger
        >
        <Select.Content>
          {#each ticketStatusStageOptions as option (option.value)}
            <Select.Item value={option.value}>{option.label}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
      <Input
        bind:value={maxActiveRuns}
        type="number"
        min="1"
        step="1"
        class="h-9 w-40 text-sm"
        placeholder="Unlimited"
      />
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
    <p class="text-muted-foreground text-xs">
      Stage controls lifecycle semantics such as dependency resolution and workflow terminal states.
    </p>
  </div>
</div>
