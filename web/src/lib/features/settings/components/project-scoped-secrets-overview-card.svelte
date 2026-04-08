<script lang="ts">
  import { organizationPath } from '$lib/stores/app-context'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'

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
    onCreate: () => void
    name: string
    description: string
    value: string
  } = $props()
</script>

<Card.Root class="rounded-2xl border-slate-200">
  <Card.Header>
    <Card.Title>Scoped secrets</Card.Title>
    <Card.Description>
      Effective runtime secrets inherit org defaults until this project adds an override with the
      same binding key.
    </Card.Description>
  </Card.Header>
  <Card.Content class="space-y-4">
    <div class="grid gap-3 sm:grid-cols-3">
      <div class="rounded-2xl border border-slate-200 bg-slate-50/70 p-4">
        <div class="text-xs font-semibold tracking-[0.18em] text-slate-500 uppercase">
          Effective now
        </div>
        <div class="mt-2 text-3xl font-semibold text-slate-950">{effectiveCount}</div>
        <p class="mt-1 text-sm text-slate-600">What runtime resolution sees after overrides.</p>
      </div>
      <div class="rounded-2xl border border-slate-200 bg-white p-4">
        <div class="text-xs font-semibold tracking-[0.18em] text-slate-500 uppercase">
          Project overrides
        </div>
        <div class="mt-2 text-3xl font-semibold text-slate-950">{projectOverrideCount}</div>
        <p class="mt-1 text-sm text-slate-600">Project-only secrets and org overrides.</p>
      </div>
      <div class="rounded-2xl border border-slate-200 bg-white p-4">
        <div class="text-xs font-semibold tracking-[0.18em] text-slate-500 uppercase">
          Org defaults
        </div>
        <div class="mt-2 text-3xl font-semibold text-slate-950">{organizationSecretCount}</div>
        <p class="mt-1 text-sm text-slate-600">
          Central inventory managed from org admin settings.
        </p>
      </div>
    </div>

    <div class="rounded-2xl border border-slate-200 bg-white p-4">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 class="text-sm font-semibold text-slate-950">Create project override</h3>
          <p class="mt-1 text-sm text-slate-600">
            Use the same name as an inherited org secret to override it for this project only.
          </p>
        </div>
        {#if organizationId}
          <a
            href={`${organizationPath(organizationId)}/admin/settings`}
            class="text-sm font-medium text-slate-700 underline-offset-4 hover:text-slate-950 hover:underline"
          >
            Manage org inventory
          </a>
        {/if}
      </div>

      <div class="mt-4 grid gap-4 lg:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
        <div class="space-y-3">
          <div class="space-y-2">
            <Label for="project-secret-name">Secret name</Label>
            <Input id="project-secret-name" bind:value={name} placeholder="OPENAI_API_KEY" />
          </div>
          <div class="space-y-2">
            <Label for="project-secret-value">Secret value</Label>
            <Input
              id="project-secret-value"
              type="password"
              bind:value
              placeholder="Paste the new secret value"
            />
            <p class="text-xs text-slate-500">
              The raw value is only accepted on write and never shown again.
            </p>
          </div>
        </div>

        <div class="space-y-2">
          <Label for="project-secret-description">Description</Label>
          <Textarea
            id="project-secret-description"
            bind:value={description}
            rows={4}
            placeholder="Optional context for operators and future rotations."
          />
        </div>
      </div>

      <div class="mt-4 flex justify-end">
        <Button onclick={onCreate} disabled={creating}>
          {creating ? 'Creating…' : 'Create override'}
        </Button>
      </div>
    </div>
  </Card.Content>
</Card.Root>
