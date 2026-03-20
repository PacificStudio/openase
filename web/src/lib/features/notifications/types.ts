export type NotificationChannelType = 'webhook' | 'telegram' | 'slack' | 'wecom'

export type NotificationChannel = {
  id: string
  organization_id: string
  name: string
  type: NotificationChannelType
  config: Record<string, unknown>
  is_enabled: boolean
  created_at: string
}

export type NotificationRuleEventType = {
  event_type: string
  label: string
  default_template: string
}

export type NotificationRule = {
  id: string
  project_id: string
  channel_id: string
  name: string
  event_type: string
  filter: Record<string, unknown>
  template: string
  is_enabled: boolean
  created_at: string
  channel: NotificationChannel
}

export type NotificationChannelForm = {
  name: string
  type: NotificationChannelType
  isEnabled: boolean
  webhookURL: string
  webhookHeaders: string
  webhookSecret: string
  telegramBotToken: string
  telegramChatID: string
  slackWebhookURL: string
  wecomWebhookKey: string
}

export type NotificationRuleForm = {
  name: string
  eventType: string
  channelId: string
  filterText: string
  template: string
  isEnabled: boolean
}

export function defaultChannelForm(
  type: NotificationChannelType = 'webhook',
): NotificationChannelForm {
  return {
    name: '',
    type,
    isEnabled: true,
    webhookURL: '',
    webhookHeaders: '{}',
    webhookSecret: '',
    telegramBotToken: '',
    telegramChatID: '',
    slackWebhookURL: '',
    wecomWebhookKey: '',
  }
}

export function defaultRuleForm(eventType = '', template = ''): NotificationRuleForm {
  return {
    name: '',
    eventType,
    channelId: '',
    filterText: '{}',
    template,
    isEnabled: true,
  }
}

export function toChannelForm(channel: NotificationChannel): NotificationChannelForm {
  return {
    ...defaultChannelForm(channel.type),
    name: channel.name,
    type: channel.type,
    isEnabled: channel.is_enabled,
  }
}

export function toRuleForm(rule: NotificationRule): NotificationRuleForm {
  return {
    name: rule.name,
    eventType: rule.event_type,
    channelId: rule.channel_id,
    filterText: formatJSONObject(rule.filter),
    template: rule.template,
    isEnabled: rule.is_enabled,
  }
}

export function formatJSONObject(value: Record<string, unknown> | undefined) {
  return JSON.stringify(value ?? {}, null, 2)
}

export function channelTypeLabel(type: NotificationChannelType) {
  switch (type) {
    case 'webhook':
      return 'Webhook'
    case 'telegram':
      return 'Telegram'
    case 'slack':
      return 'Slack'
    case 'wecom':
      return 'WeCom'
  }
}

export function configSummaryLines(channel: NotificationChannel) {
  switch (channel.type) {
    case 'webhook':
      return joinSummaryLines({
        URL: channel.config.url,
        Headers: summarizeObject(channel.config.headers),
        Secret: channel.config.secret,
      })
    case 'telegram':
      return joinSummaryLines({
        'Bot token': channel.config.bot_token,
        'Chat ID': channel.config.chat_id,
      })
    case 'slack':
      return joinSummaryLines({
        'Webhook URL': channel.config.webhook_url,
      })
    case 'wecom':
      return joinSummaryLines({
        'Webhook key': channel.config.webhook_key,
      })
  }
}

function summarizeObject(value: unknown) {
  if (!value || typeof value !== 'object') {
    return ''
  }

  const keys = Object.keys(value as Record<string, unknown>)
  return keys.length > 0 ? keys.join(', ') : ''
}

function joinSummaryLines(entries: Record<string, unknown>) {
  return Object.entries(entries)
    .filter(([, value]) => {
      if (typeof value === 'string') {
        return value.trim().length > 0
      }
      return Boolean(value)
    })
    .map(([label, value]) => `${label}: ${String(value)}`)
}
