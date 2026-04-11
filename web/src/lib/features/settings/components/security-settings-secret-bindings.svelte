<script lang="ts">
  import type { ScopedSecret, ScopedSecretBinding, Ticket, Workflow } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import {
    KeyRound,
    Link,
    Plus,
    Ticket as TicketIcon,
    Trash2,
    Workflow as WorkflowIcon,
  } from '@lucide/svelte'

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

  let dialogOpen = $state(false)

  const scopeOptions = [
    { value: 'workflow' as const, label: 'Workflow' },
    { value: 'ticket' as const, label: 'Ticket' },
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

  const sortedBindings = $derived(
    [...bindings].sort((a, b) => {
      if (a.scope !== b.scope) return a.scope.localeCompare(b.scope)
      if (a.target.name !== b.target.name) return a.target.name.localeCompare(b.target.name)
      return a.binding_key.localeCompare(b.binding_key)
    }),
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
    return secret.scope === 'organization' ? 'Org' : 'Project'
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

  function handleCreate() {
    onCreate()
    dialogOpen = false
  }
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between gap-3">
    <div class="flex items-center gap-2">
      <KeyRound class="text-muted-foreground size-4" />
      <h3 class="text-sm font-semibold">Runtime secret bindings</h3>
    </div>

    <Dialog.Root bind:open={dialogOpen}>
      <Dialog.Trigger>
        {#snippet child({ props })}
          <Button size="sm" {...props}>
            <Plus class="size-4" />
            <span class="hidden sm:inline">Bind secret</span>
          </Button>
        {/snippet}
      </Dialog.Trigger>
      <Dialog.Content class="sm:max-w-md">
        <Dialog.Header>
          <Dialog.Title>Bind secret to runtime</Dialog.Title>
          <Dialog.Description>
            Attach a shared secret to a workflow or ticket. Values are never exposed here.
          </Dialog.Description>
        </Dialog.Header>

        <form
          class="flex min-h-0 flex-1 flex-col gap-6"
          onsubmit={(event) => {
            event.preventDefault()
            handleCreate()
          }}
        >
          <Dialog.Body class="space-y-4">
            <div class="grid gap-4 sm:grid-cols-2">
              <div class="space-y-2">
                <Label>Scope</Label>
                <Select.Root
                  type="single"
                  value={draft.scope}
                  onValueChange={(value) => handleScopeChange(value || 'workflow')}
                >
                  <Select.Trigger class="w-full">
                    {scopeOptions.find((o) => o.value === draft.scope)?.label ?? draft.scope}
                  </Select.Trigger>
                  <Select.Content>
                    {#each scopeOptions as option (option.value)}
                      <Select.Item value={option.value}>{option.label}</Select.Item>
                    {/each}
                  </Select.Content>
                </Select.Root>
              </div>

              <div class="space-y-2">
                <Label>{draft.scope === 'workflow' ? 'Workflow' : 'Ticket'}</Label>
                <Select.Root
                  type="single"
                  value={draft.scopeResourceId}
                  onValueChange={(value) => patchDraft({ scopeResourceId: value || '' })}
                >
                  <Select.Trigger class="w-full">
                    {#if draft.scopeResourceId}
                      {@const target = availableTargets.find((t) => t.id === draft.scopeResourceId)}
                      {target ? targetLabel(target) : 'Select...'}
                    {:else}
                      Select {draft.scope}...
                    {/if}
                  </Select.Trigger>
                  <Select.Content>
                    {#each availableTargets as target (target.id)}
                      <Select.Item value={target.id}>{targetLabel(target)}</Select.Item>
                    {/each}
                  </Select.Content>
                </Select.Root>
                {#if availableTargets.length === 0}
                  <p class="text-muted-foreground text-xs">
                    No {draft.scope === 'workflow' ? 'workflows' : 'tickets'} in this project.
                  </p>
                {/if}
              </div>
            </div>

            <div class="space-y-2">
              <Label>Secret</Label>
              <Select.Root
                type="single"
                value={draft.secretId}
                onValueChange={(value) => patchDraft({ secretId: value || '' })}
              >
                <Select.Trigger class="w-full">
                  {#if draft.secretId}
                    {@const secret = sortedSecrets.find((s) => s.id === draft.secretId)}
                    {secret ? `${secret.name} (${secretScopeLabel(secret)})` : 'Select...'}
                  {:else}
                    Select secret...
                  {/if}
                </Select.Trigger>
                <Select.Content>
                  {#each sortedSecrets as secret (secret.id)}
                    <Select.Item value={secret.id}>
                      {secret.name}
                      <span class="text-muted-foreground ml-1 text-xs">
                        {secretScopeLabel(secret)}{secret.disabled ? ' · disabled' : ''}
                      </span>
                    </Select.Item>
                  {/each}
                </Select.Content>
              </Select.Root>
              {#if sortedSecrets.length === 0}
                <p class="text-muted-foreground text-xs">
                  Create a scoped secret first to enable bindings.
                </p>
              {/if}
            </div>

            <div class="space-y-2">
              <Label for="binding-key">Binding key</Label>
              <Input
                id="binding-key"
                placeholder="OPENAI_API_KEY"
                value={draft.bindingKey}
                oninput={(event) =>
                  patchDraft({ bindingKey: (event.currentTarget as HTMLInputElement).value })}
              />
              <p class="text-muted-foreground text-xs">
                Upper-snake env var name. Ticket bindings take precedence over workflow bindings.
              </p>
            </div>
          </Dialog.Body>

          <Dialog.Footer>
            <Dialog.Close>
              {#snippet child({ props })}
                <Button variant="outline" {...props}>Cancel</Button>
              {/snippet}
            </Dialog.Close>
            <Button type="submit" disabled={!canCreate || mutationKey === 'create'}>
              <Link class="size-4" />
              {mutationKey === 'create' ? 'Binding...' : 'Bind secret'}
            </Button>
          </Dialog.Footer>
        </form>
      </Dialog.Content>
    </Dialog.Root>
  </div>

  {#if loading}
    <div class="bg-muted h-32 animate-pulse rounded-lg"></div>
  {:else if error}
    <div class="text-destructive rounded-lg border px-3 py-2 text-sm">{error}</div>
  {:else if sortedBindings.length === 0}
    <div
      class="text-muted-foreground flex flex-col items-center gap-2 rounded-lg border border-dashed px-4 py-8 text-center text-sm"
    >
      <KeyRound class="text-muted-foreground/50 size-8" />
      <p>No secret bindings yet.</p>
      <p class="text-xs">Bind a shared secret to a workflow or ticket to use it at runtime.</p>
    </div>
  {:else}
    <div class="divide-border border-border overflow-hidden rounded-lg border">
      {#each sortedBindings as binding, index (binding.id)}
        <div
          class="flex flex-col gap-3 px-4 py-3 text-sm sm:flex-row sm:items-center sm:justify-between {index >
          0
            ? 'border-border border-t'
            : ''}"
        >
          <div class="min-w-0 space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <Badge variant={binding.scope === 'ticket' ? 'default' : 'secondary'}>
                {#if binding.scope === 'ticket'}
                  <TicketIcon class="mr-1 size-3" />
                {:else}
                  <WorkflowIcon class="mr-1 size-3" />
                {/if}
                {binding.scope}
              </Badge>
              <code class="bg-muted truncate rounded px-1.5 py-0.5 text-xs">
                {binding.binding_key}
              </code>
            </div>
            <div class="text-muted-foreground flex flex-wrap items-center gap-x-2 text-xs">
              <span class="font-medium">{bindingTargetLabel(binding)}</span>
              <span>·</span>
              <span>{binding.secret.name}</span>
              <span class="text-muted-foreground/70">
                ({binding.secret.scope === 'organization' ? 'org' : 'project'})
              </span>
              {#if binding.secret.disabled}
                <span class="text-amber-600">disabled</span>
              {/if}
            </div>
          </div>

          <Button
            variant="ghost"
            size="icon-sm"
            class="text-muted-foreground hover:text-destructive shrink-0 self-end sm:self-auto"
            title="Delete binding"
            onclick={() => onDelete(binding.id)}
            disabled={mutationKey === `delete:${binding.id}`}
          >
            <Trash2 class="size-4" />
            <span class="sr-only">Delete binding</span>
          </Button>
        </div>
      {/each}
    </div>
  {/if}
</div>
