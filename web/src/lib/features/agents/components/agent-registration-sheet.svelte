<script lang="ts">
  import { Button } from '$ui/button'
  import { adapterIconPath, availabilityDotColor } from '$lib/features/providers'
  import { cn } from '$lib/utils'
  import { Input } from '$ui/input'
  import { Wrench } from '@lucide/svelte'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { deriveWorkspaceConvention } from '../registration'
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
    currentOrgSlug,
    currentProjectSlug,
    saving = false,
    onDraftChange,
    onSubmit,
    onOpenChange,
  }: {
    open?: boolean
    providers: AgentProvider[]
    draft: AgentRegistrationDraft
    currentOrgSlug?: string
    currentProjectSlug?: string
    saving?: boolean
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

  const selectedProvider = $derived(providers.find((item) => item.id === draft.providerId) ?? null)

  const workspaceConvention = $derived(
    deriveWorkspaceConvention(
      providers.find((item) => item.id === draft.providerId),
      currentOrgSlug,
      currentProjectSlug,
    ),
  )
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
            <Select.Trigger class="h-auto w-full py-2">
              {#if selectedProvider}
                {@const iconPath = adapterIconPath(selectedProvider.adapter_type)}
                <div class="flex items-center gap-2.5">
                  {#if iconPath}
                    <img src={iconPath} alt="" class="size-5 shrink-0" />
                  {:else}
                    <Wrench class="text-muted-foreground size-5 shrink-0" />
                  {/if}
                  <div class="min-w-0 text-left">
                    <div class="text-foreground truncate text-sm font-medium">
                      {selectedProvider.name}
                    </div>
                    <div class="text-muted-foreground truncate text-xs">
                      {selectedProvider.machine_name} &middot; {selectedProvider.model_name}
                    </div>
                  </div>
                  <span
                    class={cn(
                      'ml-auto size-2 shrink-0 rounded-full',
                      availabilityDotColor(selectedProvider.available),
                    )}
                  ></span>
                </div>
              {:else}
                <span class="text-muted-foreground">Select provider</span>
              {/if}
            </Select.Trigger>
            <Select.Content>
              {#each providers as provider (provider.id)}
                {@const iconPath = adapterIconPath(provider.adapter_type)}
                <Select.Item value={provider.id}>
                  <div class="flex items-center gap-2.5 py-0.5">
                    {#if iconPath}
                      <img src={iconPath} alt="" class="size-4 shrink-0" />
                    {:else}
                      <Wrench class="text-muted-foreground size-4 shrink-0" />
                    {/if}
                    <span class="truncate">{provider.name}</span>
                    <span
                      class={cn(
                        'size-1.5 shrink-0 rounded-full',
                        availabilityDotColor(provider.available),
                      )}
                    ></span>
                    <span class="text-muted-foreground ml-auto truncate text-xs"
                      >{provider.machine_name}</span
                    >
                  </div>
                </Select.Item>
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
          <Label>Workspace convention</Label>
          <div class="border-border text-muted-foreground rounded-md border px-3 py-2 text-sm">
            <div class="font-mono break-all">{workspaceConvention}</div>
            <div class="mt-2 text-xs">
              Ticket workspaces are derived by OpenASE from org, project, and ticket identity.
            </div>
          </div>
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
