<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { EphemeralChatPanel } from '$lib/features/chat'
  import * as Dialog from '$ui/dialog'

  let {
    open = $bindable(false),
    projectId,
    cronContextNote,
    cronMessagePrefix,
  }: {
    open?: boolean
    projectId: string
    cronContextNote: string
    cronMessagePrefix: string
  } = $props()
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="flex h-[80vh] max-h-[48rem] max-w-3xl flex-col overflow-hidden p-0">
    <Dialog.Header class="border-border border-b px-6 py-5">
      <Dialog.Title>Cron helper</Dialog.Title>
      <Dialog.Description>
        Ask AI to translate natural-language schedules, review an existing cron expression, or
        suggest safer timing.
      </Dialog.Description>
    </Dialog.Header>

    {#if open}
      <EphemeralChatPanel
        source="project_sidebar"
        organizationId={appStore.currentOrg?.id ?? ''}
        defaultProviderId={appStore.currentProject?.default_agent_provider_id ?? null}
        context={{ projectId }}
        title="Scheduled Job AI"
        description="Project context plus cron-specific guidance."
        placeholder="Describe the schedule you want, or ask what the current cron expression means."
        emptyStateTitle="Cron context is ready"
        emptyStateDescription="Ask for cron translation, validation, or safer scheduling guidance for this recurring job."
        contextNote={cronContextNote}
        messagePrefix={cronMessagePrefix}
      />
    {/if}
  </Dialog.Content>
</Dialog.Root>
