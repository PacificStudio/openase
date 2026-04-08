<script lang="ts">
  import type { ScopedSecret, ScopedSecretBinding, Ticket, Workflow } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { KeyRound } from '@lucide/svelte'

  import SecuritySettingsSecretBindingsList from './security-settings-secret-bindings-list.svelte'

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

  function secretScopeLabel(secret: ScopedSecret) {
    return secret.scope === 'organization' ? 'Org secret' : 'Project secret'
  }

  function targetLabel(target: Workflow | Ticket) {
    if ('identifier' in target) {
      return `${target.identifier} - ${target.title}`
    }
    return target.name
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

      <SecuritySettingsSecretBindingsList {bindings} {error} {mutationKey} {onDelete} />
    </div>
  {/if}
</div>
