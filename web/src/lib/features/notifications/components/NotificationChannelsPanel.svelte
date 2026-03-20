<script lang="ts">
  import { Bell, PencilLine, Send, Trash2, Zap } from '@lucide/svelte'
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
  import { channelTypeLabel, configSummaryLines } from '../types'

  let {
    controller,
  }: {
    controller: ReturnType<typeof createNotificationsController>
  } = $props()

  const types = ['webhook', 'telegram', 'slack', 'wecom'] as const
</script>

<Card class="border-border/80 bg-background/85 backdrop-blur">
  <CardHeader>
    <CardTitle class="flex items-center gap-2">
      <Bell class="size-4" />
      <span>Channels</span>
    </CardTitle>
    <CardDescription>
      Configure delivery targets at the organization level. Test sends use the formal notification
      API, not local mocks.
    </CardDescription>
  </CardHeader>

  <CardContent class="space-y-5">
    <ScrollPane class="max-h-[22rem]">
      <div class="grid gap-3">
      {#if controller.state.channels.length === 0}
        <div
          class="border-border/70 bg-muted/30 rounded-3xl border border-dashed px-4 py-5 text-sm"
        >
          No notification channels yet.
        </div>
      {:else}
        {#each controller.state.channels as channel}
          <article
            class={`rounded-3xl border px-4 py-4 transition ${
              controller.state.selectedChannelId === channel.id
                ? 'border-foreground/30 bg-foreground/[0.04]'
                : 'border-border/70 bg-background/60'
            }`}
          >
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="space-y-2">
                <div class="flex flex-wrap items-center gap-2">
                  <h3 class="text-sm font-semibold">{channel.name}</h3>
                  <Badge variant="outline">{channelTypeLabel(channel.type)}</Badge>
                  <Badge variant={channel.is_enabled ? 'secondary' : 'outline'}>
                    {channel.is_enabled ? 'Enabled' : 'Disabled'}
                  </Badge>
                </div>

                <div class="text-muted-foreground space-y-1 text-xs leading-5">
                  {#each configSummaryLines(channel) as line}
                    <p>{line}</p>
                  {/each}
                </div>
              </div>

              <div class="flex flex-wrap gap-2">
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onclick={() => controller.beginChannelEdit(channel.id)}
                >
                  <PencilLine class="size-4" />
                  Edit
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  disabled={controller.state.testingChannelId === channel.id}
                  onclick={() => void controller.sendChannelTest(channel.id)}
                >
                  <Send class="size-4" />
                  {controller.state.testingChannelId === channel.id ? 'Sending…' : 'Test send'}
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="destructive"
                  onclick={() => void controller.removeChannel(channel.id)}
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
      class="space-y-4 rounded-[1.75rem] border border-amber-500/20 bg-amber-500/5 p-5"
      onsubmit={(event) => {
        event.preventDefault()
        void controller.saveChannel()
      }}
    >
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p class="text-sm font-semibold">
            {controller.state.channelMode === 'create' ? 'Create channel' : 'Edit channel'}
          </p>
          <p class="text-muted-foreground mt-1 text-xs leading-5">
            {#if controller.state.channelMode === 'create'}
              Add a new delivery endpoint for project rules to target.
            {:else}
              Existing transport config stays unchanged until you enable replacement below.
            {/if}
          </p>
        </div>

        {#if controller.state.channelMode === 'edit'}
          <Button type="button" size="sm" variant="ghost" onclick={controller.resetChannelForm}>
            Create new instead
          </Button>
        {/if}
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <label class="space-y-2 text-sm">
          <span class="font-medium">Name</span>
          <input
            class="border-border/70 bg-background/80 w-full rounded-2xl border px-3 py-2.5"
            bind:value={controller.state.channelForm.name}
            placeholder="Ops alerts"
          />
        </label>

        <label class="space-y-2 text-sm">
          <span class="font-medium">Channel type</span>
          <select
            class="border-border/70 bg-background/80 w-full rounded-2xl border px-3 py-2.5"
            bind:value={controller.state.channelForm.type}
            onchange={() => controller.setChannelType(controller.state.channelForm.type)}
          >
            {#each types as type}
              <option value={type}>{channelTypeLabel(type)}</option>
            {/each}
          </select>
        </label>
      </div>

      <label class="flex items-center gap-3 text-sm">
        <input type="checkbox" bind:checked={controller.state.channelForm.isEnabled} />
        <span>Channel enabled</span>
      </label>

      {#if controller.state.channelMode === 'edit'}
        <label class="flex items-center gap-3 text-sm">
          <input
            type="checkbox"
            bind:checked={controller.state.channelReplaceConfig}
            onchange={() =>
              controller.setChannelReplaceConfig(controller.state.channelReplaceConfig)}
          />
          <span>Replace transport config</span>
        </label>
      {/if}

      {#if controller.state.channelMode === 'create' || controller.state.channelReplaceConfig}
        <div class="grid gap-4">
          {#if controller.state.channelForm.type === 'webhook'}
            <label class="space-y-2 text-sm">
              <span class="font-medium">Webhook URL</span>
              <input
                class="border-border/70 bg-background/80 w-full rounded-2xl border px-3 py-2.5"
                bind:value={controller.state.channelForm.webhookURL}
                placeholder="https://hooks.example.com/openase"
              />
            </label>
            <label class="space-y-2 text-sm">
              <span class="font-medium">Headers JSON</span>
              <textarea
                class="border-border/70 bg-background/80 min-h-24 w-full rounded-2xl border px-3 py-2.5 font-mono text-xs"
                bind:value={controller.state.channelForm.webhookHeaders}
              ></textarea>
            </label>
            <label class="space-y-2 text-sm">
              <span class="font-medium">Signing secret</span>
              <input
                class="border-border/70 bg-background/80 w-full rounded-2xl border px-3 py-2.5"
                bind:value={controller.state.channelForm.webhookSecret}
                placeholder="Optional HMAC secret"
              />
            </label>
          {:else if controller.state.channelForm.type === 'telegram'}
            <label class="space-y-2 text-sm">
              <span class="font-medium">Bot token</span>
              <input
                class="border-border/70 bg-background/80 w-full rounded-2xl border px-3 py-2.5"
                bind:value={controller.state.channelForm.telegramBotToken}
              />
            </label>
            <label class="space-y-2 text-sm">
              <span class="font-medium">Chat ID</span>
              <input
                class="border-border/70 bg-background/80 w-full rounded-2xl border px-3 py-2.5"
                bind:value={controller.state.channelForm.telegramChatID}
              />
            </label>
          {:else if controller.state.channelForm.type === 'slack'}
            <label class="space-y-2 text-sm">
              <span class="font-medium">Incoming webhook URL</span>
              <input
                class="border-border/70 bg-background/80 w-full rounded-2xl border px-3 py-2.5"
                bind:value={controller.state.channelForm.slackWebhookURL}
              />
            </label>
          {:else}
            <label class="space-y-2 text-sm">
              <span class="font-medium">WeCom webhook key</span>
              <input
                class="border-border/70 bg-background/80 w-full rounded-2xl border px-3 py-2.5"
                bind:value={controller.state.channelForm.wecomWebhookKey}
              />
            </label>
          {/if}
        </div>
      {:else}
        <div
          class="border-border/70 bg-background/50 rounded-2xl border px-4 py-3 text-xs leading-5"
        >
          Current delivery config is preserved server-side. Enable <span class="font-medium">
            Replace transport config
          </span> to submit a new secret, URL, or webhook target.
        </div>
      {/if}

      {#if controller.state.channelNotice}
        <div class="rounded-2xl border border-emerald-500/25 bg-emerald-500/10 px-4 py-3 text-sm">
          <div class="flex items-center gap-2">
            <Zap class="size-4" />
            <span>{controller.state.channelNotice}</span>
          </div>
        </div>
      {/if}

      {#if controller.state.channelError}
        <div
          class="text-destructive border-destructive/25 bg-destructive/10 rounded-2xl border px-4 py-3 text-sm"
        >
          {controller.state.channelError}
        </div>
      {/if}

      <div class="flex flex-wrap gap-3">
        <Button type="submit" disabled={controller.state.channelBusy}>
          {controller.state.channelBusy
            ? 'Saving…'
            : controller.state.channelMode === 'create'
              ? 'Create channel'
              : 'Save channel'}
        </Button>
        {#if controller.state.channelMode === 'edit'}
          <Button type="button" variant="outline" onclick={controller.resetChannelForm}>
            Cancel
          </Button>
        {/if}
      </div>
    </form>
  </CardContent>
</Card>
