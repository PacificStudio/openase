import type { TranslationKey } from '$lib/i18n'
import type { ProjectSection } from '$lib/stores/app-context'

export type PageSection = Exclude<ProjectSection, 'dashboard'>

export type PageTourStep = {
  tourId: string
  titleKey: TranslationKey
  descriptionKey: TranslationKey
  side?: 'top' | 'right' | 'bottom' | 'left'
}

const helpStep = (section: PageSection): PageTourStep => ({
  tourId: `page-help-${section}`,
  titleKey: 'tour.page.helpButton.title',
  descriptionKey: 'tour.page.helpButton.description',
  side: 'bottom',
})

const aiAssistantStep: PageTourStep = {
  tourId: 'sidebar-ai-assistant',
  titleKey: 'tour.sidebarAiAssistant.title',
  descriptionKey: 'tour.sidebarAiAssistant.description',
  side: 'right',
}

const sseStatusStep: PageTourStep = {
  tourId: 'topbar-sse-status',
  titleKey: 'tour.topbarSseStatus.title',
  descriptionKey: 'tour.topbarSseStatus.description',
  side: 'bottom',
}

export const PAGE_TOURS: Record<PageSection, PageTourStep[]> = {
  tickets: [
    {
      tourId: 'board-toolbar',
      titleKey: 'tour.page.tickets.toolbar.title',
      descriptionKey: 'tour.page.tickets.toolbar.description',
      side: 'bottom',
    },
    {
      tourId: 'board-view-toggle',
      titleKey: 'tour.page.tickets.viewToggle.title',
      descriptionKey: 'tour.page.tickets.viewToggle.description',
      side: 'bottom',
    },
    {
      tourId: 'board-columns-container',
      titleKey: 'tour.page.tickets.boardColumns.title',
      descriptionKey: 'tour.page.tickets.boardColumns.description',
      side: 'top',
    },
    {
      tourId: 'topbar-new-ticket',
      titleKey: 'tour.topbarNewTicket.title',
      descriptionKey: 'tour.topbarNewTicket.description',
      side: 'bottom',
    },
    aiAssistantStep,
    helpStep('tickets'),
  ],
  agents: [
    {
      tourId: 'agents-register',
      titleKey: 'tour.page.agents.register.title',
      descriptionKey: 'tour.page.agents.register.description',
      side: 'bottom',
    },
    {
      tourId: 'agents-list-panel',
      titleKey: 'tour.page.agents.listPanel.title',
      descriptionKey: 'tour.page.agents.listPanel.description',
      side: 'top',
    },
    {
      tourId: 'agent-cards-list',
      titleKey: 'tour.page.agents.cardActions.title',
      descriptionKey: 'tour.page.agents.cardActions.description',
      side: 'top',
    },
    aiAssistantStep,
    helpStep('agents'),
  ],
  machines: [
    {
      tourId: 'machines-actions',
      titleKey: 'tour.page.machines.actions.title',
      descriptionKey: 'tour.page.machines.actions.description',
      side: 'bottom',
    },
    {
      tourId: 'machines-search',
      titleKey: 'tour.page.machines.search.title',
      descriptionKey: 'tour.page.machines.search.description',
      side: 'bottom',
    },
    {
      tourId: 'machines-list-panel',
      titleKey: 'tour.page.machines.listPanel.title',
      descriptionKey: 'tour.page.machines.listPanel.description',
      side: 'top',
    },
    sseStatusStep,
    helpStep('machines'),
  ],
  updates: [
    {
      tourId: 'updates-composer',
      titleKey: 'tour.page.updates.composer.title',
      descriptionKey: 'tour.page.updates.composer.description',
      side: 'bottom',
    },
    {
      tourId: 'project-updates-threads',
      titleKey: 'tour.page.updates.threadsList.title',
      descriptionKey: 'tour.page.updates.threadsList.description',
      side: 'top',
    },
    aiAssistantStep,
    helpStep('updates'),
  ],
  activity: [
    {
      tourId: 'activity-filters',
      titleKey: 'tour.page.activity.filters.title',
      descriptionKey: 'tour.page.activity.filters.description',
      side: 'bottom',
    },
    {
      tourId: 'activity-timeline',
      titleKey: 'tour.page.activity.timeline.title',
      descriptionKey: 'tour.page.activity.timeline.description',
      side: 'top',
    },
    sseStatusStep,
    helpStep('activity'),
  ],
  workflows: [
    {
      tourId: 'workflows-actions',
      titleKey: 'tour.page.workflows.actions.title',
      descriptionKey: 'tour.page.workflows.actions.description',
      side: 'bottom',
    },
    {
      tourId: 'workflows-list-panel',
      titleKey: 'tour.page.workflows.listPanel.title',
      descriptionKey: 'tour.page.workflows.listPanel.description',
      side: 'right',
    },
    {
      tourId: 'workflow-detail-panel',
      titleKey: 'tour.page.workflows.detailPanel.title',
      descriptionKey: 'tour.page.workflows.detailPanel.description',
      side: 'left',
    },
    aiAssistantStep,
    helpStep('workflows'),
  ],
  skills: [
    {
      tourId: 'skills-stats-region',
      titleKey: 'tour.page.skills.statsRegion.title',
      descriptionKey: 'tour.page.skills.statsRegion.description',
      side: 'bottom',
    },
    {
      tourId: 'skills-toolbar',
      titleKey: 'tour.page.skills.toolbar.title',
      descriptionKey: 'tour.page.skills.toolbar.description',
      side: 'bottom',
    },
    {
      tourId: 'skills-create',
      titleKey: 'tour.page.skills.create.title',
      descriptionKey: 'tour.page.skills.create.description',
      side: 'bottom',
    },
    {
      tourId: 'skills-list-grid',
      titleKey: 'tour.page.skills.listGrid.title',
      descriptionKey: 'tour.page.skills.listGrid.description',
      side: 'top',
    },
    helpStep('skills'),
  ],
  'scheduled-jobs': [
    {
      tourId: 'scheduled-jobs-page',
      titleKey: 'tour.page.scheduledJobs.panel.title',
      descriptionKey: 'tour.page.scheduledJobs.panel.description',
      side: 'bottom',
    },
    {
      tourId: 'scheduled-jobs-summary',
      titleKey: 'tour.page.scheduledJobs.summary.title',
      descriptionKey: 'tour.page.scheduledJobs.summary.description',
      side: 'bottom',
    },
    {
      tourId: 'scheduled-jobs-list',
      titleKey: 'tour.page.scheduledJobs.list.title',
      descriptionKey: 'tour.page.scheduledJobs.list.description',
      side: 'top',
    },
    helpStep('scheduled-jobs'),
  ],
  settings: [
    {
      tourId: 'settings-nav',
      titleKey: 'tour.page.settings.nav.title',
      descriptionKey: 'tour.page.settings.nav.description',
      side: 'right',
    },
    {
      tourId: 'settings-content-panel',
      titleKey: 'tour.page.settings.contentPanel.title',
      descriptionKey: 'tour.page.settings.contentPanel.description',
      side: 'left',
    },
    aiAssistantStep,
    helpStep('settings'),
  ],
}
