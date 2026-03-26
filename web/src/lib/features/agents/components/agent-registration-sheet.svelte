<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import {
    Sheet,
    SheetContent,
    SheetDescription,
    SheetFooter,
    SheetHeader,
    SheetTitle,
  } from '$ui/sheet'
  import type { AgentProvider } from '$lib/api/contracts'
  import type { AgentRegistrationDraft, AgentRegistrationDraftField } from '../registration'

  let {
    open = $bindable(false),
    providers,
    draft,
    saving = false,
    error = '',
    feedback = '',
    onDraftChange,
    onSubmit,
    onOpenChange,
  }: {
    open?: boolean
    providers: AgentProvider[]
    draft: AgentRegistrationDraft
    saving?: boolean
    error?: string
    feedback?: string
    onDraftChange?: (field: AgentRegistrationDraftField, value: string) => void
    onSubmit?: () => void
    onOpenChange?: (open: boolean) => void
  } = $props()

  $effect(() => {
    onOpenChange?.(open)
  })

  function updateField(field: AgentRegistrationDraftField, event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    onDraftChange?.(field, target.value)
  }

  function handleSubmit(event: SubmitEvent) {
    event.preventDefault()
    onSubmit?.()
  }

  function providerLabel(provider: AgentProvider) {
    return provider.available
      ? `${provider.name} · ${provider.adapter_type} · ${provider.model_name}`
      : `${provider.name} · unavailable · ${provider.adapter_type} · ${provider.model_name}`
  }

  function selectedProviderLabel() {
    const provider = providers.find((item) => item.id === draft.providerId)
    return provider ? providerLabel(provider) : 'Select provider'
  }
</script>

<Sheet bind:open>
  <SheetContent side="right" class="w-full sm:max-w-xl">
    <SheetHeader class="space-y-1 px-6 py-6">
      <SheetTitle>Register agent</SheetTitle>
      <SheetDescription>
        Create a runnable agent instance for the current project using an existing provider.
      </SheetDescription>
    </SheetHeader>

    <form class="flex h-full flex-col gap-5 px-6 pb-6" onsubmit={handleSubmit}>
      {#if error}
        <div
          class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
        >
          {error}
        </div>
      {/if}

      {#if feedback}
        <div
          class="rounded-md border border-emerald-500/40 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-700 dark:text-emerald-300"
        >
          {feedback}
        </div>
      {/if}

      {#if providers.length === 0}
        <div
          class="border-border bg-muted/40 text-muted-foreground rounded-md border px-4 py-3 text-sm"
        >
          Register an agent provider first. Agent registration needs at least one provider.
        </div>
      {/if}

      <div class="grid gap-4">
        <div class="space-y-2">
          <Label>Provider</Label>
          <Select.Root
            type="single"
            value={draft.providerId}
            onValueChange={(value) => onDraftChange?.('providerId', value || '')}
          >
            <Select.Trigger class="w-full">{selectedProviderLabel()}</Select.Trigger>
            <Select.Content>
              {#each providers as provider (provider.id)}
                <Select.Item value={provider.id}>{providerLabel(provider)}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>

        <div class="space-y-2">
          <Label for="agent-name">Name</Label>
          <Input
            id="agent-name"
            value={draft.name}
            placeholder="coding-01"
            oninput={(event) => updateField('name', event)}
          />
        </div>

        <div class="space-y-2">
          <Label for="agent-workspace-path">Workspace path</Label>
          <Input
            id="agent-workspace-path"
            value={draft.workspacePath}
            placeholder="/srv/openase/workspaces/coding-01"
            oninput={(event) => updateField('workspacePath', event)}
          />
        </div>
      </div>

      <SheetFooter class="mt-auto gap-2 px-0 pb-0 sm:justify-end">
        <Button
          type="button"
          variant="outline"
          onclick={() => onOpenChange?.(false)}
          disabled={saving}
        >
          Cancel
        </Button>
        <Button type="submit" disabled={saving || providers.length === 0}>
          {saving ? 'Registering…' : 'Register agent'}
        </Button>
      </SheetFooter>
    </form>
  </SheetContent>
</Sheet>
