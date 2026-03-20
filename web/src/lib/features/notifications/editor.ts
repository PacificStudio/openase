import {
  defaultChannelForm,
  defaultRuleForm,
  toChannelForm,
  toRuleForm,
  type NotificationChannel,
  type NotificationRule,
  type NotificationRuleEventType,
} from './types'

export type NotificationsState = {
  channels: NotificationChannel[]
  rules: NotificationRule[]
  eventTypes: NotificationRuleEventType[]
  channelBusy: boolean
  ruleBusy: boolean
  testingChannelId: string
  channelError: string
  ruleError: string
  channelNotice: string
  ruleNotice: string
  channelMode: 'create' | 'edit'
  ruleMode: 'create' | 'edit'
  selectedChannelId: string
  selectedRuleId: string
  channelReplaceConfig: boolean
  channelForm: ReturnType<typeof defaultChannelForm>
  ruleForm: ReturnType<typeof defaultRuleForm>
}

export function createNotificationsState(): NotificationsState {
  return {
    channels: [],
    rules: [],
    eventTypes: [],
    channelBusy: false,
    ruleBusy: false,
    testingChannelId: '',
    channelError: '',
    ruleError: '',
    channelNotice: '',
    ruleNotice: '',
    channelMode: 'create',
    ruleMode: 'create',
    selectedChannelId: '',
    selectedRuleId: '',
    channelReplaceConfig: false,
    channelForm: defaultChannelForm(),
    ruleForm: defaultRuleForm(),
  }
}

export function resetChannelEditor(state: NotificationsState) {
  state.channelMode = 'create'
  state.selectedChannelId = ''
  state.channelReplaceConfig = false
  state.channelForm = defaultChannelForm()
  state.channelError = ''
}

export function beginChannelEdit(state: NotificationsState, channel: NotificationChannel | null) {
  if (!channel) {
    resetChannelEditor(state)
    return
  }

  state.channelMode = 'edit'
  state.selectedChannelId = channel.id
  state.channelReplaceConfig = false
  state.channelForm = toChannelForm(channel)
  state.channelNotice = ''
  state.channelError = ''
}

export function setChannelType(
  state: NotificationsState,
  nextType: NotificationChannel['type'],
  currentChannel: NotificationChannel | null,
) {
  state.channelForm.type = nextType
  if (state.channelMode === 'edit' && nextType !== currentChannel?.type) {
    state.channelReplaceConfig = true
  }
}

export function setChannelReplaceConfig(
  state: NotificationsState,
  enabled: boolean,
  currentChannel: NotificationChannel | null,
) {
  state.channelReplaceConfig = enabled
  if (!enabled && currentChannel) {
    state.channelForm.type = currentChannel.type
  }
}

export function resetRuleEditor(state: NotificationsState) {
  const firstEvent = state.eventTypes[0]
  state.ruleMode = 'create'
  state.selectedRuleId = ''
  state.ruleForm = defaultRuleForm(firstEvent?.event_type ?? '', firstEvent?.default_template ?? '')
  if (state.channels[0]) {
    state.ruleForm.channelId = state.channels[0].id
  }
  state.ruleError = ''
}

export function beginRuleEdit(state: NotificationsState, rule: NotificationRule | null) {
  if (!rule) {
    resetRuleEditor(state)
    return
  }

  state.ruleMode = 'edit'
  state.selectedRuleId = rule.id
  state.ruleForm = toRuleForm(rule)
  state.ruleNotice = ''
  state.ruleError = ''
}

export function applyRuleEventType(state: NotificationsState, eventType: string) {
  const currentEntry = state.eventTypes.find((item) => item.event_type === state.ruleForm.eventType)
  const nextEntry = state.eventTypes.find((item) => item.event_type === eventType)
  state.ruleForm.eventType = eventType
  if (
    nextEntry &&
    state.ruleMode === 'create' &&
    (!state.ruleForm.template || state.ruleForm.template === currentEntry?.default_template)
  ) {
    state.ruleForm.template = nextEntry.default_template
  }
}
