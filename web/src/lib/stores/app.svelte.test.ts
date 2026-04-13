import { afterEach, describe, expect, it } from 'vitest'

import { appStore } from './app.svelte'

describe('appStore provisional project state', () => {
  const originalProjects = appStore.projects
  const originalCurrentProject = appStore.currentProject

  afterEach(() => {
    appStore.projects = originalProjects
    appStore.currentProject = originalCurrentProject
  })

  it('uses a valid dashboard status for unresolved projects', () => {
    appStore.projects = []
    appStore.currentProject = null

    const project = appStore.resolveProject('org-1', 'project-1')

    expect(project?.status).toBe('Planned')
  })
})
