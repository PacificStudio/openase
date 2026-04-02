<script lang="ts">
  import { formatBytes, formatCount, formatCurrency } from '$lib/utils'
  import { appStore } from '$lib/stores/app.svelte'
  import { Skeleton } from '$ui/skeleton'
  import ActivityFeedPanel from './activity-feed-panel.svelte'
  import HRAdvisorPanel from './hr-advisor-panel.svelte'
  import OrgDashboardHeader from './org-dashboard-header.svelte'
  import OrgDashboardUpdatesPanel from './org-dashboard-updates-panel.svelte'
  import { createOrgDashboardController } from './org-dashboard-controller.svelte'
  import { OnboardingPanel } from '$lib/features/onboarding'
  import { Bot, Coins, Ticket } from '@lucide/svelte'

  const controller = createOrgDashboardController()
</script>

<div class="flex min-h-0 flex-col">
  <OrgDashboardHeader
    editingInfo={controller.editingInfo}
    editName={controller.editName}
    editDescription={controller.editDescription}
    savingInfo={controller.savingInfo}
    savingStatus={controller.savingStatus}
    currentStatus={controller.currentStatus}
    projectName={controller.projectName}
    projectDescription={controller.projectDescription}
    onEditNameChange={controller.setEditName}
    onEditDescriptionChange={controller.setEditDescription}
    onStartEditInfo={controller.startEditInfo}
    onCancelEditInfo={controller.cancelEditInfo}
    onSaveInfo={controller.saveInfo}
    onProjectStatusChange={controller.handleProjectStatusChange}
  />

  <div class="min-h-0 flex-1 overflow-y-auto px-4 py-4 pb-8 sm:px-6">
    {#if controller.showOnboarding && appStore.currentProject && appStore.currentOrg}
      <OnboardingPanel
        projectId={appStore.currentProject.id}
        orgId={appStore.currentOrg.id}
        projectName={controller.projectName}
        projectStatus={controller.currentStatus}
        onOnboardingComplete={() => {
          controller.dismissOnboarding(appStore.currentProject!.id)
        }}
      />
    {:else}
      <div class="space-y-3">
        {#if controller.error && !controller.loading}
          <div
            class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-3 py-2 text-sm"
          >
            {controller.error}
          </div>
        {/if}

        <div
          class="border-border bg-card flex flex-wrap items-center gap-x-5 gap-y-1.5 rounded-md border px-3 py-2"
        >
          {#if controller.loading}
            {#each { length: 6 } as _}
              <div class="flex items-center gap-1.5">
                <Skeleton class="h-3 w-12" />
                <Skeleton class="h-3.5 w-8" />
              </div>
            {/each}
          {:else}
            <div class="flex items-center gap-1">
              <Bot class="text-muted-foreground size-3" />
              <span class="text-muted-foreground text-[11px]">Agents</span>
              <span class="text-foreground text-xs font-semibold"
                >{controller.stats.runningAgents}</span
              >
            </div>
            <div class="flex items-center gap-1">
              <Ticket class="text-muted-foreground size-3" />
              <span class="text-muted-foreground text-[11px]">Tickets</span>
              <span class="text-foreground text-xs font-semibold"
                >{controller.stats.activeTickets}</span
              >
            </div>
            <div class="flex items-center gap-1">
              <Coins class="text-muted-foreground size-3" />
              <span class="text-muted-foreground text-[11px]">Spend</span>
              <span class="text-foreground text-xs font-semibold"
                >{formatCurrency(controller.stats.ticketSpendToday)}</span
              >
            </div>
            <div class="flex items-center gap-1">
              <span class="text-muted-foreground text-[11px]">Tokens</span>
              <span class="text-foreground text-xs font-semibold"
                >{formatCount(controller.totalTicketTokens)}</span
              >
            </div>
            <div class="flex items-center gap-1">
              <span class="text-muted-foreground text-[11px]">Heap</span>
              <span class="text-foreground text-xs font-semibold"
                >{controller.memory ? formatBytes(controller.memory.heap_inuse_bytes) : '—'}</span
              >
            </div>
            {#if controller.exceptions.length > 0}
              <div class="flex items-center gap-1">
                <span
                  class="bg-destructive/10 text-destructive inline-flex size-4 items-center justify-center rounded-full text-[9px] font-semibold"
                  >{controller.exceptions.length}</span
                >
                <span class="text-destructive text-[11px]">
                  {controller.exceptions.length === 1 ? 'exception' : 'exceptions'}
                </span>
              </div>
            {/if}
          {/if}
        </div>

        {#if controller.loading}
          <div class="grid grid-cols-1 gap-3 lg:grid-cols-2">
            {#each { length: 2 } as _}
              <div class="border-border bg-card rounded-md border">
                <div class="border-border flex items-center justify-between border-b px-3 py-2">
                  <Skeleton class="h-3.5 w-20" />
                  <Skeleton class="h-3 w-10" />
                </div>
                <div class="space-y-2.5 p-3">
                  {#each { length: 4 } as _}
                    <div class="flex items-start gap-2">
                      <Skeleton class="mt-0.5 size-3.5 shrink-0 rounded-full" />
                      <div class="flex-1 space-y-1">
                        <Skeleton class="h-3 w-3/4" />
                        <Skeleton class="h-2.5 w-1/3" />
                      </div>
                    </div>
                  {/each}
                </div>
              </div>
            {/each}
          </div>
        {:else if !controller.error}
          <div class="grid grid-cols-1 gap-3 lg:grid-cols-2">
            <div class="flex min-h-0 flex-col">
              <ActivityFeedPanel activities={controller.activities} />

              {#if controller.hrAdvisor && appStore.currentProject}
                <div class="mt-3">
                  {#key appStore.currentProject.id}
                    <HRAdvisorPanel
                      projectId={appStore.currentProject.id}
                      advisor={controller.hrAdvisor}
                    />
                  {/key}
                </div>
              {/if}
            </div>

            <OrgDashboardUpdatesPanel
              threads={controller.projectUpdates.threads}
              loading={controller.projectUpdates.loading}
              initialLoaded={controller.projectUpdates.initialLoaded}
              creatingThread={controller.projectUpdates.creatingThread}
              error={controller.projectUpdates.error}
              notice={controller.projectUpdates.notice}
              onSubmit={controller.projectUpdates.handleCreateThread}
              onUpdateThread={controller.projectUpdates.handleSaveThread}
              onDeleteThread={controller.projectUpdates.handleDeleteThread}
              onCreateComment={controller.projectUpdates.handleCreateComment}
              onUpdateComment={controller.projectUpdates.handleSaveComment}
              onDeleteComment={controller.projectUpdates.handleDeleteComment}
            />
          </div>
        {/if}
      </div>
    {/if}
  </div>
</div>
