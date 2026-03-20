<script lang="ts">
  import { BellDot, Send, ShieldCheck } from '@lucide/svelte'
  import MasterDetailLayout from '$lib/components/layout/MasterDetailLayout.svelte'
  import SurfacePanel from '$lib/components/layout/SurfacePanel.svelte'
  import { Badge } from '$lib/components/ui/badge'
  import type { createNotificationsController } from '../controller.svelte'
  import NotificationChannelsPanel from './NotificationChannelsPanel.svelte'
  import NotificationRulesPanel from './NotificationRulesPanel.svelte'

  let {
    controller,
  }: {
    controller: ReturnType<typeof createNotificationsController>
  } = $props()
</script>

<svelte:head>
  <title>Notifications · OpenASE</title>
</svelte:head>

<div class="space-y-4">
  <SurfacePanel>
    {#snippet header()}
      <div class="flex flex-wrap items-center gap-2">
        <Badge variant="outline">Settings</Badge>
        <Badge variant="outline">Notification delivery</Badge>
      </div>
    {/snippet}

    <div class="grid gap-3 p-4 xl:grid-cols-3">
          <div class="rounded-3xl border border-amber-500/20 bg-amber-500/8 px-4 py-4">
            <div class="flex items-center gap-2 text-sm font-semibold">
              <BellDot class="size-4" />
              <span>{controller.state.channels.length} channels</span>
            </div>
            <p class="text-muted-foreground mt-2 text-xs">Scoped to the selected organization.</p>
          </div>
          <div class="rounded-3xl border border-teal-500/20 bg-teal-500/8 px-4 py-4">
            <div class="flex items-center gap-2 text-sm font-semibold">
              <ShieldCheck class="size-4" />
              <span>{controller.state.rules.length} rules</span>
            </div>
            <p class="text-muted-foreground mt-2 text-xs">Scoped to the selected project.</p>
          </div>
          <div class="rounded-3xl border border-sky-500/20 bg-sky-500/8 px-4 py-4">
            <div class="flex items-center gap-2 text-sm font-semibold">
              <Send class="size-4" />
              <span>{controller.state.eventTypes.length} event types</span>
            </div>
            <p class="text-muted-foreground mt-2 text-xs">
              Ticket lifecycle events available for subscriptions.
            </p>
          </div>
    </div>
  </SurfacePanel>

  <MasterDetailLayout detailWidthClass="xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
    {#snippet main()}
      <NotificationChannelsPanel {controller} />
    {/snippet}

    {#snippet detail()}
      <NotificationRulesPanel {controller} />
    {/snippet}
  </MasterDetailLayout>
</div>
