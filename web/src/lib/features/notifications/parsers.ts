import type {
  JSONArray,
  JSONValue,
  JSONObject,
  NotificationChannel,
  NotificationChannelType,
  NotificationRule,
  NotificationRuleEventType,
  SlackChannelConfig,
  TelegramChannelConfig,
  WebhookChannelConfig,
  WeComChannelConfig,
} from './types'

export function parseNotificationChannel(raw: unknown): NotificationChannel {
  const source = expectRecord(raw, 'channel')
  const type = expectChannelType(source.type, 'channel.type')
  const base = {
    id: expectString(source.id, 'channel.id'),
    organization_id: expectString(source.organization_id, 'channel.organization_id'),
    name: expectString(source.name, 'channel.name'),
    is_enabled: expectBoolean(source.is_enabled, 'channel.is_enabled'),
    created_at: expectString(source.created_at, 'channel.created_at'),
  }

  switch (type) {
    case 'webhook':
      return { ...base, type, config: parseWebhookChannelConfig(source.config) }
    case 'telegram':
      return { ...base, type, config: parseTelegramChannelConfig(source.config) }
    case 'slack':
      return { ...base, type, config: parseSlackChannelConfig(source.config) }
    case 'wecom':
      return { ...base, type, config: parseWeComChannelConfig(source.config) }
  }
}

export function parseNotificationRuleEventType(raw: unknown): NotificationRuleEventType {
  const source = expectRecord(raw, 'notification event type')
  return {
    event_type: expectString(source.event_type, 'notification event type.event_type'),
    label: expectString(source.label, 'notification event type.label'),
    default_template: expectString(
      source.default_template,
      'notification event type.default_template',
    ),
  }
}

export function parseNotificationRule(raw: unknown): NotificationRule {
  const source = expectRecord(raw, 'notification rule')
  return {
    id: expectString(source.id, 'notification rule.id'),
    project_id: expectString(source.project_id, 'notification rule.project_id'),
    channel_id: expectString(source.channel_id, 'notification rule.channel_id'),
    name: expectString(source.name, 'notification rule.name'),
    event_type: expectString(source.event_type, 'notification rule.event_type'),
    filter: parseOptionalJSONObject(source.filter) ?? {},
    template: expectString(source.template, 'notification rule.template'),
    is_enabled: expectBoolean(source.is_enabled, 'notification rule.is_enabled'),
    created_at: expectString(source.created_at, 'notification rule.created_at'),
    channel: parseNotificationChannel(source.channel),
  }
}

export function areJSONValuesEqual(left: JSONValue, right: JSONValue): boolean {
  if (Array.isArray(left)) {
    return Array.isArray(right) && arraysEqual(left, right)
  }

  if (Array.isArray(right)) {
    return false
  }

  if (isJSONObject(left)) {
    return isJSONObject(right) && objectsEqual(left, right)
  }

  if (isJSONObject(right)) {
    return false
  }

  return left === right
}

function arraysEqual(left: JSONArray, right: JSONArray) {
  return left.length === right.length && left.every((item, index) => areJSONValuesEqual(item, right[index]))
}

function objectsEqual(left: JSONObject, right: JSONObject) {
  const leftKeys = Object.keys(left).sort()
  const rightKeys = Object.keys(right).sort()
  if (leftKeys.length !== rightKeys.length) {
    return false
  }

  return leftKeys.every((key, index) => {
    if (key !== rightKeys[index]) {
      return false
    }

    const rightValue = right[key]
    if (rightValue === undefined) {
      return false
    }

    return areJSONValuesEqual(left[key], rightValue)
  })
}

function parseWebhookChannelConfig(raw: unknown): WebhookChannelConfig {
  const source = expectRecord(raw, 'webhook channel.config')
  return {
    url: readString(source, 'url'),
    headers: parseOptionalJSONObject(source.headers) ?? {},
    secret: readString(source, 'secret'),
  }
}

function parseTelegramChannelConfig(raw: unknown): TelegramChannelConfig {
  const source = expectRecord(raw, 'telegram channel.config')
  return {
    bot_token: readString(source, 'bot_token'),
    chat_id: readString(source, 'chat_id'),
  }
}

function parseSlackChannelConfig(raw: unknown): SlackChannelConfig {
  const source = expectRecord(raw, 'slack channel.config')
  return {
    webhook_url: readString(source, 'webhook_url'),
  }
}

function parseWeComChannelConfig(raw: unknown): WeComChannelConfig {
  const source = expectRecord(raw, 'wecom channel.config')
  return {
    webhook_key: readString(source, 'webhook_key'),
  }
}

function expectRecord(value: unknown, label: string) {
  if (!isJSONObject(value)) {
    throw new Error(`Expected ${label} to be an object.`)
  }

  return value
}

function expectString(value: unknown, label: string) {
  if (typeof value !== 'string') {
    throw new Error(`Expected ${label} to be a string.`)
  }

  return value
}

function readString(source: Record<string, unknown>, key: string) {
  const value = source[key]
  return typeof value === 'string' ? value : undefined
}

function expectBoolean(value: unknown, label: string) {
  if (typeof value !== 'boolean') {
    throw new Error(`Expected ${label} to be a boolean.`)
  }

  return value
}

function expectChannelType(value: unknown, label: string): NotificationChannelType {
  if (value === 'webhook' || value === 'telegram' || value === 'slack' || value === 'wecom') {
    return value
  }

  throw new Error(`Expected ${label} to be a supported notification channel type.`)
}

function parseOptionalJSONObject(value: unknown): JSONObject | undefined {
  if (!isJSONObject(value)) {
    return undefined
  }

  const parsed: JSONObject = {}
  for (const [key, entry] of Object.entries(value)) {
    const jsonValue = parseJSONValue(entry)
    if (jsonValue === undefined) {
      return undefined
    }
    parsed[key] = jsonValue
  }
  return parsed
}

function parseJSONValue(value: unknown): JSONValue | undefined {
  if (
    value === null ||
    typeof value === 'string' ||
    typeof value === 'number' ||
    typeof value === 'boolean'
  ) {
    return value
  }

  if (Array.isArray(value)) {
    const parsed: JSONArray = []
    for (const item of value) {
      const jsonValue = parseJSONValue(item)
      if (jsonValue === undefined) {
        return undefined
      }
      parsed.push(jsonValue)
    }
    return parsed
  }

  return parseOptionalJSONObject(value)
}

function isJSONObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
