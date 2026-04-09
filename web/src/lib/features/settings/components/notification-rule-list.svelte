<script lang="ts">
  import type { NotificationRule, NotificationRuleEventType } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import { Switch } from '$ui/switch'
  import { findEventType } from '../notification-rules'

  let {
    canCreateRule,
    rules,
    eventTypes,
    togglingId,
    severityClass,
    severityLabel,
    onEditRule,
    onToggleRule,
  }: {
    canCreateRule: boolean
    rules: NotificationRule[]
    eventTypes: NotificationRuleEventType[]
    togglingId: string | null
    severityClass: (eventType: string) => string
    severityLabel: (eventType: string) => string
    onEditRule: (rule: NotificationRule) => void
    onToggleRule: (rule: NotificationRule) => void
  } = $props()
</script>

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
  <div class="border-border bg-card rounded-md border">
    <table class="w-full text-sm">
      <thead>
        <tr class="border-b">
          <th class="text-muted-foreground px-4 py-2.5 text-left text-xs font-medium">Rule</th>
          <th
            class="text-muted-foreground hidden px-4 py-2.5 text-left text-xs font-medium sm:table-cell"
          >
            Event
          </th>
          <th
            class="text-muted-foreground hidden px-4 py-2.5 text-left text-xs font-medium md:table-cell"
          >
            Channel
          </th>
          <th
            class="text-muted-foreground hidden px-4 py-2.5 text-left text-xs font-medium lg:table-cell"
          >
            Severity
          </th>
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
                <span class="size-2 shrink-0 rounded-full {severityClass(rule.event_type)}"></span>
                {severityLabel(rule.event_type)}
              </span>
            </td>
            <td class="px-4 py-3">
              <div class="flex items-center justify-end gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  class="h-7 px-2 text-xs"
                  onclick={() => onEditRule(rule)}
                >
                  Edit
                </Button>
                <Switch
                  checked={rule.is_enabled}
                  disabled={togglingId === rule.id}
                  onCheckedChange={() => onToggleRule(rule)}
                />
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}
