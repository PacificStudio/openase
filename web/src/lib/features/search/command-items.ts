import type { Project } from '$lib/api/contracts'
import type { SearchItem } from './types'

export function buildProjectAssistantCommand(project: Project): SearchItem {
  return {
    id: 'command-open-project-ai',
    group: 'Commands',
    kind: 'command',
    title: 'Ask AI',
    subtitle: `Open project AI for ${project.name}.`,
    badge: 'Command',
    searchText:
      'Ask AI Open project AI. Command ai assistant project chat cron repo setup blocked work',
    action: { kind: 'open_project_ai' },
  }
}
