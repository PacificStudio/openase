<script lang="ts">
  import { BellDot, Send, ShieldCheck } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
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

<div class="space-y-6">
  <Card class="border-border/80 bg-background/80 overflow-hidden backdrop-blur">
    <CardHeader class="gap-4">
      <div class="flex flex-wrap items-center gap-2">
        <Badge variant="outline">Notification management</Badge>
        <Badge variant="outline">Formal API</Badge>
        <Badge variant="outline">Phase 2 feature split</Badge>
      </div>
      <div class="grid gap-5 xl:grid-cols-[minmax(0,1.2fr)_18rem]">
        <div class="space-y-3">
          <CardTitle class="text-2xl tracking-[-0.04em]">
            Channels, routing rules, and test sends now live outside the route layer.
          </CardTitle>
          <CardDescription class="max-w-3xl text-sm leading-7">
            Configure organization-level delivery channels, bind project rules to supported ticket
            events, and send a live test message before rolling the workflow out.
          </CardDescription>
        </div>

        <CardContent class="grid gap-3 p-0">
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
        </CardContent>
      </div>
    </CardHeader>
  </Card>

  <div class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
    <NotificationChannelsPanel {controller} />
    <NotificationRulesPanel {controller} />
  </div>
</div>
