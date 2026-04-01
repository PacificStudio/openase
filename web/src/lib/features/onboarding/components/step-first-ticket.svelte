<script lang="ts">
  import { untrack } from 'svelte'
  import { goto } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import { createTicket, listProjectRepos } from '$lib/api/openase'
  import type { ProjectRepoRecord, Workflow } from '$lib/api/contracts'
  import { appStore } from '$lib/stores/app.svelte'
  import { projectPath } from '$lib/stores/app-context'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Textarea } from '$ui/textarea'
  import * as Select from '$ui/select'
  import { Loader2, Ticket, Info, CheckCircle2 } from '@lucide/svelte'
  import { getBootstrapPreset } from '../model'

  let {
    projectId,
    orgId,
    projectStatus,
    workflows,
    ticketCount,
    onComplete,
  }: {
    projectId: string
    orgId: string
    projectStatus: string
    workflows: Workflow[]
    ticketCount: number
    onComplete: () => void
  } = $props()

  const preset = $derived(getBootstrapPreset(projectStatus))

  let title = $state(untrack(() => preset.exampleTicketTitle))
  let description = $state('')
  let selectedWorkflowId = $state(untrack(() => workflows[0]?.id ?? ''))
  let creating = $state(false)
  let repos = $state<ProjectRepoRecord[]>([])
  let selectedRepoId = $state('')
  let loadedRepos = $state(false)

  const hasTickets = $derived(ticketCount > 0)

  $effect(() => {
    if (loadedRepos) return
    const load = async () => {
      try {
        const payload = await listProjectRepos(projectId)
        repos = payload.repos
        if (payload.repos.length === 1) {
          selectedRepoId = payload.repos[0]!.id
        }
        loadedRepos = true
      } catch {
        // ignore
      }
    }
    void load()
  })

  // Find the selected workflow's pickup status name
  const selectedWorkflow = $derived(workflows.find((w) => w.id === selectedWorkflowId))
  const pickupStatusLabel = $derived(
    selectedWorkflow?.pickup_status_ids?.length ? `进入 Workflow 的 Pickup 状态` : '—',
  )

  async function handleCreate() {
    if (!title.trim()) {
      toastStore.error('请输入 Ticket 标题。')
      return
    }
    creating = true
    try {
      const payload = await createTicket(projectId, {
        title: title.trim(),
        description: description.trim() || undefined,
        workflow_id: selectedWorkflowId || undefined,
        repo_scopes: selectedRepoId ? [{ repo_id: selectedRepoId }] : undefined,
      })

      toastStore.success(`Ticket ${payload.ticket.identifier} 已创建。`)
      onComplete()

      // Navigate to ticket detail by opening right panel
      appStore.openRightPanel({ type: 'ticket', id: payload.ticket.id })
    } catch (caughtError) {
      toastStore.error(caughtError instanceof ApiError ? caughtError.detail : '创建 Ticket 失败。')
    } finally {
      creating = false
    }
  }
</script>

<div class="space-y-4">
  {#if hasTickets}
    <div
      class="flex items-center gap-3 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 dark:border-emerald-900/50 dark:bg-emerald-950/30"
    >
      <CheckCircle2 class="size-4 shrink-0 text-emerald-600 dark:text-emerald-400" />
      <Ticket class="text-muted-foreground size-4 shrink-0" />
      <div>
        <p class="text-foreground text-sm font-medium">已创建 {ticketCount} 个 Ticket</p>
        <p class="text-muted-foreground text-xs">前往 Tickets 页面查看详情</p>
      </div>
      <Button
        variant="outline"
        size="sm"
        class="ml-auto"
        onclick={() => void goto(projectPath(orgId, projectId, 'tickets'))}
      >
        查看 Tickets
      </Button>
    </div>
  {:else}
    <div class="space-y-3">
      <div>
        <p class="text-foreground mb-1 text-xs font-medium">Ticket 标题</p>
        <Input
          bind:value={title}
          placeholder={preset.exampleTicketTitle}
          class="text-sm"
          autofocus
        />
      </div>

      <div>
        <p class="text-foreground mb-1 text-xs font-medium">描述（可选）</p>
        <Textarea bind:value={description} placeholder="描述任务内容..." rows={2} class="text-sm" />
      </div>

      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        {#if workflows.length > 0}
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">Workflow</p>
            <Select.Root
              type="single"
              value={selectedWorkflowId}
              onValueChange={(v) => {
                if (v) selectedWorkflowId = v
              }}
            >
              <Select.Trigger class="h-9 w-full text-sm">
                {selectedWorkflow?.name ?? '选择 Workflow'}
              </Select.Trigger>
              <Select.Content>
                {#each workflows as wf (wf.id)}
                  <Select.Item value={wf.id}>{wf.name}</Select.Item>
                {/each}
              </Select.Content>
            </Select.Root>
          </div>
        {/if}

        {#if repos.length > 1}
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">Repo 范围</p>
            <Select.Root
              type="single"
              value={selectedRepoId}
              onValueChange={(v) => {
                if (v) selectedRepoId = v
              }}
            >
              <Select.Trigger class="h-9 w-full text-sm">
                {repos.find((r) => r.id === selectedRepoId)?.name ?? '选择 Repo'}
              </Select.Trigger>
              <Select.Content>
                {#each repos as repo (repo.id)}
                  <Select.Item value={repo.id}>{repo.name}</Select.Item>
                {/each}
              </Select.Content>
            </Select.Root>
          </div>
        {/if}
      </div>

      <!-- Info about what happens next -->
      <div class="bg-muted/50 flex items-start gap-2 rounded-md p-3">
        <Info class="text-muted-foreground mt-0.5 size-3.5 shrink-0" />
        <div class="text-muted-foreground space-y-1 text-xs">
          <p>工单将{pickupStatusLabel}，编排引擎会自动领取并分配给 Agent。</p>
          <p>Agent 的工作进展会实时出现在 Ticket 详情页的时间线中。</p>
        </div>
      </div>

      <Button class="w-full" onclick={handleCreate} disabled={creating || !title.trim()}>
        {#if creating}
          <Loader2 class="mr-1.5 size-3.5 animate-spin" />
          创建中...
        {:else}
          <Ticket class="mr-1.5 size-3.5" />
          创建 Ticket
        {/if}
      </Button>
    </div>
  {/if}
</div>
