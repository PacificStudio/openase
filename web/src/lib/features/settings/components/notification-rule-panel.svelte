<script lang="ts">
  import type {
    NotificationChannel,
    NotificationRule,
    NotificationRuleEventType,
  } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import { Badge } from '$ui/badge'
  import { Switch } from '$ui/switch'
  import * as Dialog from '$ui/dialog'
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

  <!-- Rules list (when rules exist) or empty state -->
  {#if !canCreateRule}
    <div
      class="border-border bg-muted/30 flex flex-col items-center gap-2 rounded-lg border border-dashed px-6 py-8 text-center"
    >
      <p class="text-muted-foreground text-sm">Add a channel first to create notification rules.</p>
    </div>
  {:else if rules.length === 0}
    <div
      class="border-border bg-muted/30 flex flex-col items-center gap-2 rounded-lg border border-dashed px-6 py-8 text-center"
    >
      <p class="text-muted-foreground text-sm">No notification rules yet.</p>
      <p class="text-muted-foreground text-xs">
        Browse available events below and click "+ Add rule" to subscribe.
      </p>
    </div>
  {:else}
    <!-- Rules table -->
    <div class="border-border bg-card rounded-md border">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b">
            <th class="text-muted-foreground px-4 py-2.5 text-left text-xs font-medium">Rule</th>
            <th
              class="text-muted-foreground hidden px-4 py-2.5 text-left text-xs font-medium sm:table-cell"
              >Event</th
            >
            <th
              class="text-muted-foreground hidden px-4 py-2.5 text-left text-xs font-medium md:table-cell"
              >Channel</th
            >
            <th
              class="text-muted-foreground hidden px-4 py-2.5 text-left text-xs font-medium lg:table-cell"
              >Severity</th
            >
            <th class="px-4 py-2.5 text-right"></th>
          </tr>
        </thead>
        <tbody>
          {#each rules as rule (rule.id)}
            {@const et = findEventType(eventTypes, rule.event_type)}
            <tr class="border-b last:border-0">
              <td class="px-4 py-3">
                <div class="font-medium">{rule.name}</div>
                <div class="text-muted-foreground mt-0.5 block text-xs sm:hidden">
                  {et?.label ?? rule.event_type}
                  {#if !rule.channel.is_enabled}
                    <span class="text-amber-500">· channel disabled</span>
                  {/if}
                </div>
              </td>
              <td class="text-muted-foreground hidden px-4 py-3 text-xs sm:table-cell">
                {et?.label ?? rule.event_type}
                {#if !rule.channel.is_enabled}
                  <div class="mt-0.5 text-amber-500">channel disabled</div>
                {/if}
              </td>
              <td class="text-muted-foreground hidden px-4 py-3 text-xs md:table-cell">
                {rule.channel.name}
              </td>
              <td class="hidden px-4 py-3 lg:table-cell">
                <span class="flex items-center gap-1.5 text-xs">
                  <span class="size-2 shrink-0 rounded-full {severityClass(rule.event_type)}"
                  ></span>
                  {severityLabel(rule.event_type)}
                </span>
              </td>
              <td class="px-4 py-3">
                <div class="flex items-center justify-end gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    class="h-7 px-2 text-xs"
                    onclick={() => openEditRule(rule)}
                  >
                    Edit
                  </Button>
                  <Switch
                    checked={rule.is_enabled}
                    disabled={togglingId === rule.id}
                    onCheckedChange={() => handleToggleRule(rule)}
                  />
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}

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

<!-- Rule create / edit dialog -->
<Dialog.Root
  bind:open={dialogOpen}
  onOpenChange={(open) => {
    if (!open) closeDialog()
  }}
>
  <Dialog.Content class="flex flex-col sm:max-w-2xl">
    <Dialog.Header>
      <Dialog.Title>{editingRule ? 'Edit rule' : 'New rule'}</Dialog.Title>
      {#if editingRule}
        <Dialog.Description>{editingRule.name}</Dialog.Description>
      {:else}
        <Dialog.Description>Route a project event to a notification channel.</Dialog.Description>
      {/if}
    </Dialog.Header>

    <div class="min-h-0 overflow-y-auto">
      <NotificationRuleEditor
        {channels}
        {draft}
        {eventTypes}
        selectedRule={editingRule}
        onDraftChange={(nextDraft: RuleDraft) => {
          draft = nextDraft
        }}
      />
    </div>

    <Dialog.Footer>
      {#if editingRule}
        <Button
          variant="destructive"
          onclick={() => (confirmDeleteOpen = true)}
          disabled={saving || deleting}
          class="mr-auto"
        >
          Delete
        </Button>
      {/if}
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={saving || deleting}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button onclick={handleSave} disabled={saving || deleting}>
        {saving ? 'Saving…' : editingRule ? 'Save changes' : 'Create rule'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<!-- Delete confirmation dialog -->
<Dialog.Root bind:open={confirmDeleteOpen}>
  <Dialog.Content class="sm:max-w-sm">
    <Dialog.Header>
      <Dialog.Title>Delete "{editingRule?.name}"?</Dialog.Title>
      <Dialog.Description>
        This rule will stop delivering notifications. This cannot be undone.
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={deleting}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button variant="destructive" onclick={handleDelete} disabled={deleting}>
        {deleting ? 'Deleting…' : 'Delete rule'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
