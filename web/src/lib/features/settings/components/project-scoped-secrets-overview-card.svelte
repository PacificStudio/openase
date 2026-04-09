<script lang="ts">
  import { organizationPath } from '$lib/stores/app-context'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import { KeyRound, Plus, X } from '@lucide/svelte'

  let {
    effectiveCount,
    projectOverrideCount,
    organizationSecretCount,
    organizationId,
    creating,
    onCreate,
    name = $bindable(),
    description = $bindable(),
    value = $bindable(),
  }: {
    effectiveCount: number
    projectOverrideCount: number
    organizationSecretCount: number
    organizationId: string
    creating: boolean
    onCreate: () => Promise<boolean>
    name: string
    description: string
    value: string
  } = $props()

  let formOpen = $state(false)

  // Auto-open when a pre-fill arrives from "Use as override draft"
  $effect(() => {
    if (name && !formOpen) formOpen = true
  })

  async function handleCreate() {
    const success = await onCreate()
    if (success) formOpen = false
  }

  function close() {
    formOpen = false
  }
</script>

<div class="space-y-4">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div class="flex items-center gap-2">
      <KeyRound class="text-muted-foreground size-4" />
      <h3 class="text-sm font-semibold">Scoped secrets</h3>
    </div>
    <div class="flex items-center gap-3">
      {#if organizationId}
        <a
          href={`${organizationPath(organizationId)}/admin/settings`}
          class="text-muted-foreground hover:text-foreground text-sm transition-colors"
        >
          Manage org inventory →
        </a>
      {/if}
      <Button size="sm" variant="outline" onclick={() => (formOpen = !formOpen)}>
        {#if formOpen}
          <X class="size-3.5" />
          Cancel
        {:else}
          <Plus class="size-3.5" />
          New override
        {/if}
      </Button>
    </div>
  </div>

  <div
    class="bg-muted/40 flex flex-wrap items-center gap-x-4 gap-y-1 rounded-lg px-4 py-2.5 text-sm"
  >
    <span>
      <span class="font-semibold">{effectiveCount}</span>
      <span class="text-muted-foreground"> effective</span>
    </span>
    <span class="text-muted-foreground">·</span>
    <span>
      <span class="font-semibold">{projectOverrideCount}</span>
      <span class="text-muted-foreground"> project overrides</span>
    </span>
    <span class="text-muted-foreground">·</span>
    <span>
      <span class="font-semibold">{organizationSecretCount}</span>
      <span class="text-muted-foreground"> org defaults</span>
    </span>
  </div>

  {#if formOpen}
    <div class="border-border rounded-lg border p-4">
      <div class="mb-4">
        <h4 class="text-sm font-medium">Create project override</h4>
        <p class="text-muted-foreground mt-0.5 text-xs">
          Use the same name as an inherited org secret to override it for this project only.
        </p>
      </div>

      <div class="grid gap-4 lg:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
        <div class="space-y-3">
          <div class="space-y-1.5">
            <Label for="project-secret-name">Secret name</Label>
            <Input id="project-secret-name" bind:value={name} placeholder="OPENAI_API_KEY" />
          </div>
          <div class="space-y-1.5">
            <Label for="project-secret-value">Secret value</Label>
            <Input
              id="project-secret-value"
              type="password"
              bind:value
              placeholder="Paste the new secret value"
            />
            <p class="text-muted-foreground text-xs">
              The raw value is only accepted on write and never shown again.
            </p>
          </div>
        </div>

        <div class="space-y-1.5">
          <Label for="project-secret-description">Description</Label>
          <Textarea
            id="project-secret-description"
            bind:value={description}
            rows={4}
            placeholder="Optional context for operators and future rotations."
          />
        </div>
      </div>

      <div class="mt-4 flex items-center justify-end gap-3">
        <Button variant="ghost" size="sm" onclick={close} disabled={creating}>Cancel</Button>
        <Button size="sm" onclick={() => void handleCreate()} disabled={creating}>
          {creating ? 'Creating…' : 'Create override'}
        </Button>
      </div>
    </div>
  {/if}
</div>
