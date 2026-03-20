import { areJSONValuesEqual } from './parsers'
import type { JSONObject, NotificationChannel, NotificationRule } from './types'
import type { NotificationsState } from './editor'

export function buildChannelCreateInput(state: NotificationsState) {
  return {
    name: state.channelForm.name.trim(),
    type: state.channelForm.type,
    config: parseChannelConfig(state),
    is_enabled: state.channelForm.isEnabled,
  }
}

export function buildChannelUpdateInput(
  state: NotificationsState,
  channel: NotificationChannel,
): Record<string, unknown> {
  const patch: Record<string, unknown> = {}
  if (state.channelForm.name.trim() !== channel.name) {
    patch.name = state.channelForm.name.trim()
  }
  if (state.channelForm.isEnabled !== channel.is_enabled) {
    patch.is_enabled = state.channelForm.isEnabled
  }
  if (state.channelReplaceConfig) {
    if (state.channelForm.type !== channel.type) {
      patch.type = state.channelForm.type
    }
    patch.config = parseChannelConfig(state)
  }
  if (Object.keys(patch).length === 0) {
    throw new Error('No channel changes to save.')
  }
  return patch
}

export function buildRuleCreateInput(state: NotificationsState) {
  return {
    name: state.ruleForm.name.trim(),
    event_type: state.ruleForm.eventType,
    channel_id: state.ruleForm.channelId,
    filter: parseJSONObject(state.ruleForm.filterText, 'Rule filter'),
    template: state.ruleForm.template.trim(),
    is_enabled: state.ruleForm.isEnabled,
  }
}

export function buildRuleUpdateInput(
  state: NotificationsState,
  rule: NotificationRule,
): Record<string, unknown> {
  const parsedFilter = parseJSONObject(state.ruleForm.filterText, 'Rule filter')
  const patch: Record<string, unknown> = {}
  if (state.ruleForm.name.trim() !== rule.name) {
    patch.name = state.ruleForm.name.trim()
  }
  if (state.ruleForm.eventType !== rule.event_type) {
    patch.event_type = state.ruleForm.eventType
  }
  if (state.ruleForm.channelId !== rule.channel_id) {
    patch.channel_id = state.ruleForm.channelId
  }
  if (!areJSONValuesEqual(parsedFilter, rule.filter)) {
    patch.filter = parsedFilter
  }
  if (state.ruleForm.template.trim() !== rule.template) {
    patch.template = state.ruleForm.template.trim()
  }
  if (state.ruleForm.isEnabled !== rule.is_enabled) {
    patch.is_enabled = state.ruleForm.isEnabled
  }
  if (Object.keys(patch).length === 0) {
    throw new Error('No rule changes to save.')
  }
  return patch
}

function parseChannelConfig(state: NotificationsState): JSONObject {
  switch (state.channelForm.type) {
    case 'webhook':
      return {
        url: requireNonEmpty(state.channelForm.webhookURL, 'Webhook URL'),
        headers: parseJSONObject(state.channelForm.webhookHeaders, 'Webhook headers'),
        secret: state.channelForm.webhookSecret.trim(),
      }
    case 'telegram':
      return {
        bot_token: requireNonEmpty(state.channelForm.telegramBotToken, 'Telegram bot token'),
        chat_id: requireNonEmpty(state.channelForm.telegramChatID, 'Telegram chat ID'),
      }
    case 'slack':
      return {
        webhook_url: requireNonEmpty(state.channelForm.slackWebhookURL, 'Slack webhook URL'),
      }
    case 'wecom':
      return {
        webhook_key: requireNonEmpty(state.channelForm.wecomWebhookKey, 'WeCom webhook key'),
      }
  }
}

function parseJSONObject(raw: string, label: string) {
  const trimmed = raw.trim()
  if (!trimmed) {
    return {}
  }

  let parsed: unknown
  try {
    parsed = JSON.parse(trimmed)
  } catch {
    throw new Error(`${label} must be valid JSON.`)
  }
  if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
    throw new Error(`${label} must be a JSON object.`)
  }
  return parsed as JSONObject
}

function requireNonEmpty(value: string, label: string) {
  const trimmed = value.trim()
  if (!trimmed) {
    throw new Error(`${label} is required.`)
  }
  return trimmed
}
