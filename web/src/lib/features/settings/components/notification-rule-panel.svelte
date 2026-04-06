<script lang="ts">
  import type {
    NotificationChannel,
    NotificationRule,
    NotificationRuleEventType,
  } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import {
    buildCreateRuleInput,
    buildUpdateRuleInput,
    createRuleDraft,
    findEventType,
    ruleDraftFromRecord,
    type RuleCreateInput,
    type RuleDraft,
    type RuleUpdateInput,
  } from '../notification-rules'
  import { actionErrorMessage } from '../notification-support'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { buildEventCatalog } from '../notification-event-catalog'
  import NotificationRuleEditor from './notification-rule-editor.svelte'
  import NotificationEventGroup from './notification-event-group.svelte'

  let {
    channels,
    eventTypes,
    rules,
    onCreate,
    onUpdate,
    onDelete,
  }: {
    channels: NotificationChannel[]
    eventTypes: NotificationRuleEventType[]
    rules: NotificationRule[]
    onCreate: (input: RuleCreateInput) => Promise<NotificationRule>
    onUpdate: (ruleId: string, input: RuleUpdateInput) => Promise<NotificationRule>
    onDelete: (ruleId: string) => Promise<void>
  } = $props()

  let editingRule = $state<NotificationRule | null>(null)
  let creatingNew = $state(false)
  let draft = $state<RuleDraft>(createRuleDraft([], ''))
  let saving = $state(false)
  let deleting = $state(false)
  let togglingId = $state<string | null>(null)
  let expandedGroups = $state<Set<string>>(new Set())

  const catalog = $derived(buildEventCatalog(eventTypes))
  const canCreateRule = $derived(channels.length > 0 && eventTypes.length > 0)

  function toggleGroup(groupKey: string) {
    const next = new Set(expandedGroups)
    if (next.has(groupKey)) {
      next.delete(groupKey)
    } else {
      next.add(groupKey)
    }
    expandedGroups = next
  }

  function openNewRule(eventType?: string) {
    editingRule = null
    creatingNew = true
    const d = createRuleDraft(eventTypes, channels[0]?.id || '')
    if (eventType) {
      const et = findEventType(eventTypes, eventType)
      if (et) {
        d.eventType = eventType
        d.template = et.default_template
        d.name = et.label
      }
    }
    draft = d
  }

  function openEditRule(rule: NotificationRule) {
    editingRule = rule
    creatingNew = true
    draft = ruleDraftFromRecord(rule)
  }

  function closeEditor() {
    editingRule = null
    creatingNew = false
  }

  async function handleSave() {
    if (editingRule) {
      const parsed = buildUpdateRuleInput(draft, editingRule)
      if (!parsed.ok) {
        toastStore.error(parsed.error)
        return
      }
      if (!parsed.value.changed) {
        toastStore.info('No rule changes to save.')
        return
      }

      saving = true
      try {
        await onUpdate(editingRule.id, parsed.value.value)
        toastStore.success('Rule updated.')
        closeEditor()
      } catch (caughtError) {
        toastStore.error(actionErrorMessage(caughtError, 'Failed to update rule.'))
      } finally {
        saving = false
      }
      return
    }

    const parsed = buildCreateRuleInput(draft)
    if (!parsed.ok) {
      toastStore.error(parsed.error)
      return
    }

    saving = true
    try {
      await onCreate(parsed.value)
      toastStore.success('Rule created.')
      closeEditor()
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, 'Failed to create rule.'))
    } finally {
      saving = false
    }
  }

  async function handleDelete() {
    if (!editingRule) return
    if (!window.confirm(`Delete rule "${editingRule.name}"?`)) return

    deleting = true
    try {
      await onDelete(editingRule.id)
      toastStore.success('Rule deleted.')
      closeEditor()
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, 'Failed to delete rule.'))
    } finally {
      deleting = false
    }
  }

  async function handleToggleRule(rule: NotificationRule) {
    togglingId = rule.id
    try {
      if (rule.is_enabled) {
        await onDelete(rule.id)
        toastStore.success('Rule disabled.')
      } else {
        await onUpdate(rule.id, { is_enabled: true })
        toastStore.success('Rule enabled.')
      }
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, 'Failed to update rule state.'))
    } finally {
      togglingId = null
    }
  }
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between gap-4">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Notification Rules</h3>
      <p class="text-muted-foreground mt-0.5 text-xs">
        Subscribe to project events and route them to channels.
      </p>
    </div>
    {#if canCreateRule}
      <Button variant="outline" size="sm" onclick={() => openNewRule()}>Add rule</Button>
    {/if}
  </div>

  {#if !canCreateRule}
    <div
      class="border-border bg-muted/30 flex flex-col items-center gap-2 rounded-lg border border-dashed px-6 py-8 text-center"
    >
      <p class="text-muted-foreground text-sm">Add a channel first to create notification rules.</p>
    </div>
  {:else if rules.length === 0 && !creatingNew}
    <div
      class="border-border bg-muted/30 flex flex-col items-center gap-2 rounded-lg border border-dashed px-6 py-8 text-center"
    >
      <p class="text-muted-foreground text-sm">No notification rules yet.</p>
      <p class="text-muted-foreground text-xs">
        Rules connect project events to channels. Expand event groups below to get started.
      </p>
    </div>
  {/if}

  {#if canCreateRule}
    <div class="flex flex-wrap items-center gap-3 text-xs">
      <span class="text-muted-foreground">Severity:</span>
      <span class="flex items-center gap-1">
        <span class="size-2 rounded-full bg-blue-500"></span>
        <span class="text-muted-foreground">Info</span>
      </span>
      <span class="flex items-center gap-1">
        <span class="size-2 rounded-full bg-amber-500"></span>
        <span class="text-muted-foreground">Warning</span>
      </span>
      <span class="flex items-center gap-1">
        <span class="size-2 rounded-full bg-red-500"></span>
        <span class="text-muted-foreground">Critical</span>
      </span>
    </div>

    <div class="space-y-2">
      {#each catalog as group (group.key)}
        <NotificationEventGroup
          {group}
          expanded={expandedGroups.has(group.key)}
          {rules}
          {togglingId}
          onToggleGroup={toggleGroup}
          onToggleRule={handleToggleRule}
          onNewRule={openNewRule}
          onEditRule={openEditRule}
        />
      {/each}
    </div>
  {/if}

  {#if creatingNew && canCreateRule}
    <div class="border-border bg-card rounded-lg border">
      <div class="flex items-center justify-between border-b border-border/50 px-5 py-3">
        <h4 class="text-sm font-medium">
          {editingRule ? `Edit: ${editingRule.name}` : 'New rule'}
        </h4>
        <Button variant="ghost" size="sm" class="h-7 px-2 text-xs" onclick={closeEditor}>
          Cancel
        </Button>
      </div>
      <NotificationRuleEditor
        {channels}
        {draft}
        {eventTypes}
        {saving}
        {deleting}
        {canCreateRule}
        selectedRule={editingRule}
        onDraftChange={(nextDraft: RuleDraft) => {
          draft = nextDraft
        }}
        onSave={handleSave}
        onDelete={handleDelete}
      />
    </div>
  {/if}
</div>
