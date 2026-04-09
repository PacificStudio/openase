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
  import { buildEventCatalog, getSeverity } from '../notification-event-catalog'
  import NotificationRuleDialog from './notification-rule-dialog.svelte'
  import NotificationEventGroup from './notification-event-group.svelte'
  import NotificationRuleList from './notification-rule-list.svelte'

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
  let draft = $state<RuleDraft>(createRuleDraft([], ''))
  let dialogOpen = $state(false)
  let confirmDeleteOpen = $state(false)
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
    dialogOpen = true
  }

  function openEditRule(rule: NotificationRule) {
    editingRule = rule
    draft = ruleDraftFromRecord(rule)
    dialogOpen = true
  }

  function closeDialog() {
    if (saving || deleting) return
    dialogOpen = false
    editingRule = null
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
        dialogOpen = false
        editingRule = null
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
      dialogOpen = false
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, 'Failed to create rule.'))
    } finally {
      saving = false
    }
  }

  async function handleDelete() {
    if (!editingRule) return
    deleting = true
    try {
      await onDelete(editingRule.id)
      toastStore.success('Rule deleted.')
      dialogOpen = false
      confirmDeleteOpen = false
      editingRule = null
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

  function severityClass(eventType: string): string {
    const s = getSeverity(eventType, eventTypes)
    if (s === 'critical') return 'bg-red-500'
    if (s === 'warning') return 'bg-amber-500'
    return 'bg-blue-500'
  }

  function severityLabel(eventType: string): string {
    const s = getSeverity(eventType, eventTypes)
    if (s === 'critical') return 'Critical'
    if (s === 'warning') return 'Warning'
    return 'Info'
  }
</script>

<div class="space-y-6">
  <!-- Rules list header -->
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

  <NotificationRuleList
    {canCreateRule}
    {rules}
    {eventTypes}
    {togglingId}
    {severityClass}
    {severityLabel}
    onEditRule={openEditRule}
    onToggleRule={handleToggleRule}
  />

  {#if canCreateRule}
    <!-- Event catalog -->
    <div class="space-y-3">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h4 class="text-foreground text-xs font-semibold">Available events</h4>
          <p class="text-muted-foreground mt-0.5 text-xs">
            Browse events and click "+ Add rule" to subscribe.
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-3 text-xs">
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
    </div>
  {/if}
</div>

<NotificationRuleDialog
  {channels}
  {eventTypes}
  {editingRule}
  {draft}
  {dialogOpen}
  {confirmDeleteOpen}
  {saving}
  {deleting}
  onDialogOpenChange={(open) => {
    if (!open) closeDialog()
    else dialogOpen = true
  }}
  onConfirmDeleteOpenChange={(open) => (confirmDeleteOpen = open)}
  onDraftChange={(nextDraft) => {
    draft = nextDraft
  }}
  onSave={handleSave}
  onDelete={handleDelete}
/>
