<script lang="ts">
  import type { BuiltinRole } from '$lib/api/contracts'
  import { normalizeWorkflowFamily, workflowFamilyColors } from '$lib/features/workflows'
  import { Badge } from '$ui/badge'
  import * as Dialog from '$ui/dialog'

  let {
    open = $bindable(false),
    harness,
    roleName = '',
    loading = false,
    error = '',
  }: {
    open?: boolean
    harness: BuiltinRole | null
    roleName?: string
    loading?: boolean
    error?: string
  } = $props()

  const workflowFamily = $derived(normalizeWorkflowFamily(harness?.workflow_family ?? ''))
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="flex h-[80vh] max-h-[48rem] max-w-4xl flex-col overflow-hidden p-0">
    <Dialog.Header class="border-border border-b px-6 py-5">
      <Dialog.Title>{harness?.name || roleName || '角色 Harness'}</Dialog.Title>
      <Dialog.Description>
        {harness?.summary || 'Builtin role harness used by HR Advisor one-click activation.'}
      </Dialog.Description>
    </Dialog.Header>

    <div class="space-y-4 overflow-auto px-6 py-5">
      {#if harness}
        <div class="flex flex-wrap gap-2 text-xs">
          <Badge variant="outline">{harness.slug}</Badge>
          <Badge variant="outline">{harness.workflow_type}</Badge>
          <Badge variant="outline" class={workflowFamilyColors[workflowFamily]}>
            {harness.workflow_family}
          </Badge>
          <Badge variant="outline">{harness.harness_path}</Badge>
        </div>
      {/if}

      {#if loading}
        <p class="text-muted-foreground text-sm">Loading role harness…</p>
      {:else if error}
        <div
          class="border-destructive/30 bg-destructive/10 text-destructive rounded-md border px-3 py-2 text-sm"
        >
          {error}
        </div>
      {:else if harness}
        <pre
          class="bg-muted/30 text-foreground overflow-x-auto rounded-md border p-4 text-xs leading-5 whitespace-pre-wrap">{harness.content}</pre>
      {:else}
        <p class="text-muted-foreground text-sm">No harness preview available for this role.</p>
      {/if}
    </div>
  </Dialog.Content>
</Dialog.Root>
