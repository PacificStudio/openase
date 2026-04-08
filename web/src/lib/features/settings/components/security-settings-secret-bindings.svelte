<script lang="ts">
  import type { ScopedSecret, ScopedSecretBinding, Ticket, Workflow } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { KeyRound, Ticket as TicketIcon, Trash2, Workflow as WorkflowIcon } from '@lucide/svelte'

  export type SecretBindingDraft = {
    bindingKey: string
    scope: 'workflow' | 'ticket'
    scopeResourceId: string
    secretId: string
  }

  let {
    secrets,
    bindings,
    workflows,
    tickets,
    draft,
    loading = false,
    error = '',
    mutationKey = '',
    onDraftChange,
    onCreate,
    onDelete,
  }: {
    secrets: ScopedSecret[]
    bindings: ScopedSecretBinding[]
    workflows: Workflow[]
    tickets: Ticket[]
    draft: SecretBindingDraft
    loading?: boolean
    error?: string
    mutationKey?: string
    onDraftChange: (draft: SecretBindingDraft) => void
    onCreate: () => void
    onDelete: (bindingId: string) => void
  } = $props()

  const scopeOptions = [
    { value: 'workflow' as const, label: 'Workflow override' },
    { value: 'ticket' as const, label: 'Ticket override' },
  ]

  const sortedSecrets = $derived(
    [...secrets].sort((a, b) => {
      if (a.scope !== b.scope) {
        return a.scope.localeCompare(b.scope)
      }
      return a.name.localeCompare(b.name)
    }),
  )

  const sortedWorkflows = $derived([...workflows].sort((a, b) => a.name.localeCompare(b.name)))
  const sortedTickets = $derived(
    [...tickets].sort((a, b) => a.identifier.localeCompare(b.identifier)),
  )
  const availableTargets = $derived(draft.scope === 'workflow' ? sortedWorkflows : sortedTickets)
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
  const canCreate = $derived(
    draft.bindingKey.trim().length > 0 &&
      draft.secretId.length > 0 &&
      draft.scopeResourceId.length > 0,
  )

  function patchDraft(patch: Partial<SecretBindingDraft>) {
    onDraftChange({ ...draft, ...patch })
  }

  function handleScopeChange(value: string) {
    const nextScope = value === 'ticket' ? 'ticket' : 'workflow'
    onDraftChange({
      bindingKey: draft.bindingKey,
      scope: nextScope,
      scopeResourceId: '',
      secretId: draft.secretId,
    })
  }

  function scopeBadgeVariant(scope: ScopedSecretBinding['scope']) {
    return scope === 'ticket' ? 'default' : 'secondary'
  }

  function secretScopeLabel(secret: ScopedSecret) {
    return secret.scope === 'organization' ? 'Org secret' : 'Project secret'
  }

  function targetLabel(target: Workflow | Ticket) {
    if ('identifier' in target) {
      return `${target.identifier} - ${target.title}`
    }
    return target.name
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

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <KeyRound class="text-muted-foreground size-4" />
    <h3 class="text-sm font-semibold">Runtime secret bindings</h3>
  </div>

  <div class="bg-muted/40 grid gap-3 rounded-xl px-4 py-3 text-sm lg:grid-cols-3">
    <div>
      <div class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Secrets in scope</div>
      <div class="mt-1 font-semibold">{secrets.length}</div>
      <div class="text-muted-foreground text-xs">
        Shared store entries available to this project
      </div>
    </div>
    <div>
      <div class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Workflow bindings</div>
      <div class="mt-1 font-semibold">
        {bindings.filter((binding) => binding.scope === 'workflow').length}
      </div>
      <div class="text-muted-foreground text-xs">Reusable runtime context for workflows</div>
    </div>
    <div>
      <div class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Ticket bindings</div>
      <div class="mt-1 font-semibold">
        {bindings.filter((binding) => binding.scope === 'ticket').length}
      </div>
      <div class="text-muted-foreground text-xs">One-off overrides that win at runtime</div>
    </div>
  </div>

  {#if loading}
    <div class="grid gap-4 xl:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
      <div class="bg-muted h-80 animate-pulse rounded-2xl"></div>
      <div class="bg-muted h-80 animate-pulse rounded-2xl"></div>
    </div>
  {:else}
    <div class="grid gap-4 xl:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
      <div class="border-border bg-card space-y-4 rounded-2xl border p-5">
        <div>
          <div class="text-sm font-semibold">Create binding</div>
          <p class="text-muted-foreground mt-1 text-xs leading-5">
            Bind an existing shared secret to a workflow for reusable runs or to a ticket for a
            one-off execution override. Secret values never appear here.
          </p>
        </div>

        <div class="grid gap-4 sm:grid-cols-2">
          <div class="space-y-1.5">
            <Label for="binding-scope">Binding scope</Label>
            <select
              id="binding-scope"
              class="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none"
              value={draft.scope}
              onchange={(event) =>
                handleScopeChange((event.currentTarget as HTMLSelectElement).value)}
            >
              {#each scopeOptions as option (option.value)}
                <option value={option.value}>{option.label}</option>
              {/each}
            </select>
          </div>

          <div class="space-y-1.5">
            <Label for="binding-target">
              {draft.scope === 'workflow' ? 'Workflow target' : 'Ticket target'}
            </Label>
            <select
              id="binding-target"
              class="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none"
              value={draft.scopeResourceId}
              onchange={(event) =>
                patchDraft({ scopeResourceId: (event.currentTarget as HTMLSelectElement).value })}
            >
              <option value="" disabled>
                {draft.scope === 'workflow' ? 'Select workflow' : 'Select ticket'}
              </option>
              {#each availableTargets as target (target.id)}
                <option value={target.id}>{targetLabel(target)}</option>
              {/each}
            </select>
            {#if availableTargets.length === 0}
              <p class="text-muted-foreground text-xs">
                {draft.scope === 'workflow'
                  ? 'No workflows available in this project.'
                  : 'No tickets available in this project.'}
              </p>
            {/if}
          </div>
        </div>

        <div class="space-y-1.5">
          <Label for="binding-secret">Secret</Label>
          <select
            id="binding-secret"
            class="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none"
            value={draft.secretId}
            onchange={(event) =>
              patchDraft({ secretId: (event.currentTarget as HTMLSelectElement).value })}
          >
            <option value="" disabled>Select secret</option>
            {#each sortedSecrets as secret (secret.id)}
              <option value={secret.id}>
                {secret.name} - {secretScopeLabel(secret)}{secret.disabled ? ' - disabled' : ''}
              </option>
            {/each}
          </select>
          {#if sortedSecrets.length === 0}
            <p class="text-muted-foreground text-xs">
              Create a scoped secret first to enable bindings.
            </p>
          {/if}
        </div>

        <div class="space-y-1.5">
          <Label for="binding-key">Binding key</Label>
          <Input
            id="binding-key"
            placeholder="OPENAI_API_KEY"
            value={draft.bindingKey}
            oninput={(event) =>
              patchDraft({ bindingKey: (event.currentTarget as HTMLInputElement).value })}
          />
          <p class="text-muted-foreground text-xs">
            Runtime lookup uses the normalized upper-snake binding key. Ticket bindings outrank
            workflow, agent, project, and organization bindings.
          </p>
        </div>

        <div class="flex items-center gap-2">
          <Button size="sm" onclick={onCreate} disabled={!canCreate || mutationKey === 'create'}>
            {mutationKey === 'create' ? 'Creating...' : 'Create binding'}
          </Button>
          <span class="text-muted-foreground text-xs">No secret value is exposed in this flow.</span
          >
        </div>
      </div>

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
            No workflow or ticket bindings yet. Shared secrets stay unattached until you bind them
            to a runtime target.
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
                        >{binding.secret.scope === 'organization'
                          ? 'Org secret'
                          : 'Project secret'}</span
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
    </div>
  {/if}
</div>
