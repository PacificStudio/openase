<script lang="ts">
  import { cn } from '$lib/utils'
  import Button from '$ui/button/button.svelte'
  import { Plus, PanelRightClose, PanelRight } from '@lucide/svelte'
  import type { WorkflowSummary, HarnessContent } from '../types'
  import WorkflowList from './workflow-list.svelte'
  import HarnessEditor from './harness-editor.svelte'
  import WorkflowDetailPanel from './workflow-detail-panel.svelte'

  let showDetail = $state(true)
  let selectedId = $state('wf-coding')

  const workflows: WorkflowSummary[] = [
    {
      id: 'wf-coding', name: 'Coding Agent', type: 'coding',
      pickupStatus: 'ready_for_dev', finishStatus: 'in_review',
      maxConcurrent: 3, maxRetry: 2, timeoutMinutes: 30,
      isActive: true, lastModified: '2026-03-20T08:30:00Z',
      recentSuccessRate: 87, version: 4,
    },
    {
      id: 'wf-test', name: 'Test Suite Runner', type: 'test',
      pickupStatus: 'needs_tests', finishStatus: 'tests_passing',
      maxConcurrent: 5, maxRetry: 3, timeoutMinutes: 15,
      isActive: true, lastModified: '2026-03-19T14:00:00Z',
      recentSuccessRate: 92, version: 2,
    },
    {
      id: 'wf-security', name: 'Security Scan', type: 'security',
      pickupStatus: 'pending_scan', finishStatus: 'scan_complete',
      maxConcurrent: 2, maxRetry: 1, timeoutMinutes: 45,
      isActive: true, lastModified: '2026-03-18T10:15:00Z',
      recentSuccessRate: 78, version: 3,
    },
    {
      id: 'wf-docs', name: 'Documentation Gen', type: 'doc',
      pickupStatus: 'needs_docs', finishStatus: 'docs_ready',
      maxConcurrent: 2, maxRetry: 1, timeoutMinutes: 20,
      isActive: false, lastModified: '2026-03-15T16:45:00Z',
      recentSuccessRate: 95, version: 1,
    },
    {
      id: 'wf-deploy', name: 'Deploy Pipeline', type: 'deploy',
      pickupStatus: 'approved', finishStatus: 'deployed',
      maxConcurrent: 1, maxRetry: 2, timeoutMinutes: 60,
      isActive: true, lastModified: '2026-03-20T06:00:00Z',
      recentSuccessRate: 64, version: 5,
    },
  ]

  const harnessMap: Record<string, HarnessContent> = {
    'wf-coding': {
      frontmatter: 'type: coding\npickup_status: ready_for_dev\nfinish_status: in_review',
      body: 'You are a coding agent.\n\nGiven a ticket, implement the required changes:\n1. Read the ticket description and acceptance criteria\n2. Explore the codebase for context\n3. Write clean, tested code\n4. Create a pull request with a clear description\n\nConstraints:\n- Follow existing code style\n- Add unit tests for new logic\n- Keep PRs under 400 lines when possible',
      rawContent: '---\ntype: coding\npickup_status: ready_for_dev\nfinish_status: in_review\nmax_concurrent: 3\ntimeout_minutes: 30\n---\n\nYou are a coding agent.\n\nGiven a ticket, implement the required changes:\n1. Read the ticket description and acceptance criteria\n2. Explore the codebase for context\n3. Write clean, tested code\n4. Create a pull request with a clear description\n\nConstraints:\n- Follow existing code style\n- Add unit tests for new logic\n- Keep PRs under 400 lines when possible',
    },
    'wf-test': {
      frontmatter: 'type: test\npickup_status: needs_tests\nfinish_status: tests_passing',
      body: 'You are a test runner agent.\n\nRun the full test suite and report results.\nOn failure, analyze logs and suggest fixes.',
      rawContent: '---\ntype: test\npickup_status: needs_tests\nfinish_status: tests_passing\nmax_concurrent: 5\n---\n\nYou are a test runner agent.\n\nRun the full test suite and report results.\nOn failure, analyze logs and suggest fixes.',
    },
    'wf-security': {
      frontmatter: 'type: security\npickup_status: pending_scan',
      body: 'You are a security scanning agent.\n\nScan dependencies and code for vulnerabilities.\nReport findings with severity levels.',
      rawContent: '---\ntype: security\npickup_status: pending_scan\nfinish_status: scan_complete\n---\n\nYou are a security scanning agent.\n\nScan dependencies and code for vulnerabilities.\nReport findings with severity levels.',
    },
    'wf-docs': {
      frontmatter: 'type: doc\npickup_status: needs_docs',
      body: 'You are a documentation agent.\n\nGenerate or update documentation based on code changes.',
      rawContent: '---\ntype: doc\npickup_status: needs_docs\nfinish_status: docs_ready\n---\n\nYou are a documentation agent.\n\nGenerate or update documentation based on code changes.',
    },
    'wf-deploy': {
      frontmatter: 'type: deploy\npickup_status: approved',
      body: 'You are a deploy agent.\n\nExecute the deploy pipeline after approval.',
      rawContent: '---\ntype: deploy\npickup_status: approved\nfinish_status: deployed\nmax_concurrent: 1\n---\n\nYou are a deploy agent.\n\nExecute the deploy pipeline after approval.',
    },
  }

  let selectedWorkflow = $derived(workflows.find((w) => w.id === selectedId))
  let selectedHarness = $derived(harnessMap[selectedId])
</script>

<div class="flex h-full flex-col">
  <div class="flex items-center justify-between border-b border-border px-4 py-2.5">
    <h1 class="text-sm font-semibold text-foreground">Workflows</h1>
    <div class="flex items-center gap-2">
      <Button variant="ghost" size="sm" onclick={() => (showDetail = !showDetail)}>
        {#if showDetail}
          <PanelRightClose class="size-4" />
        {:else}
          <PanelRight class="size-4" />
        {/if}
      </Button>
      <Button size="sm">
        <Plus class="size-4" />
        New Workflow
      </Button>
    </div>
  </div>

  <div class="flex flex-1 overflow-hidden">
    <div class="w-60 shrink-0">
      <WorkflowList
        {workflows}
        {selectedId}
        onselect={(id) => (selectedId = id)}
      />
    </div>

    <div class="flex-1 overflow-hidden">
      {#if selectedHarness}
        <HarnessEditor
          content={selectedHarness}
          filePath="harness/{selectedId}.md"
          version={selectedWorkflow?.version ?? 1}
        />
      {/if}
    </div>

    {#if showDetail && selectedWorkflow}
      <div class="w-70 shrink-0">
        <WorkflowDetailPanel workflow={selectedWorkflow} />
      </div>
    {/if}
  </div>
</div>
