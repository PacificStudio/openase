<script lang="ts">
  import type { ScopedSecretBinding } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Ticket as TicketIcon, Trash2, Workflow as WorkflowIcon } from '@lucide/svelte'

  let {
    bindings,
    error = '',
    mutationKey = '',
    onDelete,
  }: {
    bindings: ScopedSecretBinding[]
    error?: string
    mutationKey?: string
    onDelete: (bindingId: string) => void
  } = $props()

  const sortedBindings = $derived(
    [...bindings].sort((a, b) => {
      if (a.scope !== b.scope) {
        return a.scope.localeCompare(b.scope)
      }
      if (a.target.name !== b.target.name) {
        return a.target.name.localeCompare(b.target.name)
      }
      return a.binding_key.localeCompare(b.binding_key)
    }),
  )

  function scopeBadgeVariant(scope: ScopedSecretBinding['scope']) {
    return scope === 'ticket' ? 'default' : 'secondary'
  }

  function bindingTargetLabel(binding: ScopedSecretBinding) {
    if (binding.target.identifier) {
      return `${binding.target.identifier} - ${binding.target.name}`
    }
    return binding.target.name
  }

  function bindingDeleteBusy(bindingId: string) {
    return mutationKey === `delete:${bindingId}`
  }
</script>

<div class="border-border bg-card space-y-4 rounded-2xl border p-5">
  <div class="flex items-center justify-between gap-3">
    <div>
      <div class="text-sm font-semibold">Active bindings</div>
      <p class="text-muted-foreground mt-1 text-xs leading-5">
        Review which workflows and tickets reference shared secrets before a runtime starts.
      </p>
    </div>
    <Badge variant="outline">{bindings.length}</Badge>
  </div>

  {#if error}
    <div class="text-destructive rounded-lg border px-3 py-2 text-sm">{error}</div>
  {:else if sortedBindings.length === 0}
    <div class="text-muted-foreground rounded-2xl border border-dashed px-4 py-6 text-sm">
      No workflow or ticket bindings yet. Shared secrets stay unattached until you bind them to a
      runtime target.
    </div>
  {:else}
    <div class="space-y-3">
      {#each sortedBindings as binding (binding.id)}
        <div class="border-border rounded-xl border p-4">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="space-y-2">
              <div class="flex flex-wrap items-center gap-2">
                <Badge variant={scopeBadgeVariant(binding.scope)}>
                  {binding.scope === 'ticket' ? 'Ticket override' : 'Workflow binding'}
                </Badge>
                <code class="bg-muted rounded px-2 py-1 text-xs">{binding.binding_key}</code>
              </div>

              <div class="text-sm font-medium">{bindingTargetLabel(binding)}</div>

              <div
                class="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs"
              >
                <span class="inline-flex items-center gap-1">
                  {#if binding.scope === 'ticket'}
                    <TicketIcon class="size-3.5" />
                  {:else}
                    <WorkflowIcon class="size-3.5" />
                  {/if}
                  {binding.target.scope}
                </span>
                <span>Secret {binding.secret.name}</span>
                <span
                  >{binding.secret.scope === 'organization' ? 'Org secret' : 'Project secret'}</span
                >
                {#if binding.secret.disabled}
                  <span class="text-amber-600">Disabled</span>
                {/if}
              </div>
            </div>

            <Button
              variant="ghost"
              size="icon"
              title="Delete binding"
              onclick={() => onDelete(binding.id)}
              disabled={bindingDeleteBusy(binding.id)}
            >
              <Trash2 class="size-4" />
            </Button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
