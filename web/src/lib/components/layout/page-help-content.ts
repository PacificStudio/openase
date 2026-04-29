import type { TranslationKey } from '$lib/i18n'
import type { ProjectSection } from '$lib/stores/app-context'

export type HelpItemRef = { id: string; titleKey: TranslationKey; descriptionKey: TranslationKey }
export type HelpSectionRef = { overviewKey: TranslationKey; items: HelpItemRef[] }

type HelpSection = Exclude<ProjectSection, 'dashboard'>

export const PAGE_HELP_SECTIONS: Record<HelpSection, HelpSectionRef> = {
  tickets: {
    overviewKey: 'help.tickets.overview',
    items: [
      {
        id: 'newTicket',
        titleKey: 'help.tickets.newTicket.title',
        descriptionKey: 'help.tickets.newTicket.description',
      },
      {
        id: 'viewMode',
        titleKey: 'help.tickets.viewMode.title',
        descriptionKey: 'help.tickets.viewMode.description',
      },
      {
        id: 'dragStatus',
        titleKey: 'help.tickets.dragStatus.title',
        descriptionKey: 'help.tickets.dragStatus.description',
      },
      {
        id: 'filterTickets',
        titleKey: 'help.tickets.filterTickets.title',
        descriptionKey: 'help.tickets.filterTickets.description',
      },
      {
        id: 'repositoryScope',
        titleKey: 'help.tickets.repositoryScope.title',
        descriptionKey: 'help.tickets.repositoryScope.description',
      },
      {
        id: 'archiveTicket',
        titleKey: 'help.tickets.archiveTicket.title',
        descriptionKey: 'help.tickets.archiveTicket.description',
      },
    ],
  },
  agents: {
    overviewKey: 'help.agents.overview',
    items: [
      {
        id: 'registerAgent',
        titleKey: 'help.agents.registerAgent.title',
        descriptionKey: 'help.agents.registerAgent.description',
      },
      {
        id: 'viewAgentList',
        titleKey: 'help.agents.viewAgentList.title',
        descriptionKey: 'help.agents.viewAgentList.description',
      },
      {
        id: 'pauseAgent',
        titleKey: 'help.agents.pauseAgent.title',
        descriptionKey: 'help.agents.pauseAgent.description',
      },
      {
        id: 'retireAgent',
        titleKey: 'help.agents.retireAgent.title',
        descriptionKey: 'help.agents.retireAgent.description',
      },
      {
        id: 'viewRun',
        titleKey: 'help.agents.viewRun.title',
        descriptionKey: 'help.agents.viewRun.description',
      },
      {
        id: 'runHistory',
        titleKey: 'help.agents.runHistory.title',
        descriptionKey: 'help.agents.runHistory.description',
      },
    ],
  },
  machines: {
    overviewKey: 'help.machines.overview',
    items: [
      {
        id: 'addMachine',
        titleKey: 'help.machines.addMachine.title',
        descriptionKey: 'help.machines.addMachine.description',
      },
      {
        id: 'chooseTopology',
        titleKey: 'help.machines.chooseTopology.title',
        descriptionKey: 'help.machines.chooseTopology.description',
      },
      {
        id: 'connectionTest',
        titleKey: 'help.machines.connectionTest.title',
        descriptionKey: 'help.machines.connectionTest.description',
      },
      {
        id: 'refreshHealth',
        titleKey: 'help.machines.refreshHealth.title',
        descriptionKey: 'help.machines.refreshHealth.description',
      },
      {
        id: 'sshHelper',
        titleKey: 'help.machines.sshHelper.title',
        descriptionKey: 'help.machines.sshHelper.description',
      },
      {
        id: 'issueChannelToken',
        titleKey: 'help.machines.issueChannelToken.title',
        descriptionKey: 'help.machines.issueChannelToken.description',
      },
    ],
  },
  updates: {
    overviewKey: 'help.updates.overview',
    items: [
      {
        id: 'newUpdate',
        titleKey: 'help.updates.newUpdate.title',
        descriptionKey: 'help.updates.newUpdate.description',
      },
      {
        id: 'commentUpdate',
        titleKey: 'help.updates.commentUpdate.title',
        descriptionKey: 'help.updates.commentUpdate.description',
      },
      {
        id: 'editContent',
        titleKey: 'help.updates.editContent.title',
        descriptionKey: 'help.updates.editContent.description',
      },
      {
        id: 'viewRevisions',
        titleKey: 'help.updates.viewRevisions.title',
        descriptionKey: 'help.updates.viewRevisions.description',
      },
    ],
  },
  activity: {
    overviewKey: 'help.activity.overview',
    items: [
      {
        id: 'viewStream',
        titleKey: 'help.activity.viewStream.title',
        descriptionKey: 'help.activity.viewStream.description',
      },
      {
        id: 'filterByType',
        titleKey: 'help.activity.filterByType.title',
        descriptionKey: 'help.activity.filterByType.description',
      },
      {
        id: 'searchEvents',
        titleKey: 'help.activity.searchEvents.title',
        descriptionKey: 'help.activity.searchEvents.description',
      },
      {
        id: 'openRelated',
        titleKey: 'help.activity.openRelated.title',
        descriptionKey: 'help.activity.openRelated.description',
      },
    ],
  },
  workflows: {
    overviewKey: 'help.workflows.overview',
    items: [
      {
        id: 'newWorkflow',
        titleKey: 'help.workflows.newWorkflow.title',
        descriptionKey: 'help.workflows.newWorkflow.description',
      },
      {
        id: 'editHarness',
        titleKey: 'help.workflows.editHarness.title',
        descriptionKey: 'help.workflows.editHarness.description',
      },
      {
        id: 'statusBinding',
        titleKey: 'help.workflows.statusBinding.title',
        descriptionKey: 'help.workflows.statusBinding.description',
      },
      {
        id: 'bindSkills',
        titleKey: 'help.workflows.bindSkills.title',
        descriptionKey: 'help.workflows.bindSkills.description',
      },
      {
        id: 'advancedLimits',
        titleKey: 'help.workflows.advancedLimits.title',
        descriptionKey: 'help.workflows.advancedLimits.description',
      },
      {
        id: 'impactAnalysis',
        titleKey: 'help.workflows.impactAnalysis.title',
        descriptionKey: 'help.workflows.impactAnalysis.description',
      },
      {
        id: 'versionHistory',
        titleKey: 'help.workflows.versionHistory.title',
        descriptionKey: 'help.workflows.versionHistory.description',
      },
    ],
  },
  skills: {
    overviewKey: 'help.skills.overview',
    items: [
      {
        id: 'browseSkills',
        titleKey: 'help.skills.browseSkills.title',
        descriptionKey: 'help.skills.browseSkills.description',
      },
      {
        id: 'newSkill',
        titleKey: 'help.skills.newSkill.title',
        descriptionKey: 'help.skills.newSkill.description',
      },
      {
        id: 'toggleEnabled',
        titleKey: 'help.skills.toggleEnabled.title',
        descriptionKey: 'help.skills.toggleEnabled.description',
      },
      {
        id: 'editSkill',
        titleKey: 'help.skills.editSkill.title',
        descriptionKey: 'help.skills.editSkill.description',
      },
      {
        id: 'viewBindings',
        titleKey: 'help.skills.viewBindings.title',
        descriptionKey: 'help.skills.viewBindings.description',
      },
      {
        id: 'deleteSkill',
        titleKey: 'help.skills.deleteSkill.title',
        descriptionKey: 'help.skills.deleteSkill.description',
      },
    ],
  },
  'scheduled-jobs': {
    overviewKey: 'help.scheduledJobs.overview',
    items: [
      {
        id: 'newJob',
        titleKey: 'help.scheduledJobs.newJob.title',
        descriptionKey: 'help.scheduledJobs.newJob.description',
      },
      {
        id: 'cronExpression',
        titleKey: 'help.scheduledJobs.cronExpression.title',
        descriptionKey: 'help.scheduledJobs.cronExpression.description',
      },
      {
        id: 'ticketTemplate',
        titleKey: 'help.scheduledJobs.ticketTemplate.title',
        descriptionKey: 'help.scheduledJobs.ticketTemplate.description',
      },
      {
        id: 'toggleJob',
        titleKey: 'help.scheduledJobs.toggleJob.title',
        descriptionKey: 'help.scheduledJobs.toggleJob.description',
      },
      {
        id: 'manualTrigger',
        titleKey: 'help.scheduledJobs.manualTrigger.title',
        descriptionKey: 'help.scheduledJobs.manualTrigger.description',
      },
      {
        id: 'editJob',
        titleKey: 'help.scheduledJobs.editJob.title',
        descriptionKey: 'help.scheduledJobs.editJob.description',
      },
    ],
  },
  settings: {
    overviewKey: 'help.settings.overview',
    items: [
      {
        id: 'general',
        titleKey: 'help.settings.general.title',
        descriptionKey: 'help.settings.general.description',
      },
      {
        id: 'ticketStatuses',
        titleKey: 'help.settings.ticketStatuses.title',
        descriptionKey: 'help.settings.ticketStatuses.description',
      },
      {
        id: 'repositories',
        titleKey: 'help.settings.repositories.title',
        descriptionKey: 'help.settings.repositories.description',
      },
      {
        id: 'agentProviders',
        titleKey: 'help.settings.agentProviders.title',
        descriptionKey: 'help.settings.agentProviders.description',
      },
      {
        id: 'notifications',
        titleKey: 'help.settings.notifications.title',
        descriptionKey: 'help.settings.notifications.description',
      },
      {
        id: 'security',
        titleKey: 'help.settings.security.title',
        descriptionKey: 'help.settings.security.description',
      },
      {
        id: 'archivedTickets',
        titleKey: 'help.settings.archivedTickets.title',
        descriptionKey: 'help.settings.archivedTickets.description',
      },
    ],
  },
}

export function getPageHelpSection(section: ProjectSection): HelpSectionRef | null {
  if (section === 'dashboard') return null
  return PAGE_HELP_SECTIONS[section] ?? null
}
