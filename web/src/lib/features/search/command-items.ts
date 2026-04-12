import type { Project } from '$lib/api/contracts'
import type { SearchItem } from './types'
import { searchT } from './i18n'

export function buildProjectAssistantCommand(project: Project): SearchItem {
  return {
    id: 'command-open-project-ai',
    group: 'Commands',
    kind: 'command',
    title: searchT('search.askAI'),
    subtitle: searchT('search.openProjectAISubtitle', { project: project.name }),
    badge: searchT('search.commandBadge'),
    searchText:
      'Ask AI Open project AI. Command ai assistant project chat cron repo setup blocked work',
    action: { kind: 'open_project_ai' },
  }
}
