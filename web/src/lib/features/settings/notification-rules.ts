import type { NotificationRule, NotificationRuleEventType } from '$lib/api/contracts'

import {
  formatJSONObject,
  parseJSONObject,
  type JSONObject,
  type ParseResult,
} from './notification-support'

export type RuleDraft = {
  id: string | null
  name: string
  eventType: string
  filterText: string
  channelId: string
  template: string
  isEnabled: boolean
}

export type RuleCreateInput = {
  name: string
  event_type: string
  filter: JSONObject
  channel_id: string
  template: string
  is_enabled: boolean
}

export type RuleUpdateInput = {
  name?: string
  event_type?: string
  filter?: JSONObject
  channel_id?: string
  template?: string
  is_enabled?: boolean
}

export function createRuleDraft(
  eventTypes: NotificationRuleEventType[],
  channelId = '',
): RuleDraft {
  const defaultEventType = eventTypes[0]
  return {
    id: null,
    name: '',
    eventType: defaultEventType?.event_type ?? '',
    filterText: formatJSONObject({}),
    channelId,
    template: defaultEventType?.default_template ?? '',
    isEnabled: true,
  }
}

export function ruleDraftFromRecord(rule: NotificationRule): RuleDraft {
  return {
    id: rule.id,
    name: rule.name,
    eventType: rule.event_type,
    filterText: formatJSONObject(rule.filter),
    channelId: rule.channel_id,
    template: rule.template,
    isEnabled: rule.is_enabled,
  }
}

export function applyRuleEventType(
  draft: RuleDraft,
  nextEventType: string,
  eventTypes: NotificationRuleEventType[],
): RuleDraft {
  const currentEvent = findEventType(eventTypes, draft.eventType)
  const nextEvent = findEventType(eventTypes, nextEventType)
  const shouldReplaceTemplate =
    draft.template.trim() === '' ||
    (currentEvent?.default_template !== undefined &&
      draft.template.trim() === currentEvent.default_template.trim())

  return {
    ...draft,
    eventType: nextEventType,
    template: shouldReplaceTemplate ? (nextEvent?.default_template ?? '') : draft.template,
  }
}

export function findEventType(
  eventTypes: NotificationRuleEventType[],
  eventType: string,
): NotificationRuleEventType | undefined {
  return eventTypes.find((item) => item.event_type === eventType)
}

export function buildCreateRuleInput(draft: RuleDraft): ParseResult<RuleCreateInput> {
  const name = draft.name.trim()
  if (name === '') return { ok: false, error: 'Rule name is required.' }
  if (draft.eventType.trim() === '') return { ok: false, error: 'Event type is required.' }
  if (draft.channelId.trim() === '') return { ok: false, error: 'Rule channel is required.' }

  const filter = parseJSONObject(draft.filterText, 'Rule filter')
  if (!filter.ok) {
    return filter
  }

  return {
    ok: true,
    value: {
      name,
      event_type: draft.eventType.trim(),
      filter: filter.value,
      channel_id: draft.channelId.trim(),
      template: draft.template.trim(),
      is_enabled: draft.isEnabled,
    },
  }
}

export function buildUpdateRuleInput(
  draft: RuleDraft,
  current: NotificationRule,
): ParseResult<{ changed: boolean; value: RuleUpdateInput }> {
  const name = draft.name.trim()
  if (name === '') return { ok: false, error: 'Rule name is required.' }
  if (draft.eventType.trim() === '') return { ok: false, error: 'Event type is required.' }
  if (draft.channelId.trim() === '') return { ok: false, error: 'Rule channel is required.' }

  const patch: RuleUpdateInput = {}
  const filterChanged = draft.filterText.trim() !== formatJSONObject(current.filter).trim()
  const template = draft.template.trim()

  if (name !== current.name) patch.name = name
  if (draft.eventType.trim() !== current.event_type) patch.event_type = draft.eventType.trim()
  if (filterChanged) {
    const filter = parseJSONObject(draft.filterText, 'Rule filter')
    if (!filter.ok) {
      return filter
    }
    patch.filter = filter.value
  }
  if (draft.channelId.trim() !== current.channel_id) patch.channel_id = draft.channelId.trim()
  if (template !== current.template.trim()) patch.template = template
  if (draft.isEnabled !== current.is_enabled) patch.is_enabled = draft.isEnabled

  return {
    ok: true,
    value: { changed: Object.keys(patch).length > 0, value: patch },
  }
}
