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
  import { i18nStore } from '$lib/i18n/store.svelte'

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
        toastStore.info(i18nStore.t('settings.notificationRule.messages.noChanges'))
        return
      }
      saving = true
      try {
        await onUpdate(editingRule.id, parsed.value.value)
        toastStore.success(i18nStore.t('settings.notificationRule.messages.updated'))
        dialogOpen = false
        editingRule = null
      } catch (caughtError) {
        toastStore.error(
          actionErrorMessage(caughtError, i18nStore.t('settings.notificationRule.errors.update')),
        )
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
      toastStore.success(i18nStore.t('settings.notificationRule.messages.created'))
      dialogOpen = false
    } catch (caughtError) {
      toastStore.error(
        actionErrorMessage(caughtError, i18nStore.t('settings.notificationRule.errors.create')),
      )
    } finally {
      saving = false
    }
  }

  async function handleDelete() {
    if (!editingRule) return
    deleting = true
    try {
      await onDelete(editingRule.id)
      toastStore.success(i18nStore.t('settings.notificationRule.messages.deleted'))
      dialogOpen = false
      confirmDeleteOpen = false
      editingRule = null
    } catch (caughtError) {
      toastStore.error(
        actionErrorMessage(caughtError, i18nStore.t('settings.notificationRule.errors.delete')),
      )
    } finally {
      deleting = false
    }
  }

  async function handleToggleRule(rule: NotificationRule) {
    togglingId = rule.id
    try {
      if (rule.is_enabled) {
        await onDelete(rule.id)
        toastStore.success(i18nStore.t('settings.notificationRule.messages.disabled'))
      } else {
        await onUpdate(rule.id, { is_enabled: true })
        toastStore.success(i18nStore.t('settings.notificationRule.messages.enabled'))
      }
    } catch (caughtError) {
      toastStore.error(
        actionErrorMessage(caughtError, i18nStore.t('settings.notificationRule.errors.state')),
      )
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
    if (s === 'critical') return i18nStore.t('settings.notificationRule.severity.critical')
    if (s === 'warning') return i18nStore.t('settings.notificationRule.severity.warning')
    return i18nStore.t('settings.notificationRule.severity.info')
  }
</script>

<div class="space-y-6">
  <!-- Rules list header -->
  <div class="flex items-center justify-between gap-4">
    <div>
      <h3 class="text-foreground text-sm font-semibold">
        {i18nStore.t('settings.notificationRule.heading')}
      </h3>
      <p class="text-muted-foreground mt-0.5 text-xs">
        {i18nStore.t('settings.notificationRule.description')}
      </p>
    </div>
    {#if canCreateRule}
      <Button variant="outline" size="sm" onclick={() => openNewRule()}>
        {i18nStore.t('settings.notificationRule.buttons.addRule')}
      </Button>
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
          <h4 class="text-foreground text-xs font-semibold">
            {i18nStore.t('settings.notificationRule.catalog.heading')}
          </h4>
          <p class="text-muted-foreground mt-0.5 text-xs">
            {i18nStore.t('settings.notificationRule.catalog.description', {
              action: `+ ${i18nStore.t('settings.notificationRule.buttons.addRule')}`,
            })}
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-3 text-xs">
          <span class="flex items-center gap-1">
            <span class="size-2 rounded-full bg-blue-500"></span>
            <span class="text-muted-foreground">
              {i18nStore.t('settings.notificationRule.severity.info')}
            </span>
          </span>
          <span class="flex items-center gap-1">
            <span class="size-2 rounded-full bg-amber-500"></span>
            <span class="text-muted-foreground">
              {i18nStore.t('settings.notificationRule.severity.warning')}
            </span>
          </span>
          <span class="flex items-center gap-1">
            <span class="size-2 rounded-full bg-red-500"></span>
            <span class="text-muted-foreground">
              {i18nStore.t('settings.notificationRule.severity.critical')}
            </span>
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
