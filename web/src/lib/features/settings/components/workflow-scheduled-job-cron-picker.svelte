<script lang="ts">
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Clock, Calendar, Code } from '@lucide/svelte'
  import type { TranslationKey } from '$lib/i18n'
  import {
    type ScheduleConfig,
    type ScheduleMode,
    defaultScheduleConfig,
    buildCronExpression,
    clampScheduleNumber,
    parseCronToConfig,
    getNextTriggerTimes,
    getScheduleIntervalMax,
    formatTriggerTime,
  } from './cron-utils'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    value = '',
    onchange,
  }: {
    value?: string
    onchange?: (cron: string) => void
  } = $props()

  let manualMode = $state(false)
  let manualValue = $state('')
  let config = $state<ScheduleConfig>(defaultScheduleConfig())
  let initialized = $state(false)

  $effect(() => {
    if (initialized) return
    initialized = true

    if (!value.trim()) {
      config = defaultScheduleConfig()
      manualMode = false
      return
    }

    const parsed = parseCronToConfig(value)
    if (parsed) {
      config = parsed
      manualMode = false
    } else {
      manualValue = value
      manualMode = true
    }
  })

  const cronExpression = $derived(manualMode ? manualValue : buildCronExpression(config))

  const nextTriggers = $derived.by(() => {
    if (manualMode) {
      const parsed = parseCronToConfig(manualValue)
      if (parsed) return getNextTriggerTimes(parsed, 5)
      return []
    }
    return getNextTriggerTimes(config, 5)
  })

  const SCHEDULE_MODE_SEQUENCE: ScheduleMode[] = [
    'seconds',
    'minutes',
    'hours',
    'daily',
    'monthly',
  ]
  const SCHEDULE_MODE_LABEL_KEYS: Record<ScheduleMode, TranslationKey> = {
    seconds: 'settings.workflowScheduledJobCronPicker.scheduleUnits.seconds',
    minutes: 'settings.workflowScheduledJobCronPicker.scheduleUnits.minutes',
    hours: 'settings.workflowScheduledJobCronPicker.scheduleUnits.hours',
    daily: 'settings.workflowScheduledJobCronPicker.scheduleUnits.daily',
    monthly: 'settings.workflowScheduledJobCronPicker.scheduleUnits.monthly',
  }
  const modeLabel = $derived(() => i18nStore.t(SCHEDULE_MODE_LABEL_KEYS[config.mode]))

  function emitChange() {
    const cron = manualMode ? manualValue : buildCronExpression(config)
    onchange?.(cron)
  }

  function updateConfig<K extends keyof ScheduleConfig>(key: K, val: ScheduleConfig[K]) {
    config = { ...config, [key]: val }
    emitChange()
  }

  function switchToManual() {
    manualValue = buildCronExpression(config)
    manualMode = true
  }

  function switchToPicker() {
    const parsed = parseCronToConfig(manualValue)
    if (parsed) {
      config = parsed
    } else {
      config = defaultScheduleConfig()
    }
    manualMode = false
    emitChange()
  }

  function handleManualInput(event: Event) {
    manualValue = (event.currentTarget as HTMLInputElement).value
    emitChange()
  }
</script>

<div class="space-y-3">
  {#if manualMode}
    <!-- Manual cron input -->
    <div class="space-y-1.5">
      <div class="flex items-center justify-between">
        <Label class="text-xs">{i18nStore.t('settings.workflowScheduledJobCronPicker.labels.cronExpression')}</Label>
        <button
          type="button"
          class="text-muted-foreground hover:text-foreground flex items-center gap-1 text-[11px] transition-colors"
          onclick={switchToPicker}
        >
          <Calendar class="size-3" />
          {i18nStore.t('settings.workflowScheduledJobCronPicker.actions.visualPicker')}
        </button>
      </div>
      <Input
        value={manualValue}
        placeholder={i18nStore.t('settings.workflowScheduledJobCronPicker.placeholders.example')}
        class="font-mono"
        oninput={handleManualInput}
      />
    </div>
  {:else}
    <!-- Visual picker -->
    <div class="space-y-1.5">
      <div class="flex items-center justify-between">
        <Label class="text-xs">{i18nStore.t('settings.workflowScheduledJobCronPicker.labels.schedule')}</Label>
        <button
          type="button"
          class="text-muted-foreground hover:text-foreground flex items-center gap-1 text-[11px] transition-colors"
          onclick={switchToManual}
        >
          <Code class="size-3" />
          {i18nStore.t('settings.workflowScheduledJobCronPicker.actions.manualInput')}
        </button>
      </div>

      <!-- Every N [unit] -->
      <div class="flex items-center gap-2">
        <span class="text-muted-foreground shrink-0 text-xs">
          {i18nStore.t('settings.workflowScheduledJobCronPicker.labels.every')}
        </span>
        <Input
          type="number"
          min="1"
          max={getScheduleIntervalMax(config.mode)}
          value={String(config.interval)}
          class="h-8 w-16 text-center text-sm"
          oninput={(e) => {
            updateConfig(
              'interval',
              clampScheduleNumber(
                (e.currentTarget as HTMLInputElement).value,
                1,
                getScheduleIntervalMax(config.mode),
                1,
              ),
            )
          }}
        />
        <Select.Root
          type="single"
          value={config.mode}
          onValueChange={(val) => {
            if (val) updateConfig('mode', val as ScheduleMode)
          }}
        >
          <Select.Trigger class="h-8 w-[7.5rem] text-sm">{modeLabel}</Select.Trigger>
        <Select.Content>
          {#each SCHEDULE_MODE_SEQUENCE as mode}
            <Select.Item value={mode}>{i18nStore.t(SCHEDULE_MODE_LABEL_KEYS[mode])}</Select.Item>
          {/each}
        </Select.Content>
        </Select.Root>
      </div>
    </div>

    <!-- Conditional time fields -->
    {#if config.mode === 'hours'}
      <div class="flex items-center gap-2">
        <span class="text-muted-foreground shrink-0 text-xs">
          {i18nStore.t('settings.workflowScheduledJobCronPicker.labels.atMinute')}
        </span>
        <Input
          type="number"
          min="0"
          max="59"
          value={String(config.atMinute)}
          class="h-8 w-16 text-center text-sm"
          oninput={(e) =>
            updateConfig(
              'atMinute',
              clampScheduleNumber((e.currentTarget as HTMLInputElement).value, 0, 59, 0),
            )}
        />
      </div>
    {/if}

    {#if config.mode === 'daily'}
      <div class="flex items-center gap-2">
        <span class="text-muted-foreground shrink-0 text-xs">
          {i18nStore.t('settings.workflowScheduledJobCronPicker.labels.at')}
        </span>
        <Input
          type="number"
          min="0"
          max="23"
          value={String(config.atHour).padStart(2, '0')}
          class="h-8 w-16 text-center font-mono text-sm"
          oninput={(e) =>
            updateConfig(
              'atHour',
              clampScheduleNumber((e.currentTarget as HTMLInputElement).value, 0, 23, 0),
            )}
        />
        <span class="text-muted-foreground text-sm">:</span>
        <Input
          type="number"
          min="0"
          max="59"
          value={String(config.atMinute).padStart(2, '0')}
          class="h-8 w-16 text-center font-mono text-sm"
          oninput={(e) =>
            updateConfig(
              'atMinute',
              clampScheduleNumber((e.currentTarget as HTMLInputElement).value, 0, 59, 0),
            )}
        />
      </div>
    {/if}

    {#if config.mode === 'monthly'}
      <div class="flex items-center gap-2">
        <span class="text-muted-foreground shrink-0 text-xs">
          {i18nStore.t('settings.workflowScheduledJobCronPicker.labels.onDay')}
        </span>
        <Input
          type="number"
          min="1"
          max="31"
          value={String(config.atDay)}
          class="h-8 w-16 text-center text-sm"
          oninput={(e) =>
            updateConfig(
              'atDay',
              clampScheduleNumber((e.currentTarget as HTMLInputElement).value, 1, 31, 1),
            )}
        />
        <span class="text-muted-foreground shrink-0 text-xs">
          {i18nStore.t('settings.workflowScheduledJobCronPicker.labels.at')}
        </span>
        <Input
          type="number"
          min="0"
          max="23"
          value={String(config.atHour).padStart(2, '0')}
          class="h-8 w-16 text-center font-mono text-sm"
          oninput={(e) =>
            updateConfig(
              'atHour',
              clampScheduleNumber((e.currentTarget as HTMLInputElement).value, 0, 23, 0),
            )}
        />
        <span class="text-muted-foreground text-sm">:</span>
        <Input
          type="number"
          min="0"
          max="59"
          value={String(config.atMinute).padStart(2, '0')}
          class="h-8 w-16 text-center font-mono text-sm"
          oninput={(e) =>
            updateConfig(
              'atMinute',
              clampScheduleNumber((e.currentTarget as HTMLInputElement).value, 0, 59, 0),
            )}
        />
      </div>
    {/if}
  {/if}

  <!-- Generated cron expression -->
  <div class="bg-muted/50 border-border flex items-center gap-2 rounded-md border px-3 py-2">
    <span class="text-muted-foreground shrink-0 text-[11px]">
      {i18nStore.t('settings.workflowScheduledJobCronPicker.labels.cron')}
    </span>
    <code class="text-foreground font-mono text-xs">{cronExpression}</code>
  </div>

  <!-- Next 5 triggers -->
  <div class="space-y-1.5">
    <div class="text-muted-foreground flex items-center gap-1.5 text-[11px] font-medium">
      <Clock class="size-3" />
      {i18nStore.t('settings.workflowScheduledJobCronPicker.labels.nextTriggers')}
    </div>
    {#if nextTriggers.length > 0}
      <ul class="space-y-0.5">
        {#each nextTriggers as trigger, i (i)}
          <li class="text-muted-foreground flex items-center gap-2 font-mono text-xs">
            <span class="bg-muted-foreground/30 size-1 shrink-0 rounded-full"></span>
            {formatTriggerTime(trigger)}
          </li>
        {/each}
      </ul>
    {:else if manualMode && manualValue.trim()}
      <p class="text-muted-foreground/60 text-xs">
        {i18nStore.t('settings.workflowScheduledJobCronPicker.messages.cannotPreview')}
      </p>
    {:else}
      <p class="text-muted-foreground/60 text-xs">
        {i18nStore.t('settings.workflowScheduledJobCronPicker.messages.enterCronHint')}
      </p>
    {/if}
  </div>
</div>
