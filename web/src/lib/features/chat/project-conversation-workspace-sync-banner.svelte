<script lang="ts">
  import type { ProjectConversationWorkspaceSyncPrompt } from '$lib/api/chat'
  import { Button } from '$ui/button'
  import { AlertCircle } from '@lucide/svelte'

  let {
    prompt,
    syncError = '',
    syncInFlight = false,
    centered = false,
    onSync,
  }: {
    prompt: ProjectConversationWorkspaceSyncPrompt
    syncError?: string
    syncInFlight?: boolean
    centered?: boolean
    onSync?: () => void | Promise<void>
  } = $props()

  const title = $derived(
    prompt.reason === 'repo_binding_changed'
      ? 'Workspace sync required'
      : 'Some project repos are missing from this workspace',
  )
  const description = $derived(
    prompt.reason === 'repo_binding_changed'
      ? 'This conversation workspace was prepared before the latest project repo binding changes. Newly bound repos have not been cloned into this workspace yet, so browse and diff can be incomplete until you sync.'
      : 'One or more repos are bound to this project but are still missing from the current conversation workspace. Sync the workspace to clone them before browsing or diffing.',
  )
  const missingRepos = $derived(prompt.missingRepos.map((repo) => repo.path).join(', '))
</script>

{#if centered}
  <div class="flex flex-1 items-center justify-center px-6">
    <div class="border-border bg-muted/20 max-w-lg rounded-xl border p-5 text-left">
      <p class="text-sm font-medium">{title}</p>
      <p class="text-muted-foreground mt-2 text-sm">{description}</p>
      <p class="text-muted-foreground mt-3 text-xs">Missing repos: {missingRepos}</p>
      {#if syncError}
        <p class="text-destructive mt-3 text-xs">{syncError}</p>
      {/if}
      <div class="mt-4 flex gap-2">
        <Button size="sm" onclick={() => void onSync?.()} disabled={syncInFlight}>
          {syncInFlight ? 'Syncing repos...' : 'Sync repos'}
        </Button>
      </div>
    </div>
  </div>
{:else}
  <div
    class="border-border border-b bg-amber-50/80 px-3 py-2 text-amber-950 dark:bg-amber-500/10 dark:text-amber-100"
  >
    <div class="flex items-start gap-3">
      <AlertCircle class="mt-0.5 size-4 shrink-0" />
      <div class="min-w-0 flex-1">
        <p class="text-sm font-medium">{title}</p>
        <p class="mt-1 text-xs leading-5">{description}</p>
        <p class="mt-2 text-xs">Missing repos: {missingRepos}</p>
        {#if syncError}
          <p class="text-destructive mt-2 text-xs">{syncError}</p>
        {/if}
      </div>
      <Button
        size="sm"
        variant="secondary"
        class="shrink-0"
        onclick={() => void onSync?.()}
        disabled={syncInFlight}
      >
        {syncInFlight ? 'Syncing...' : 'Sync repos'}
      </Button>
    </div>
  </div>
{/if}
