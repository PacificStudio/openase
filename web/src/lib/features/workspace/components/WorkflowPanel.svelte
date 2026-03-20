<script lang="ts">
  import { Plus, Save, Sparkles, Trash2 } from '@lucide/svelte'
  import ScrollPane from '$lib/components/layout/ScrollPane.svelte'
  import { Badge } from '$lib/components/ui/badge'
  import { Button } from '$lib/components/ui/button'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
  import { inputClass, workflowTypes } from '$lib/features/workspace/constants'
  import type {
    BuiltinRole,
    HRAdvisorPayload,
    TicketStatus,
    Workflow,
    WorkflowForm,
  } from '$lib/features/workspace/types'

  let {
    selectedProject = null,
    selectedWorkflowId = '',
    selectedWorkflow = null,
    selectedBuiltinRoleSlug = '',
    builtinRoles = [],
    workflows = [],
    ticketStatuses = [],
    createForm,
    editForm,
    hrAdvisor = null,
    busy = false,
    onSelectRole,
    onClearRole,
    onSelectWorkflow,
    onCreate,
    onUpdate,
    onDelete,
    onLoadRecommendedRole,
  }: {
    selectedProject?: { name: string } | null
    selectedWorkflowId?: string
    selectedWorkflow?: Workflow | null
    selectedBuiltinRoleSlug?: string
    builtinRoles?: BuiltinRole[]
    workflows?: Workflow[]
    ticketStatuses?: TicketStatus[]
    createForm: WorkflowForm
    editForm: WorkflowForm
    hrAdvisor?: HRAdvisorPayload | null
    busy?: boolean
    onSelectRole?: (role: BuiltinRole) => void
    onClearRole?: () => void
    onSelectWorkflow?: (workflow: Workflow) => void
    onCreate?: () => void
    onUpdate?: () => void
    onDelete?: () => void
    onLoadRecommendedRole?: (recommendation: {
      role_slug: string
      suggested_workflow_name: string
    }) => void
  } = $props()
</script>

<Card class="border-border/80 bg-background/70">
  <CardHeader>
    <CardTitle class="flex items-center gap-2">
      <Sparkles class="size-4" />
      <span>Workflow</span>
    </CardTitle>
    <CardDescription
      >Create lanes from role templates, then edit the selected workflow.</CardDescription
    >
  </CardHeader>

  <CardContent class="space-y-5">
    <div class="space-y-3">
      <div class="flex flex-wrap gap-2">
        {#each builtinRoles.slice(0, 6) as role}
          <Button
            type="button"
            size="sm"
            variant={role.slug === selectedBuiltinRoleSlug ? 'default' : 'outline'}
            onclick={() => onSelectRole?.(role)}
          >
            {role.name}
          </Button>
        {/each}
        {#if selectedBuiltinRoleSlug}
          <Button type="button" size="sm" variant="ghost" onclick={onClearRole}>Clear</Button>
        {/if}
      </div>

      {#if hrAdvisor && hrAdvisor.recommendations.length > 0}
        <div class="border-border/70 bg-background/60 rounded-3xl border px-4 py-4">
          <p class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
            Recommended next role
          </p>
          <div class="mt-3 flex flex-wrap items-center justify-between gap-3">
            <div>
              <p class="text-sm font-semibold">{hrAdvisor.recommendations[0]?.role_name}</p>
              <p class="text-muted-foreground mt-1 text-sm">
                {hrAdvisor.recommendations[0]?.reason}
              </p>
            </div>
            <Button
              type="button"
              size="sm"
              variant="outline"
              onclick={() => onLoadRecommendedRole?.(hrAdvisor.recommendations[0])}
            >
              Load
            </Button>
          </div>
        </div>
      {/if}
    </div>

    <form
      class="space-y-3"
      onsubmit={(event) => {
        event.preventDefault()
        void onCreate?.()
      }}
    >
      <input
        class={inputClass}
        bind:value={createForm.name}
        placeholder="Coding"
        disabled={!selectedProject}
      />
      <select class={inputClass} bind:value={createForm.type} disabled={!selectedProject}>
        {#each workflowTypes as workflowType}
          <option value={workflowType}>{workflowType}</option>
        {/each}
      </select>
      <div class="grid gap-3 sm:grid-cols-2">
        <select
          class={inputClass}
          bind:value={createForm.pickupStatusId}
          disabled={!selectedProject}
        >
          {#each ticketStatuses as status}
            <option value={status.id}>{status.name}</option>
          {/each}
        </select>
        <select
          class={inputClass}
          bind:value={createForm.finishStatusId}
          disabled={!selectedProject}
        >
          <option value="">No auto-finish status</option>
          {#each ticketStatuses as status}
            <option value={status.id}>{status.name}</option>
          {/each}
        </select>
      </div>
      <Button
        class="w-full"
        type="submit"
        disabled={!selectedProject || busy || ticketStatuses.length === 0}
      >
        <Plus class="mr-2 size-4" />
        Create workflow
      </Button>
    </form>

    <div class="border-border/70 space-y-3 border-t pt-5">
      <div class="flex items-center justify-between gap-3">
        <p class="text-sm font-semibold">Available workflows</p>
        <Badge variant="outline">{workflows.length}</Badge>
      </div>
      {#if workflows.length === 0}
        <div
          class="text-muted-foreground border-border/70 bg-muted/30 rounded-3xl border border-dashed px-4 py-5 text-sm"
        >
          No workflows in the active project yet.
        </div>
      {:else}
        <ScrollPane class="max-h-64">
          <div class="grid gap-2">
          {#each workflows as workflow}
            <button
              type="button"
              class={`w-full rounded-2xl border px-4 py-3 text-left transition ${
                workflow.id === selectedWorkflowId
                  ? 'border-foreground/30 bg-foreground text-background'
                  : 'border-border/70 bg-background/60 hover:border-foreground/15 hover:bg-background'
              }`}
              onclick={() => void onSelectWorkflow?.(workflow)}
            >
              <div class="flex items-center justify-between gap-3">
                <span class="text-sm font-semibold">{workflow.name}</span>
                <Badge variant={workflow.id === selectedWorkflowId ? 'secondary' : 'outline'}>
                  {workflow.type}
                </Badge>
              </div>
            </button>
          {/each}
          </div>
        </ScrollPane>
      {/if}
    </div>

    <div class="border-border/70 border-t pt-5">
      {#if selectedWorkflow}
        <form
          class="space-y-3"
          onsubmit={(event) => {
            event.preventDefault()
            void onUpdate?.()
          }}
        >
          <input class={inputClass} bind:value={editForm.name} />
          <select class={inputClass} bind:value={editForm.type}>
            {#each workflowTypes as workflowType}
              <option value={workflowType}>{workflowType}</option>
            {/each}
          </select>
          <div class="grid gap-3 sm:grid-cols-2">
            <select class={inputClass} bind:value={editForm.pickupStatusId}>
              {#each ticketStatuses as status}
                <option value={status.id}>{status.name}</option>
              {/each}
            </select>
            <select class={inputClass} bind:value={editForm.finishStatusId}>
              <option value="">No auto-finish status</option>
              {#each ticketStatuses as status}
                <option value={status.id}>{status.name}</option>
              {/each}
            </select>
          </div>
          <div class="grid gap-3 sm:grid-cols-2">
            <input class={inputClass} bind:value={editForm.maxConcurrent} min="1" type="number" />
            <input
              class={inputClass}
              bind:value={editForm.maxRetryAttempts}
              min="0"
              type="number"
            />
          </div>
          <div class="grid gap-3 sm:grid-cols-2">
            <input class={inputClass} bind:value={editForm.timeoutMinutes} min="1" type="number" />
            <input
              class={inputClass}
              bind:value={editForm.stallTimeoutMinutes}
              min="1"
              type="number"
            />
          </div>
          <label
            class="border-border/70 bg-background/60 flex items-center gap-3 rounded-2xl border px-4 py-3 text-sm"
          >
            <input
              bind:checked={editForm.isActive}
              class="border-border size-4 rounded"
              type="checkbox"
            />
            <span>Workflow is active</span>
          </label>
          <div class="flex gap-3">
            <Button class="flex-1" type="submit" disabled={busy}>
              <Save class="mr-2 size-4" />
              Save
            </Button>
            <Button
              class="flex-1"
              type="button"
              variant="outline"
              disabled={busy}
              onclick={() => void onDelete?.()}
            >
              <Trash2 class="mr-2 size-4" />
              Delete
            </Button>
          </div>
        </form>
      {:else}
        <div
          class="text-muted-foreground border-border/70 bg-muted/30 rounded-3xl border border-dashed px-4 py-5 text-sm"
        >
          Select a workflow to edit it.
        </div>
      {/if}
    </div>
  </CardContent>
</Card>
