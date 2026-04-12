<script lang="ts">
  import type { ScopedSecret, ScopedSecretBinding, Ticket, Workflow } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import type { TranslationKey } from '$lib/i18n'
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

  const scopeOptions: { value: SecretBindingDraft['scope']; labelKey: TranslationKey }[] = [
    { value: 'workflow', labelKey: 'settings.secretBindings.scope.workflow' },
    { value: 'ticket', labelKey: 'settings.secretBindings.scope.ticket' },
  ]

  function scopeLabelKey(scope: SecretBindingDraft['scope']): TranslationKey {
    return scope === 'ticket'
      ? 'settings.secretBindings.scope.ticket'
      : 'settings.secretBindings.scope.workflow'
  }

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

  function secretScopeLabel(secret: Pick<ScopedSecret, 'scope'>) {
    return secret.scope === 'organization'
      ? 'settings.secretBindings.scopeLabel.organization'
      : 'settings.secretBindings.scopeLabel.project'
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
        <h3 class="text-sm font-semibold">
          {i18nStore.t('settings.secretBindings.heading')}
        </h3>
      </div>

    <Dialog.Root bind:open={dialogOpen}>
      <Dialog.Trigger>
        {#snippet child({ props })}
          <Button size="sm" {...props}>
            <Plus class="size-4" />
            <span class="hidden sm:inline">
              {i18nStore.t('settings.secretBindings.buttons.bindSecret')}
            </span>
          </Button>
        {/snippet}
      </Dialog.Trigger>
      <Dialog.Content class="sm:max-w-md">
        <Dialog.Header>
          <Dialog.Title>
            {i18nStore.t('settings.secretBindings.dialogs.bindTitle')}
          </Dialog.Title>
          <Dialog.Description>
            {i18nStore.t('settings.secretBindings.dialogs.bindDescription')}
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
                <Label>{i18nStore.t('settings.secretBindings.labels.scope')}</Label>
                <Select.Root
                  type="single"
                  value={draft.scope}
                  onValueChange={(value) => handleScopeChange(value || 'workflow')}
                >
                  <Select.Trigger class="w-full">
                    {@const scopeOption = scopeOptions.find((o) => o.value === draft.scope)}
                    {scopeOption
                      ? i18nStore.t(scopeOption.labelKey)
                      : draft.scope}
                  </Select.Trigger>
                  <Select.Content>
                    {#each scopeOptions as option (option.value)}
                      <Select.Item value={option.value}>{i18nStore.t(option.labelKey)}</Select.Item>
                    {/each}
                  </Select.Content>
                </Select.Root>
              </div>

              <div class="space-y-2">
                <Label>
                  {i18nStore.t(
                    draft.scope === 'workflow'
                      ? 'settings.secretBindings.scope.workflow'
                      : 'settings.secretBindings.scope.ticket',
                  )}
                </Label>
                <Select.Root
                  type="single"
                  value={draft.scopeResourceId}
                  onValueChange={(value) => patchDraft({ scopeResourceId: value || '' })}
                >
                  <Select.Trigger class="w-full">
                    {#if draft.scopeResourceId}
                      {@const target = availableTargets.find((t) => t.id === draft.scopeResourceId)}
                      {target
                        ? targetLabel(target)
                        : i18nStore.t('settings.secretBindings.placeholders.select')}
                    {:else}
                      {i18nStore.t(
                        draft.scope === 'workflow'
                          ? 'settings.secretBindings.placeholders.selectWorkflowTarget'
                          : 'settings.secretBindings.placeholders.selectTicketTarget',
                      )}
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
                    {i18nStore.t(
                      draft.scope === 'workflow'
                        ? 'settings.secretBindings.messages.noWorkflows'
                        : 'settings.secretBindings.messages.noTickets',
                    )}
                  </p>
                {/if}
              </div>
            </div>

            <div class="space-y-2">
              <Label>{i18nStore.t('settings.secretBindings.labels.secret')}</Label>
              <Select.Root
                type="single"
                value={draft.secretId}
                onValueChange={(value) => patchDraft({ secretId: value || '' })}
              >
                <Select.Trigger class="w-full">
                  {#if draft.secretId}
                    {@const secret = sortedSecrets.find((s) => s.id === draft.secretId)}
                    {secret
                      ? `${secret.name} (${i18nStore.t(secretScopeLabel(secret))})`
                      : i18nStore.t('settings.secretBindings.placeholders.select')}
                  {:else}
                    {i18nStore.t('settings.secretBindings.placeholders.selectSecret')}
                  {/if}
                </Select.Trigger>
                <Select.Content>
                  {#each sortedSecrets as secret (secret.id)}
                    <Select.Item value={secret.id}>
                      {secret.name}
                      <span class="text-muted-foreground ml-1 text-xs">
                        {i18nStore.t(secretScopeLabel(secret))}
                        {secret.disabled
                          ? ` · ${i18nStore.t('settings.secretBindings.status.disabled')}`
                          : ''}
                      </span>
                    </Select.Item>
                  {/each}
                </Select.Content>
              </Select.Root>
              {#if sortedSecrets.length === 0}
                <p class="text-muted-foreground text-xs">
                  {i18nStore.t('settings.secretBindings.messages.createSecretFirst')}
                </p>
              {/if}
            </div>

            <div class="space-y-2">
              <Label for="binding-key">
                {i18nStore.t('settings.secretBindings.labels.bindingKey')}
              </Label>
              <Input
                id="binding-key"
                placeholder={i18nStore.t('settings.secretBindings.placeholders.bindingKey')}
                value={draft.bindingKey}
                oninput={(event) =>
                  patchDraft({ bindingKey: (event.currentTarget as HTMLInputElement).value })}
              />
              <p class="text-muted-foreground text-xs">
                {i18nStore.t('settings.secretBindings.hints.bindingKey')}
              </p>
            </div>
          </Dialog.Body>

          <Dialog.Footer>
            <Dialog.Close>
              {#snippet child({ props })}
                <Button variant="outline" {...props}>
                  {i18nStore.t('settings.secretBindings.buttons.cancel')}
                </Button>
              {/snippet}
            </Dialog.Close>
            <Button type="submit" disabled={!canCreate || mutationKey === 'create'}>
              <Link class="size-4" />
              {mutationKey === 'create'
                ? i18nStore.t('settings.secretBindings.buttons.binding')
                : i18nStore.t('settings.secretBindings.buttons.bindSecret')}
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
      <p>{i18nStore.t('settings.secretBindings.emptyState.title')}</p>
      <p class="text-xs">{i18nStore.t('settings.secretBindings.emptyState.hint')}</p>
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
                {i18nStore.t(scopeLabelKey(binding.scope as 'workflow' | 'ticket'))}
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
              ({i18nStore.t(secretScopeLabel(binding.secret))})
            </span>
            {#if binding.secret.disabled}
              <span class="text-amber-600">
                {i18nStore.t('settings.secretBindings.status.disabled')}
              </span>
            {/if}
          </div>
        </div>

          <Button
            variant="ghost"
            size="icon-sm"
            class="text-muted-foreground hover:text-destructive shrink-0 self-end sm:self-auto"
            title={i18nStore.t('settings.secretBindings.buttons.deleteBinding')}
            onclick={() => onDelete(binding.id)}
            disabled={mutationKey === `delete:${binding.id}`}
          >
            <Trash2 class="size-4" />
            <span class="sr-only">
              {i18nStore.t('settings.secretBindings.buttons.deleteBinding')}
            </span>
          </Button>
        </div>
      {/each}
    </div>
  {/if}
</div>
