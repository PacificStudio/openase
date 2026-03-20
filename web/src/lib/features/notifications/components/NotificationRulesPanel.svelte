<script lang="ts">
  import { BellRing, FileText, PencilLine, Trash2 } from '@lucide/svelte'
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
  import type { createNotificationsController } from '../controller.svelte'

  let {
    controller,
  }: {
    controller: ReturnType<typeof createNotificationsController>
  } = $props()

  function eventLabel(eventType: string) {
    return (
      controller.state.eventTypes.find((item) => item.event_type === eventType)?.label ?? eventType
    )
  }
</script>

<Card class="border-border/80 bg-background/85 backdrop-blur">
  <CardHeader>
    <CardTitle class="flex items-center gap-2">
      <BellRing class="size-4" />
      <span>Rules</span>
    </CardTitle>
    <CardDescription>
      Scope rules to the selected project. Filters accept JSON objects and templates render with the
      event payload.
    </CardDescription>
  </CardHeader>

  <CardContent class="space-y-5">
    <div class="flex flex-wrap gap-2">
      {#each controller.state.eventTypes as eventType}
        <Badge variant="outline">{eventType.label}</Badge>
      {/each}
    </div>

    <ScrollPane class="max-h-[22rem]">
      <div class="grid gap-3">
      {#if controller.state.rules.length === 0}
        <div
          class="border-border/70 bg-muted/30 rounded-3xl border border-dashed px-4 py-5 text-sm"
        >
          No notification rules yet for this project.
        </div>
      {:else}
        {#each controller.state.rules as rule}
          <article
            class={`rounded-3xl border px-4 py-4 transition ${
              controller.state.selectedRuleId === rule.id
                ? 'border-foreground/30 bg-foreground/[0.04]'
                : 'border-border/70 bg-background/60'
            }`}
          >
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="space-y-2">
                <div class="flex flex-wrap items-center gap-2">
                  <h3 class="text-sm font-semibold">{rule.name}</h3>
                  <Badge variant="outline">{eventLabel(rule.event_type)}</Badge>
                  <Badge variant="outline">{rule.channel.name}</Badge>
                  <Badge variant={rule.is_enabled ? 'secondary' : 'outline'}>
                    {rule.is_enabled ? 'Enabled' : 'Disabled'}
                  </Badge>
                </div>
                <p class="text-muted-foreground text-xs leading-5">
                  {Object.keys(rule.filter ?? {}).length > 0
                    ? JSON.stringify(rule.filter)
                    : 'No filter: all matching events trigger this rule.'}
                </p>
              </div>

              <div class="flex flex-wrap gap-2">
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onclick={() => controller.beginRuleEdit(rule.id)}
                >
                  <PencilLine class="size-4" />
                  Edit
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="destructive"
                  onclick={() => void controller.removeRule(rule.id)}
                >
                  <Trash2 class="size-4" />
                  Delete
                </Button>
              </div>
            </div>
          </article>
        {/each}
      {/if}
      </div>
    </ScrollPane>

    <form
      class="space-y-4 rounded-[1.75rem] border border-teal-500/20 bg-teal-500/5 p-5"
      onsubmit={(event) => {
        event.preventDefault()
        void controller.saveRule()
      }}
    >
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p class="text-sm font-semibold">
            {controller.state.ruleMode === 'create' ? 'Create rule' : 'Edit rule'}
          </p>
          <p class="text-muted-foreground mt-1 text-xs leading-5">
            Pair an event type with a channel, optional JSON filters, and a rendered message.
          </p>
        </div>

        {#if controller.state.ruleMode === 'edit'}
          <Button type="button" size="sm" variant="ghost" onclick={controller.resetRuleForm}>
            Create new instead
          </Button>
        {/if}
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <label class="space-y-2 text-sm">
          <span class="font-medium">Rule name</span>
          <input
            class="border-border/70 bg-background/80 w-full rounded-2xl border px-3 py-2.5"
            bind:value={controller.state.ruleForm.name}
            placeholder="Ticket created alerts"
          />
        </label>

        <label class="space-y-2 text-sm">
          <span class="font-medium">Delivery channel</span>
          <select
            class="border-border/70 bg-background/80 w-full rounded-2xl border px-3 py-2.5"
            bind:value={controller.state.ruleForm.channelId}
          >
            {#each controller.state.channels as channel}
              <option value={channel.id}>{channel.name}</option>
            {/each}
          </select>
        </label>
      </div>

      <div class="grid gap-4 md:grid-cols-[16rem_minmax(0,1fr)]">
        <label class="space-y-2 text-sm">
          <span class="font-medium">Event type</span>
          <select
            class="border-border/70 bg-background/80 w-full rounded-2xl border px-3 py-2.5"
            bind:value={controller.state.ruleForm.eventType}
            onchange={() => controller.applyRuleEventType(controller.state.ruleForm.eventType)}
          >
            {#each controller.state.eventTypes as eventType}
              <option value={eventType.event_type}>{eventType.label}</option>
            {/each}
          </select>
        </label>

        <label class="flex items-center gap-3 pt-8 text-sm">
          <input type="checkbox" bind:checked={controller.state.ruleForm.isEnabled} />
          <span>Rule enabled</span>
        </label>
      </div>

      <label class="space-y-2 text-sm">
        <span class="font-medium">Filter JSON</span>
        <textarea
          class="border-border/70 bg-background/80 min-h-24 w-full rounded-2xl border px-3 py-2.5 font-mono text-xs"
          bind:value={controller.state.ruleForm.filterText}
        ></textarea>
      </label>

      <label class="space-y-2 text-sm">
        <span class="font-medium">Message template</span>
        <textarea
          class="border-border/70 bg-background/80 min-h-32 w-full rounded-2xl border px-3 py-2.5 font-mono text-xs"
          bind:value={controller.state.ruleForm.template}
        ></textarea>
      </label>

      <div class="border-border/70 bg-background/50 rounded-2xl border px-4 py-3 text-xs leading-5">
        <div class="flex items-center gap-2 font-medium">
          <FileText class="size-4" />
          <span>Template context</span>
        </div>
        <p class="text-muted-foreground mt-2">
          Use `ticket.identifier`, `ticket.title`, `ticket.status_name`, `project_id`, `event_type`
          and any payload fields exposed by the notification engine.
        </p>
      </div>

      {#if controller.state.ruleNotice}
        <div class="rounded-2xl border border-emerald-500/25 bg-emerald-500/10 px-4 py-3 text-sm">
          {controller.state.ruleNotice}
        </div>
      {/if}

      {#if controller.state.ruleError}
        <div
          class="text-destructive border-destructive/25 bg-destructive/10 rounded-2xl border px-4 py-3 text-sm"
        >
          {controller.state.ruleError}
        </div>
      {/if}

      <div class="flex flex-wrap gap-3">
        <Button
          type="submit"
          disabled={controller.state.ruleBusy || controller.state.channels.length === 0}
        >
          {controller.state.ruleBusy
            ? 'Saving…'
            : controller.state.ruleMode === 'create'
              ? 'Create rule'
              : 'Save rule'}
        </Button>
        {#if controller.state.ruleMode === 'edit'}
          <Button type="button" variant="outline" onclick={controller.resetRuleForm}>Cancel</Button>
        {/if}
      </div>
    </form>
  </CardContent>
</Card>
