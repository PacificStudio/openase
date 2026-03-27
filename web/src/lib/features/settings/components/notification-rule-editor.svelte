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

  let {
    channels,
    eventTypes,
    selectedRule,
    draft,
    canCreateRule,
    saving = false,
    deleting = false,
    toggling = false,
    onDraftChange,
    onSave,
    onDelete,
    onToggle,
  }: {
    channels: NotificationChannel[]
    eventTypes: NotificationRuleEventType[]
    selectedRule: NotificationRule | null
    draft: RuleDraft
    canCreateRule: boolean
    saving?: boolean
    deleting?: boolean
    toggling?: boolean
    onDraftChange: (draft: RuleDraft) => void
    onSave: () => void
    onDelete: () => void
    onToggle: () => void
  } = $props()

  function updateTextField(field: 'name' | 'filterText' | 'template', event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    onDraftChange({ ...draft, [field]: target.value })
  }
</script>

<div class="space-y-5 px-5 py-5">
  <div class="flex flex-wrap items-start justify-between gap-3">
    <div>
      <h4 class="text-foreground text-sm font-semibold">
        {selectedRule ? selectedRule.name : 'Create rule'}
      </h4>
      <p class="text-muted-foreground mt-1 text-sm">
        Event type templates can be edited per rule. Filter JSON narrows the delivery context.
      </p>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      {#if selectedRule}
        <Button
          variant="outline"
          size="sm"
          onclick={onToggle}
          disabled={saving || deleting || toggling}
        >
          {toggling ? 'Updating…' : selectedRule.is_enabled ? 'Disable' : 'Enable'}
        </Button>
      {/if}
      <Button
        size="sm"
        onclick={onSave}
        disabled={!canCreateRule || saving || deleting || toggling}
      >
        {saving ? 'Saving…' : selectedRule ? 'Save changes' : 'Create rule'}
      </Button>
      {#if selectedRule}
        <Button
          variant="destructive"
          size="sm"
          onclick={onDelete}
          disabled={saving || deleting || toggling}
        >
          {deleting ? 'Deleting…' : 'Delete'}
        </Button>
      {/if}
    </div>
  </div>

  {#if !canCreateRule}
    <div class="border-border bg-muted/40 rounded-xl border px-4 py-3 text-sm">
      Add at least one notification channel to enable rule management.
    </div>
  {/if}

  <div class="grid gap-4 md:grid-cols-2">
    <div class="space-y-2">
      <Label for="notification-rule-name">Rule name</Label>
      <Input
        id="notification-rule-name"
        value={draft.name}
        disabled={!canCreateRule}
        oninput={(event) => updateTextField('name', event)}
      />
    </div>

    <div class="space-y-2">
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
            <Select.Item value={channel.id}>{channel.name}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>
  </div>

  <div class="grid gap-4 md:grid-cols-2">
    <div class="space-y-2">
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
    </div>

    <div class="space-y-2">
      <Label>Rule state</Label>
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

  <div class="space-y-2">
    <Label for="notification-rule-template">Template</Label>
    <Textarea
      id="notification-rule-template"
      value={draft.template}
      rows={7}
      class="font-mono text-xs"
      oninput={(event) => updateTextField('template', event)}
    />
  </div>

  <div class="space-y-2">
    <Label for="notification-rule-filter">Filter JSON</Label>
    <Textarea
      id="notification-rule-filter"
      value={draft.filterText}
      rows={8}
      class="font-mono text-xs"
      oninput={(event) => updateTextField('filterText', event)}
    />
    <p class="text-muted-foreground text-xs">
      Example: <code>{'{"priority":"high"}'}</code> or <code>{'{"new_status":"Done"}'}</code>.
    </p>
  </div>
</div>
