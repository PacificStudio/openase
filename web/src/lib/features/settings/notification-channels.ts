import type { NotificationChannel } from '$lib/api/contracts'

import {
  formatJSONObject,
  parseJSONObject,
  type JSONObject,
  type ParseResult,
} from './notification-support'

export type ChannelDraft = {
  id: string | null
  name: string
  type: string
  configText: string
  isEnabled: boolean
}

export type ChannelCreateInput = {
  name: string
  type: string
  config: JSONObject
  is_enabled: boolean
}

export type ChannelUpdateInput = {
  name?: string
  type?: string
  config?: JSONObject
  is_enabled?: boolean
}

const channelConfigTemplates: Record<string, JSONObject> = {
  webhook: {
    url: 'https://example.com/hooks/openase',
    headers: { 'X-Team': 'platform' },
    secret: 'replace-me',
  },
  telegram: {
    bot_token: '<telegram-bot-token>',
    chat_id: '<chat-id>',
  },
  slack: {
    webhook_url: 'https://hooks.slack.com/services/T000/B000/replace-me',
  },
  wecom: {
    webhook_key: '<wecom-webhook-key>',
  },
}

export function createChannelDraft(channelType = 'webhook'): ChannelDraft {
  return {
    id: null,
    name: '',
    type: channelType,
    configText: formatJSONObject(channelConfigTemplates[channelType] ?? {}),
    isEnabled: true,
  }
}

export function channelDraftFromRecord(channel: NotificationChannel): ChannelDraft {
  return {
    id: channel.id,
    name: channel.name,
    type: channel.type,
    configText: formatJSONObject(channel.config),
    isEnabled: channel.is_enabled,
  }
}

export function applyChannelTypeTemplate(draft: ChannelDraft, nextType: string): ChannelDraft {
  const normalizedType = normalizeChannelType(nextType)
  const currentTemplate = formatJSONObject(channelConfigTemplates[draft.type] ?? {})
  const nextTemplate = formatJSONObject(channelConfigTemplates[normalizedType] ?? {})
  const shouldReplaceTemplate =
    draft.configText.trim() === '' || draft.configText.trim() === currentTemplate.trim()

  return {
    ...draft,
    type: normalizedType,
    configText: shouldReplaceTemplate ? nextTemplate : draft.configText,
  }
}

export function buildCreateChannelInput(draft: ChannelDraft): ParseResult<ChannelCreateInput> {
  const name = draft.name.trim()
  if (name === '') {
    return { ok: false, error: 'Channel name is required.' }
  }

  const type = normalizeChannelType(draft.type)
  if (type === '') {
    return { ok: false, error: 'Channel type is required.' }
  }

  const config = parseJSONObject(draft.configText, 'Channel config')
  if (!config.ok) {
    return config
  }

  return {
    ok: true,
    value: { name, type, config: config.value, is_enabled: draft.isEnabled },
  }
}

export function buildUpdateChannelInput(
  draft: ChannelDraft,
  current: NotificationChannel,
): ParseResult<{ changed: boolean; value: ChannelUpdateInput }> {
  const name = draft.name.trim()
  if (name === '') {
    return { ok: false, error: 'Channel name is required.' }
  }

  const type = normalizeChannelType(draft.type)
  if (type === '') {
    return { ok: false, error: 'Channel type is required.' }
  }

  const patch: ChannelUpdateInput = {}
  const configChanged = draft.configText.trim() !== formatJSONObject(current.config).trim()

  if (name !== current.name) patch.name = name
  if (type !== current.type) patch.type = type
  if (configChanged || patch.type !== undefined) {
    const config = parseJSONObject(draft.configText, 'Channel config')
    if (!config.ok) {
      return config
    }
    patch.config = config.value
  }
  if (draft.isEnabled !== current.is_enabled) patch.is_enabled = draft.isEnabled

  return {
    ok: true,
    value: { changed: Object.keys(patch).length > 0, value: patch },
  }
}

function normalizeChannelType(value: string) {
  return value.trim().toLowerCase()
}
