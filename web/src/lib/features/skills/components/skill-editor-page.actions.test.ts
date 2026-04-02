import { fireEvent, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import {
  bindSkill,
  buildSkillEditorData,
  deleteSkill,
  disableSkill,
  enableSkill,
  goto,
  initialContent,
  loadSkillEditorData,
  renderPage,
  resetSkillEditorPageTestState,
  runbookContent,
  setupSkillEditorPageGlobals,
  toastStore,
  unbindSkill,
  updateSkill,
} from './skill-editor-page.test-helpers'

describe('SkillEditorPage', () => {
  beforeAll(() => {
    setupSkillEditorPageGlobals()
  })

  afterEach(() => {
    resetSkillEditorPageTestState()
  })

  it('saves edited description and content then reloads the skill data', async () => {
    const savedContent = `${initialContent}\n\nVerify rollback before production deploys.`
    loadSkillEditorData
      .mockResolvedValueOnce(
        buildSkillEditorData({
          workflows: [{ id: 'workflow-1', name: 'Coding Workflow' }],
        }),
      )
      .mockResolvedValueOnce(
        buildSkillEditorData({
          skill: {
            description: 'Deploy safely with rollback checks',
            current_version: 4,
          },
          content: savedContent,
          files: [
            {
              path: 'references/runbook.md',
              file_kind: 'reference',
              media_type: 'text/markdown; charset=utf-8',
              encoding: 'utf8',
              is_executable: false,
              size_bytes: runbookContent.length,
              sha256: 'sha-runbook',
              content: runbookContent,
              content_base64: 'ignored',
            },
          ],
          workflows: [{ id: 'workflow-1', name: 'Coding Workflow' }],
        }),
      )
    updateSkill.mockResolvedValue({ skill: { id: 'skill-1' } })

    const { container, findByRole, findByPlaceholderText } = await renderPage({
      workflows: [{ id: 'workflow-1', name: 'Coding Workflow' }],
    })

    const descriptionInput = (await findByPlaceholderText('Description...')) as HTMLInputElement
    await fireEvent.input(descriptionInput, {
      target: { value: 'Deploy safely with rollback checks' },
    })

    const editor = container.querySelector('textarea')
    if (!(editor instanceof HTMLTextAreaElement)) {
      throw new Error('expected skill editor textarea to render')
    }
    await fireEvent.input(editor, { target: { value: savedContent } })
    await fireEvent.click(await findByRole('button', { name: 'Save' }))

    await waitFor(() => {
      expect(updateSkill).toHaveBeenCalledWith(
        'skill-1',
        expect.objectContaining({
          description: 'Deploy safely with rollback checks',
          content: savedContent,
        }),
      )
      expect(loadSkillEditorData).toHaveBeenLastCalledWith('skill-1', 'project-1')
      expect(toastStore.success).toHaveBeenCalledWith('Saved deploy.')
      expect(descriptionInput.value).toBe('Deploy safely with rollback checks')
      expect(editor.value).toBe(savedContent)
    })
  })

  it('toggles the enabled state and updates the header action', async () => {
    loadSkillEditorData.mockResolvedValue(
      buildSkillEditorData({
        skill: {
          is_enabled: true,
        },
      }),
    )
    disableSkill.mockResolvedValue({
      skill: {
        ...buildSkillEditorData().skill,
        is_enabled: false,
      },
    })
    enableSkill.mockResolvedValue({
      skill: {
        ...buildSkillEditorData().skill,
        is_enabled: true,
      },
    })

    const { findByTitle } = await renderPage({
      skill: {
        is_enabled: true,
      },
    })

    const disableButton = await findByTitle('Disable')
    await fireEvent.click(disableButton)

    await waitFor(() => {
      expect(disableSkill).toHaveBeenCalledWith('skill-1')
      expect(toastStore.success).toHaveBeenCalledWith('Disabled deploy.')
    })

    const enableButton = await findByTitle('Enable')
    await fireEvent.click(enableButton)

    await waitFor(() => {
      expect(enableSkill).toHaveBeenCalledWith('skill-1')
      expect(toastStore.success).toHaveBeenCalledWith('Enabled deploy.')
    })
  })

  it('binds and unbinds workflows from the metadata bar', async () => {
    loadSkillEditorData.mockResolvedValue(
      buildSkillEditorData({
        workflows: [{ id: 'workflow-1', name: 'Coding Workflow' }],
      }),
    )
    bindSkill.mockResolvedValue({
      skill: {
        ...buildSkillEditorData().skill,
        bound_workflows: [{ id: 'workflow-1' }],
      },
    })
    unbindSkill.mockResolvedValue({
      skill: {
        ...buildSkillEditorData().skill,
        bound_workflows: [],
      },
    })

    const { findByTitle } = await renderPage({
      workflows: [{ id: 'workflow-1', name: 'Coding Workflow' }],
    })

    await fireEvent.click(await findByTitle('Bind to Coding Workflow'))

    await waitFor(() => {
      expect(bindSkill).toHaveBeenCalledWith('skill-1', ['workflow-1'])
      expect(toastStore.success).toHaveBeenCalledWith('Bound deploy to Coding Workflow.')
    })

    await fireEvent.click(await findByTitle('Unbind from Coding Workflow'))

    await waitFor(() => {
      expect(unbindSkill).toHaveBeenCalledWith('skill-1', ['workflow-1'])
      expect(toastStore.success).toHaveBeenCalledWith('Unbound deploy from Coding Workflow.')
    })
  })

  it('deletes the skill after confirmation and navigates back to the skills page', async () => {
    loadSkillEditorData.mockResolvedValue(buildSkillEditorData())
    deleteSkill.mockResolvedValue({ skill: { id: 'skill-1' } })
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    const { findByTitle } = await renderPage()

    await fireEvent.click(await findByTitle('Delete skill'))

    await waitFor(() => {
      expect(deleteSkill).toHaveBeenCalledWith('skill-1')
      expect(toastStore.success).toHaveBeenCalledWith('Deleted deploy.')
      expect(goto).toHaveBeenCalledWith('/orgs/org-1/projects/project-1/skills')
    })
  })
})
