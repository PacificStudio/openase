import {
  toErrorMessage,
  type Organization,
  type Project,
} from '$lib/features/workspace/public'
import { createWorkspaceController, type WorkspaceStartOptions } from '$lib/features/workspace/controller.svelte'
import type { WorkspaceController } from '$lib/features/workspace/context'
import {
  createNotificationChannel,
  createNotificationRule,
  deleteNotificationChannel,
  deleteNotificationRule,
  loadNotificationChannels,
  loadNotificationEventTypes,
  loadNotificationRules,
  testNotificationChannel,
  updateNotificationChannel,
  updateNotificationRule,
} from './api'
import {
  applyRuleEventType as applyRuleEventSelection,
  beginChannelEdit as applyChannelEditor,
  beginRuleEdit as applyRuleEditor,
  createNotificationsState,
  resetChannelEditor,
  resetRuleEditor,
  setChannelReplaceConfig as applyChannelReplaceConfig,
  setChannelType as applyChannelType,
} from './editor'
import type { NotificationChannel } from './types'
import {
  buildChannelCreateInput,
  buildChannelUpdateInput,
  buildRuleCreateInput,
  buildRuleUpdateInput,
} from './payloads'

export function createNotificationsController(options: { workspace?: WorkspaceController } = {}) {
  const workspace = options.workspace ?? createWorkspaceController()
  const ownsWorkspace = !options.workspace
  const state = $state(createNotificationsState())
  async function start(options: WorkspaceStartOptions = {}) {
    if (ownsWorkspace) {
      await workspace.start(options)
    }

    await Promise.all([refreshEventTypes(), refreshChannels(), refreshRules()])
  }
  function destroy() {
    if (ownsWorkspace) {
      workspace.destroy()
    }
  }
  async function refreshEventTypes() {
    state.eventTypes = await loadNotificationEventTypes()
    if (!state.ruleForm.eventType && state.eventTypes[0]) {
      applyRuleEventType(state.eventTypes[0].event_type)
    }
  }

  async function refreshChannels() {
    const organizationId = workspace.state.selectedOrgId
    if (!organizationId) {
      state.channels = []
      resetChannelForm()
      return
    }

    state.channelBusy = true
    state.channelError = ''
    try {
      state.channels = await loadNotificationChannels(organizationId)
      const selectedChannel = currentChannel()
      if (selectedChannel) {
        applyChannelEditor(state, selectedChannel)
      } else if (state.channelMode === 'edit') {
        resetChannelForm()
      }
      if (!state.ruleForm.channelId && state.channels[0]) {
        state.ruleForm.channelId = state.channels[0].id
      }
    } catch (error) {
      state.channels = []
      state.channelError = toErrorMessage(error)
    } finally {
      state.channelBusy = false
    }
  }

  async function refreshRules() {
    const projectId = workspace.state.selectedProjectId
    if (!projectId) {
      state.rules = []
      resetRuleForm()
      return
    }

    state.ruleBusy = true
    state.ruleError = ''
    try {
      state.rules = await loadNotificationRules(projectId)
      const selectedRule = currentRule()
      if (selectedRule) {
        applyRuleEditor(state, selectedRule)
      } else if (state.ruleMode === 'edit') {
        resetRuleForm()
      }
    } catch (error) {
      state.rules = []
      state.ruleError = toErrorMessage(error)
    } finally {
      state.ruleBusy = false
    }
  }

  async function selectOrganization(organization: Organization) {
    await workspace.selectOrganization(organization)
    resetChannelForm()
    resetRuleForm()
    await Promise.all([refreshChannels(), refreshRules()])
  }

  async function selectProject(project: Project) {
    await workspace.selectProject(project)
    resetRuleForm()
    await refreshRules()
  }

  function resetChannelForm() {
    resetChannelEditor(state)
  }

  function beginChannelEdit(channelId: string) {
    applyChannelEditor(state, state.channels.find((item) => item.id === channelId) ?? null)
  }

  function setChannelType(nextType: NotificationChannel['type']) {
    applyChannelType(state, nextType, currentChannel() ?? null)
  }

  function setChannelReplaceConfig(enabled: boolean) {
    applyChannelReplaceConfig(state, enabled, currentChannel() ?? null)
  }

  async function saveChannel() {
    const organizationId = workspace.state.selectedOrgId
    if (!organizationId) {
      state.channelError = 'Select an organization first.'
      return
    }

    state.channelBusy = true
    state.channelError = ''
    state.channelNotice = ''

    try {
      if (state.channelMode === 'create') {
        await createNotificationChannel(organizationId, buildChannelCreateInput(state))
        state.channelNotice = 'Notification channel created.'
        resetChannelForm()
      } else {
        const channel = currentChannel()
        if (!channel) {
          throw new Error('Selected channel no longer exists.')
        }

        await updateNotificationChannel(channel.id, buildChannelUpdateInput(state, channel))
        state.channelNotice = 'Notification channel updated.'
      }

      await Promise.all([refreshChannels(), refreshRules()])
    } catch (error) {
      state.channelError = toErrorMessage(error)
    } finally {
      state.channelBusy = false
    }
  }

  async function removeChannel(channelId: string) {
    state.channelBusy = true
    state.channelError = ''
    state.channelNotice = ''
    try {
      await deleteNotificationChannel(channelId)
      if (state.selectedChannelId === channelId) {
        resetChannelForm()
      }
      state.channelNotice = 'Notification channel deleted.'
      await Promise.all([refreshChannels(), refreshRules()])
    } catch (error) {
      state.channelError = toErrorMessage(error)
    } finally {
      state.channelBusy = false
    }
  }

  async function sendChannelTest(channelId: string) {
    state.testingChannelId = channelId
    state.channelError = ''
    state.channelNotice = ''
    try {
      await testNotificationChannel(channelId)
      const name = state.channels.find((item) => item.id === channelId)?.name ?? 'Channel'
      state.channelNotice = `${name} test notification sent.`
    } catch (error) {
      state.channelError = toErrorMessage(error)
    } finally {
      state.testingChannelId = ''
    }
  }

  function resetRuleForm() {
    resetRuleEditor(state)
  }

  function beginRuleEdit(ruleId: string) {
    applyRuleEditor(state, state.rules.find((item) => item.id === ruleId) ?? null)
  }
  function applyRuleEventType(eventType: string) {
    applyRuleEventSelection(state, eventType)
  }
  async function saveRule() {
    const projectId = workspace.state.selectedProjectId
    if (!projectId) {
      state.ruleError = 'Select a project first.'
      return
    }

    state.ruleBusy = true
    state.ruleError = ''
    state.ruleNotice = ''

    try {
      if (state.ruleMode === 'create') {
        await createNotificationRule(projectId, buildRuleCreateInput(state))
        state.ruleNotice = 'Notification rule created.'
        resetRuleForm()
      } else {
        const rule = currentRule()
        if (!rule) {
          throw new Error('Selected rule no longer exists.')
        }

        await updateNotificationRule(rule.id, buildRuleUpdateInput(state, rule))
        state.ruleNotice = 'Notification rule updated.'
      }

      await refreshRules()
    } catch (error) {
      state.ruleError = toErrorMessage(error)
    } finally {
      state.ruleBusy = false
    }
  }

  async function removeRule(ruleId: string) {
    state.ruleBusy = true
    state.ruleError = ''
    state.ruleNotice = ''
    try {
      await deleteNotificationRule(ruleId)
      if (state.selectedRuleId === ruleId) {
        resetRuleForm()
      }
      state.ruleNotice = 'Notification rule deleted.'
      await refreshRules()
    } catch (error) {
      state.ruleError = toErrorMessage(error)
    } finally {
      state.ruleBusy = false
    }
  }

  return {
    workspace,
    state,
    start,
    destroy,
    refreshChannels,
    refreshRules,
    selectOrganization,
    selectProject,
    resetChannelForm,
    beginChannelEdit,
    setChannelType,
    setChannelReplaceConfig,
    saveChannel,
    removeChannel,
    sendChannelTest,
    resetRuleForm,
    beginRuleEdit,
    applyRuleEventType,
    saveRule,
    removeRule,
  }

  function currentChannel() {
    return state.channels.find((item) => item.id === state.selectedChannelId)
  }

  function currentRule() {
    return state.rules.find((item) => item.id === state.selectedRuleId)
  }
}
