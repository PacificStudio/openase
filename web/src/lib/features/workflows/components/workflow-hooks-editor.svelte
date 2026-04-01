<script lang="ts">
  import type { WorkflowHooksDraft, WorkflowHookDraftValidation } from '../workflow-hooks'
  import {
    createWorkflowHookRowDraft,
    listTicketHookEventOptions,
    listWorkflowHookEventOptions,
    type TicketHookEvent,
    type WorkflowHookEvent,
    type WorkflowHookRowDraft,
  } from '../workflow-hooks'
  import WorkflowHookGroupEditor from './workflow-hook-group-editor.svelte'

  let {
    draft,
    validation,
    disabled = false,
    onChange,
  }: {
    draft: WorkflowHooksDraft
    validation: WorkflowHookDraftValidation
    disabled?: boolean
    onChange?: (draft: WorkflowHooksDraft) => void
  } = $props()

  const workflowEvents = listWorkflowHookEventOptions()
  const ticketEvents = listTicketHookEventOptions()

  function replaceWorkflowRows(event: WorkflowHookEvent, rows: WorkflowHookRowDraft[]) {
    onChange?.({
      ...draft,
      workflowHooks: {
        ...draft.workflowHooks,
        [event]: rows,
      },
    })
  }

  function replaceTicketRows(event: TicketHookEvent, rows: WorkflowHookRowDraft[]) {
    onChange?.({
      ...draft,
      ticketHooks: {
        ...draft.ticketHooks,
        [event]: rows,
      },
    })
  }

  function addWorkflowRow(event: WorkflowHookEvent) {
    replaceWorkflowRows(event, [...draft.workflowHooks[event], createWorkflowHookRowDraft()])
  }

  function addTicketRow(event: TicketHookEvent) {
    replaceTicketRows(event, [...draft.ticketHooks[event], createWorkflowHookRowDraft()])
  }

  function updateWorkflowRow(event: WorkflowHookEvent, index: number, row: WorkflowHookRowDraft) {
    replaceWorkflowRows(
      event,
      draft.workflowHooks[event].map((currentRow, currentIndex) =>
        currentIndex === index ? row : currentRow,
      ),
    )
  }

  function updateTicketRow(event: TicketHookEvent, index: number, row: WorkflowHookRowDraft) {
    replaceTicketRows(
      event,
      draft.ticketHooks[event].map((currentRow, currentIndex) =>
        currentIndex === index ? row : currentRow,
      ),
    )
  }

  function duplicateWorkflowRow(event: WorkflowHookEvent, index: number) {
    const row = draft.workflowHooks[event][index]
    if (!row) return

    const rows = [...draft.workflowHooks[event]]
    rows.splice(index + 1, 0, createWorkflowHookRowDraft(row))
    replaceWorkflowRows(event, rows)
  }

  function duplicateTicketRow(event: TicketHookEvent, index: number) {
    const row = draft.ticketHooks[event][index]
    if (!row) return

    const rows = [...draft.ticketHooks[event]]
    rows.splice(index + 1, 0, createWorkflowHookRowDraft(row))
    replaceTicketRows(event, rows)
  }

  function deleteWorkflowRow(event: WorkflowHookEvent, index: number) {
    replaceWorkflowRows(
      event,
      draft.workflowHooks[event].filter((_, currentIndex) => currentIndex !== index),
    )
  }

  function deleteTicketRow(event: TicketHookEvent, index: number) {
    replaceTicketRows(
      event,
      draft.ticketHooks[event].filter((_, currentIndex) => currentIndex !== index),
    )
  }
</script>

<div class="space-y-6">
  <WorkflowHookGroupEditor
    label="Workflow Hooks"
    description="Workflow lifecycle hooks run in the project runtime context."
    events={workflowEvents}
    rowsByEvent={draft.workflowHooks}
    rowErrors={validation.rowErrors}
    {disabled}
    onAdd={addWorkflowRow}
    onChange={updateWorkflowRow}
    onDuplicate={duplicateWorkflowRow}
    onDelete={deleteWorkflowRow}
  />

  <WorkflowHookGroupEditor
    label="Ticket Hooks"
    description="Ticket lifecycle hooks run in the ticket workspace and can set workdir per row."
    events={ticketEvents}
    rowsByEvent={draft.ticketHooks}
    rowErrors={validation.rowErrors}
    allowWorkdir={true}
    {disabled}
    onAdd={addTicketRow}
    onChange={updateTicketRow}
    onDuplicate={duplicateTicketRow}
    onDelete={deleteTicketRow}
  />
</div>
