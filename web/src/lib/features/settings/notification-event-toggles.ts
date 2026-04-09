import type {
  NotificationChannel,
  NotificationRule,
  NotificationRuleEventType,
} from '$lib/api/contracts'

import type { RuleCreateInput } from './notification-rules'

export type NotificationEventGroup = {
  group: string
  events: NotificationRuleEventType[]
}

export function groupNotificationEventTypes(
  eventTypes: NotificationRuleEventType[],
): NotificationEventGroup[] {
  const groups = new Map<string, NotificationRuleEventType[]>()

  for (const eventType of eventTypes) {
    const group = eventType.group || 'Other'
    const items = groups.get(group) ?? []
    items.push(eventType)
    groups.set(group, items)
  }

  return Array.from(groups.entries()).map(([group, events]) => ({ group, events }))
}

export function findNotificationToggleRule(
  rules: NotificationRule[],
  channelId: string,
  eventType: string,
): NotificationRule | undefined {
  return rules.find((rule) => rule.channel_id === channelId && rule.event_type === eventType)
}

export function buildNotificationToggleRuleInput(
  eventType: NotificationRuleEventType,
  channel: NotificationChannel,
): RuleCreateInput {
  return {
    name: `${eventType.label} via ${channel.name}`,
    event_type: eventType.event_type,
    filter: {},
    channel_id: channel.id,
    template: '',
    is_enabled: true,
  }
}
