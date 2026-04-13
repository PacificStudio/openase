<script lang="ts">
  import { ticketStatusStageOptions, type TicketStatusStage } from '$lib/features/statuses/public'
  import { Button } from '$ui/button'
  import { Checkbox } from '$ui/checkbox'
  import { ColorPicker } from '$ui/color-picker'
  import { Input } from '$ui/input'
  import * as Select from '$ui/select'
  import Plus from '@lucide/svelte/icons/plus'
  import RotateCcw from '@lucide/svelte/icons/rotate-ccw'
  import { i18nStore } from '$lib/i18n/store.svelte'

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
      <h3 class="text-foreground text-sm font-medium">
        {i18nStore.t('settings.status.create.heading')}
      </h3>
      <p class="text-muted-foreground text-xs">
        {i18nStore.t('settings.status.create.description')}
      </p>
    </div>
    <Button variant="outline" size="sm" disabled={resetting || loading} onclick={onReset}>
      <RotateCcw class="size-3.5" />
      {resetting
        ? i18nStore.t('settings.status.create.actions.resetting')
        : i18nStore.t('settings.status.create.actions.reset')}
    </Button>
  </div>

  <div class="space-y-3">
    <div class="flex items-center gap-2">
      <ColorPicker bind:value={color} />
      <Input
        bind:value={name}
        class="h-9 flex-1 text-sm"
        placeholder={i18nStore.t('settings.status.create.placeholders.name')}
      />
      <Select.Root
        type="single"
        value={stage}
        disabled={creating || loading}
        onValueChange={(value) => (stage = (value as TicketStatusStage) || 'unstarted')}
      >
        <Select.Trigger class="h-9 w-40">
          {ticketStatusStageOptions.find((option) => option.value === stage)?.label ??
            i18nStore.t('settings.status.create.placeholders.stage')}
        </Select.Trigger>
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
        placeholder={i18nStore.t('settings.status.create.placeholders.unlimited')}
      />
      <Button class="shrink-0" onclick={onCreate} disabled={creating || loading}>
        <Plus class="size-3.5" />
        {creating
          ? i18nStore.t('settings.status.create.actions.adding')
          : i18nStore.t('settings.status.create.actions.add')}
      </Button>
    </div>
    <label class="flex items-center gap-2">
      <Checkbox bind:checked={isDefault} disabled={creating || loading} />
      <span class="text-sm font-medium">
        {i18nStore.t('settings.status.create.labels.createAsDefault')}
      </span>
      <span class="text-muted-foreground text-xs">
        {isDefault
          ? i18nStore.t('settings.status.create.hints.replaceDefault')
          : i18nStore.t('settings.status.create.hints.keepDefault')}
      </span>
    </label>
    <p class="text-muted-foreground text-xs">
      {i18nStore.t('settings.status.create.hints.stageControls')}
    </p>
  </div>
</div>
