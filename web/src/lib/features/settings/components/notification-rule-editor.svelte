<script lang="ts">
  import type { NotificationChannel, NotificationRuleEventType } from '$lib/api/contracts'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import { applyRuleEventType, findEventType, type RuleDraft } from '../notification-rules'
  import {
    getSeverity,
    severityLabel,
    getTemplateVariables,
    type EventSeverity,
  } from '../notification-event-catalog'

  let {
    channels,
    eventTypes,
    draft,
    onDraftChange,
  }: {
    channels: NotificationChannel[]
    eventTypes: NotificationRuleEventType[]
    draft: RuleDraft
    onDraftChange: (draft: RuleDraft) => void
  } = $props()

  const currentSeverity: EventSeverity = $derived(getSeverity(draft.eventType, eventTypes))
  const templateVarGroups = $derived(draft.eventType ? getTemplateVariables(draft.eventType) : [])

  let showVariables = $state(false)
  let templateRef = $state<HTMLTextAreaElement | null>(null)

  function updateTextField(field: 'name' | 'filterText' | 'template', event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    onDraftChange({ ...draft, [field]: target.value })
  }

  function insertVariable(varName: string) {
    const snippet = `{{ ${varName} }}`
    const current = draft.template || ''
    if (templateRef) {
      const start = templateRef.selectionStart ?? current.length
      const end = templateRef.selectionEnd ?? current.length
      const newValue = current.slice(0, start) + snippet + current.slice(end)
      onDraftChange({ ...draft, template: newValue })
      requestAnimationFrame(() => {
        templateRef?.setSelectionRange(start + snippet.length, start + snippet.length)
        templateRef?.focus()
      })
    } else {
      onDraftChange({ ...draft, template: current + snippet })
    }
  }
</script>

<div class="space-y-4">
  <div class="grid gap-4 sm:grid-cols-2">
    <div class="space-y-1.5">
      <Label for="notification-rule-name">Rule name</Label>
      <Input
        id="notification-rule-name"
        placeholder="e.g. Alert on failures"
        value={draft.name}
        oninput={(event) => updateTextField('name', event)}
      />
    </div>

    <div class="space-y-1.5">
      <Label>Channel</Label>
      <Select.Root
        type="single"
        value={draft.channelId}
        onValueChange={(value) => {
          onDraftChange({ ...draft, channelId: value || '' })
        }}
      >
        <Select.Trigger class="w-full">
          {channels.find((channel) => channel.id === draft.channelId)?.name ?? 'Select channel'}
        </Select.Trigger>
        <Select.Content>
          {#each channels as channel (channel.id)}
            <Select.Item value={channel.id}>
              <span class="flex items-center gap-2">
                {channel.name}
                {#if !channel.is_enabled}
                  <span class="text-muted-foreground text-xs">(disabled)</span>
                {/if}
              </span>
            </Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>
  </div>

  <div class="grid gap-4 sm:grid-cols-2">
    <div class="space-y-1.5">
      <Label>Event type</Label>
      <Select.Root
        type="single"
        value={draft.eventType}
        onValueChange={(value) => {
          onDraftChange(applyRuleEventType(draft, value || '', eventTypes))
        }}
      >
        <Select.Trigger class="w-full">
          {findEventType(eventTypes, draft.eventType)?.label ?? 'Select event type'}
        </Select.Trigger>
        <Select.Content>
          {#each eventTypes as eventType (eventType.event_type)}
            <Select.Item value={eventType.event_type}>{eventType.label}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
      {#if draft.eventType}
        <p class="text-xs">
          Severity:
          <span
            class="font-medium {currentSeverity === 'critical'
              ? 'text-red-500'
              : currentSeverity === 'warning'
                ? 'text-amber-500'
                : 'text-blue-500'}"
          >
            {severityLabel(currentSeverity)}
          </span>
        </p>
      {/if}
    </div>

    <div class="space-y-1.5">
      <Label>Initial state</Label>
      <Select.Root
        type="single"
        value={draft.isEnabled ? 'enabled' : 'disabled'}
        onValueChange={(value) => {
          onDraftChange({ ...draft, isEnabled: value !== 'disabled' })
        }}
      >
        <Select.Trigger class="w-full capitalize">
          {draft.isEnabled ? 'Enabled' : 'Disabled'}
        </Select.Trigger>
        <Select.Content>
          <Select.Item value="enabled">Enabled</Select.Item>
          <Select.Item value="disabled">Disabled</Select.Item>
        </Select.Content>
      </Select.Root>
    </div>
  </div>

  <div class="space-y-1.5">
    <Label for="notification-rule-template">Message template</Label>
    <Textarea
      id="notification-rule-template"
      bind:ref={templateRef}
      value={draft.template}
      rows={5}
      class="font-mono text-xs"
      oninput={(event) => updateTextField('template', event)}
    />
    <div class="flex items-center justify-between">
      <p class="text-muted-foreground text-xs">
        Jinja2 syntax — e.g.
        <code class="bg-muted rounded px-1">{'{{ ticket.identifier }}'}</code>
      </p>
      {#if templateVarGroups.length > 0}
        <button
          type="button"
          class="text-muted-foreground hover:text-foreground text-xs underline-offset-2 hover:underline"
          onclick={() => (showVariables = !showVariables)}
        >
          {showVariables ? 'Hide variables' : 'Available variables'}
        </button>
      {/if}
    </div>

    {#if showVariables && templateVarGroups.length > 0}
      <div class="border-border bg-muted/30 space-y-3 rounded-md border p-3">
        {#each templateVarGroups as group (group.label)}
          <div class="space-y-1.5">
            <p class="text-muted-foreground text-xs font-medium tracking-wide uppercase">
              {group.label}
            </p>
            <div class="flex flex-wrap gap-1.5">
              {#each group.variables as variable (variable.name)}
                <button
                  type="button"
                  title={variable.description}
                  class="bg-background border-border hover:bg-accent hover:border-ring inline-flex cursor-pointer items-center rounded border px-1.5 py-0.5 font-mono text-xs transition-colors"
                  onclick={() => insertVariable(variable.name)}
                >
                  {variable.name}
                </button>
              {/each}
            </div>
          </div>
        {/each}
        <p class="text-muted-foreground text-xs">Click a variable to insert it at the cursor.</p>
      </div>
    {/if}
  </div>

  <div class="space-y-1.5">
    <Label for="notification-rule-filter">Filter (optional)</Label>
    <Textarea
      id="notification-rule-filter"
      value={draft.filterText}
      rows={3}
      class="font-mono text-xs"
      placeholder={'e.g. {"priority":"high"}'}
      oninput={(event) => updateTextField('filterText', event)}
    />
    <p class="text-muted-foreground text-xs">
      JSON object to narrow delivery. Only events matching all filter keys will trigger this rule.
    </p>
  </div>
</div>
