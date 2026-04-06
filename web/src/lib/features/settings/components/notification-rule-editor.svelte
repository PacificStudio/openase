<script lang="ts">
  import type {
    NotificationChannel,
    NotificationRule,
    NotificationRuleEventType,
  } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import { applyRuleEventType, findEventType, type RuleDraft } from '../notification-rules'
  import { getSeverity, severityLabel, type EventSeverity } from '../notification-event-catalog'

  let {
    channels,
    eventTypes,
    selectedRule,
    draft,
    canCreateRule,
    saving = false,
    deleting = false,
    onDraftChange,
    onSave,
    onDelete,
  }: {
    channels: NotificationChannel[]
    eventTypes: NotificationRuleEventType[]
    selectedRule: NotificationRule | null
    draft: RuleDraft
    canCreateRule: boolean
    saving?: boolean
    deleting?: boolean
    onDraftChange: (draft: RuleDraft) => void
    onSave: () => void
    onDelete: () => void
  } = $props()

  const currentSeverity: EventSeverity = $derived(getSeverity(draft.eventType, eventTypes))

  function updateTextField(field: 'name' | 'filterText' | 'template', event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    onDraftChange({ ...draft, [field]: target.value })
  }
</script>

<div class="space-y-4 px-5 py-4">
  {#if !canCreateRule}
    <div class="border-border bg-muted/40 rounded-lg border px-4 py-3 text-sm">
      Add at least one notification channel to enable rule management.
    </div>
  {/if}

  <div class="grid gap-4 sm:grid-cols-2">
    <div class="space-y-1.5">
      <Label for="notification-rule-name">Rule name</Label>
      <Input
        id="notification-rule-name"
        placeholder="e.g. Alert on failures"
        value={draft.name}
        disabled={!canCreateRule}
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
            class="font-medium {currentSeverity === 'critical' ? 'text-red-500' : currentSeverity === 'warning' ? 'text-amber-500' : 'text-blue-500'}"
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
      value={draft.template}
      rows={5}
      class="font-mono text-xs"
      oninput={(event) => updateTextField('template', event)}
    />
    <p class="text-muted-foreground text-xs">
      Uses Jinja2 syntax. Variables like
      <code class="bg-muted rounded px-1">{'{{ ticket.identifier }}'}</code> are replaced at delivery.
    </p>
  </div>

  <div class="space-y-1.5">
    <Label for="notification-rule-filter">Filter (optional)</Label>
    <Textarea
      id="notification-rule-filter"
      value={draft.filterText}
      rows={4}
      class="font-mono text-xs"
      placeholder={'e.g. {"priority":"high"}'}
      oninput={(event) => updateTextField('filterText', event)}
    />
    <p class="text-muted-foreground text-xs">
      JSON object to narrow delivery. Only events matching all filter keys will trigger this rule.
    </p>
  </div>

  <div class="flex flex-wrap items-center gap-2 pt-1">
    <Button
      size="sm"
      onclick={onSave}
      disabled={!canCreateRule || saving || deleting}
    >
      {saving ? 'Saving...' : selectedRule ? 'Save changes' : 'Create rule'}
    </Button>
    {#if selectedRule}
      <Button
        variant="destructive"
        size="sm"
        onclick={onDelete}
        disabled={saving || deleting}
      >
        {deleting ? 'Deleting...' : 'Delete rule'}
      </Button>
    {/if}
  </div>
</div>
