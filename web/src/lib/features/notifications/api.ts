import { api } from '$lib/features/workspace'
import type {
  JSONObject,
  NotificationChannelForm,
  NotificationRuleForm,
} from './types'
import {
  parseNotificationChannel,
  parseNotificationRule,
  parseNotificationRuleEventType,
} from './parsers'

export async function loadNotificationEventTypes() {
  const payload = await api<{ event_types: unknown }>('/api/v1/notification-event-types')
  return parseList(payload.event_types, 'event_types', parseNotificationRuleEventType)
}

export async function loadNotificationChannels(organizationId: string) {
  const payload = await api<{ channels: unknown }>(`/api/v1/orgs/${organizationId}/channels`)
  return parseList(payload.channels, 'channels', parseNotificationChannel)
}

export async function createNotificationChannel(
  organizationId: string,
  input: {
    name: NotificationChannelForm['name']
    type: NotificationChannelForm['type']
    config: JSONObject
    is_enabled: boolean
  },
) {
  const payload = await api<{ channel: unknown }>(`/api/v1/orgs/${organizationId}/channels`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
  return parseNotificationChannel(payload.channel)
}

export async function updateNotificationChannel(channelId: string, input: Record<string, unknown>) {
  const payload = await api<{ channel: unknown }>(`/api/v1/channels/${channelId}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
  })
  return parseNotificationChannel(payload.channel)
}

export async function deleteNotificationChannel(channelId: string) {
  await api(`/api/v1/channels/${channelId}`, { method: 'DELETE' })
}

export async function testNotificationChannel(channelId: string) {
  const payload = await api<{ status: string }>(`/api/v1/channels/${channelId}/test`, {
    method: 'POST',
  })
  return payload.status
}

export async function loadNotificationRules(projectId: string) {
  const payload = await api<{ rules: unknown }>(`/api/v1/projects/${projectId}/notification-rules`)
  return parseList(payload.rules, 'rules', parseNotificationRule)
}

export async function createNotificationRule(
  projectId: string,
  input: {
    name: NotificationRuleForm['name']
    event_type: NotificationRuleForm['eventType']
    channel_id: NotificationRuleForm['channelId']
    filter: JSONObject
    template: string
    is_enabled: boolean
  },
) {
  const payload = await api<{ rule: unknown }>(`/api/v1/projects/${projectId}/notification-rules`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
  return parseNotificationRule(payload.rule)
}

export async function updateNotificationRule(ruleId: string, input: Record<string, unknown>) {
  const payload = await api<{ rule: unknown }>(`/api/v1/notification-rules/${ruleId}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
  })
  return parseNotificationRule(payload.rule)
}

export async function deleteNotificationRule(ruleId: string) {
  await api(`/api/v1/notification-rules/${ruleId}`, { method: 'DELETE' })
}

function parseList<T>(raw: unknown, label: string, parseItem: (value: unknown) => T) {
  if (!Array.isArray(raw)) {
    throw new Error(`Expected ${label} to be an array.`)
  }

  return raw.map((item) => parseItem(item))
}
