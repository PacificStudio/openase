<script lang="ts">
  import type {
    NotificationChannel,
    NotificationRule,
    NotificationRuleEventType,
  } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
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
  import NotificationRuleEditor from './notification-rule-editor.svelte'

  let {
    channels,
    eventTypes,
    rules,
    onCreate,
    onUpdate,
    onDelete,
    onToggle,
  }: {
    channels: NotificationChannel[]
    eventTypes: NotificationRuleEventType[]
    rules: NotificationRule[]
    onCreate: (input: RuleCreateInput) => Promise<NotificationRule>
    onUpdate: (ruleId: string, input: RuleUpdateInput) => Promise<NotificationRule>
    onDelete: (ruleId: string) => Promise<void>
    onToggle: (ruleId: string, isEnabled: boolean) => Promise<NotificationRule>
  } = $props()

  let selectedId = $state<string>('new')
  let draft = $state<RuleDraft>(createRuleDraft([], ''))
  let saving = $state(false)
  let deleting = $state(false)
  let toggling = $state(false)

  const selectedRule = $derived(rules.find((rule) => rule.id === selectedId) ?? null)
  const canCreateRule = $derived(channels.length > 0 && eventTypes.length > 0)

  $effect(() => {
    if (selectedId !== 'new' && !selectedRule) {
      selectedId = rules[0]?.id ?? 'new'
    }

    if (selectedRule) {
      draft = ruleDraftFromRecord(selectedRule)
      return
    }

    draft = createRuleDraft(eventTypes, channels[0]?.id || '')
  })

  function selectRule(ruleId: string) {
    selectedId = ruleId
  }

  function selectNewRule() {
    selectedId = 'new'
  }
  async function handleSave() {
    if (!canCreateRule) {
      toastStore.error('Create at least one channel before managing notification rules.')
      return
    }

    if (selectedRule) {
      const parsed = buildUpdateRuleInput(draft, selectedRule)
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
        const rule = await onUpdate(selectedRule.id, parsed.value.value)
        selectedId = rule.id
        toastStore.success('Rule updated.')
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
      const rule = await onCreate(parsed.value)
      selectedId = rule.id
      toastStore.success('Rule created.')
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, 'Failed to create rule.'))
    } finally {
      saving = false
    }
  }

  async function handleDelete() {
    if (!selectedRule) return
    if (!window.confirm(`Delete rule "${selectedRule.name}"?`)) return

    deleting = true
    try {
      await onDelete(selectedRule.id)
      selectedId = 'new'
      toastStore.success('Rule deleted.')
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, 'Failed to delete rule.'))
    } finally {
      deleting = false
    }
  }

  async function handleToggle() {
    if (!selectedRule) return

    toggling = true
    try {
      const rule = await onToggle(selectedRule.id, !selectedRule.is_enabled)
      selectedId = rule.id
      toastStore.success(rule.is_enabled ? 'Rule enabled.' : 'Rule disabled.')
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, 'Failed to update rule state.'))
    } finally {
      toggling = false
    }
  }
</script>

<div class="border-border bg-card rounded-2xl border">
  <div class="border-border flex items-start justify-between gap-4 border-b px-5 py-4">
    <div>
      <h3 class="text-foreground text-base font-semibold">Project Rules</h3>
      <p class="text-muted-foreground mt-1 text-sm">
        Rules bind project event types to org channels and control when notifications fan out.
      </p>
    </div>
    <Button variant="outline" size="sm" onclick={selectNewRule} disabled={!canCreateRule}>
      New rule
    </Button>
  </div>
  <div class="grid gap-0 lg:grid-cols-[260px_minmax(0,1fr)]">
    <div class="border-border space-y-2 border-b px-4 py-4 lg:border-r lg:border-b-0">
      {#if rules.length === 0}
        <p class="text-muted-foreground text-sm">No rules yet.</p>
      {:else}
        {#each rules as rule (rule.id)}
          <button
            type="button"
            class={`w-full rounded-xl border px-3 py-3 text-left transition-colors ${
              selectedId === rule.id
                ? 'border-primary/40 bg-primary/5'
                : 'border-border hover:bg-muted/50'
            }`}
            onclick={() => selectRule(rule.id)}
          >
            <div class="flex items-center justify-between gap-2">
              <span class="text-sm font-medium">{rule.name}</span>
              <Badge variant="outline">{rule.is_enabled ? 'Enabled' : 'Disabled'}</Badge>
            </div>
            <p class="text-muted-foreground mt-1 text-xs">
              {findEventType(eventTypes, rule.event_type)?.label ?? rule.event_type}
            </p>
            <p class="text-muted-foreground mt-1 text-xs">
              Channel: {rule.channel.name}
            </p>
          </button>
        {/each}
      {/if}
    </div>
    <NotificationRuleEditor
      {channels}
      {draft}
      {eventTypes}
      {saving}
      {deleting}
      {toggling}
      {canCreateRule}
      {selectedRule}
      onDraftChange={(nextDraft: RuleDraft) => {
        draft = nextDraft
      }}
      onSave={handleSave}
      onDelete={handleDelete}
      onToggle={handleToggle}
    />
  </div>
</div>
