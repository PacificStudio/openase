import { api } from '$lib/features/workspace'
import type {
  NotificationChannel,
  NotificationChannelForm,
  NotificationRule,
  NotificationRuleEventType,
  NotificationRuleForm,
} from './types'

export async function loadNotificationEventTypes() {
  const payload = await api<{ event_types: NotificationRuleEventType[] }>(
    '/api/v1/notification-event-types',
  )
  return payload.event_types
}

export async function loadNotificationChannels(organizationId: string) {
  const payload = await api<{ channels: NotificationChannel[] }>(
    `/api/v1/orgs/${organizationId}/channels`,
  )
  return payload.channels
}

export async function createNotificationChannel(
  organizationId: string,
  input: {
    name: NotificationChannelForm['name']
    type: NotificationChannelForm['type']
    config: Record<string, unknown>
    is_enabled: boolean
  },
) {
  const payload = await api<{ channel: NotificationChannel }>(
    `/api/v1/orgs/${organizationId}/channels`,
    {
      method: 'POST',
      body: JSON.stringify(input),
    },
  )
  return payload.channel
}

export async function updateNotificationChannel(channelId: string, input: Record<string, unknown>) {
  const payload = await api<{ channel: NotificationChannel }>(`/api/v1/channels/${channelId}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
  })
  return payload.channel
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
  const payload = await api<{ rules: NotificationRule[] }>(
    `/api/v1/projects/${projectId}/notification-rules`,
  )
  return payload.rules
}

export async function createNotificationRule(
  projectId: string,
  input: {
    name: NotificationRuleForm['name']
    event_type: NotificationRuleForm['eventType']
    channel_id: NotificationRuleForm['channelId']
    filter: Record<string, unknown>
    template: string
    is_enabled: boolean
  },
) {
  const payload = await api<{ rule: NotificationRule }>(
    `/api/v1/projects/${projectId}/notification-rules`,
    {
      method: 'POST',
      body: JSON.stringify(input),
    },
  )
  return payload.rule
}

export async function updateNotificationRule(ruleId: string, input: Record<string, unknown>) {
  const payload = await api<{ rule: NotificationRule }>(`/api/v1/notification-rules/${ruleId}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
  })
  return payload.rule
}

export async function deleteNotificationRule(ruleId: string) {
  await api(`/api/v1/notification-rules/${ruleId}`, { method: 'DELETE' })
}
