<script lang="ts">
  import { Building2, Plus, Save } from '@lucide/svelte'
  import { Button } from '$lib/components/ui/button'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
  import { inputClass } from '$lib/features/workspace/constants'
  import { slugify } from '$lib/features/workspace/mappers'
  import type { Organization, OrganizationForm } from '$lib/features/workspace/types'

  let {
    selectedOrg = null,
    createForm,
    editForm,
    busy = false,
    onCreate,
    onUpdate,
  }: {
    selectedOrg?: Organization | null
    createForm: OrganizationForm
    editForm: OrganizationForm
    busy?: boolean
    onCreate?: () => void
    onUpdate?: () => void
  } = $props()
</script>

<Card class="border-border/80 bg-background/70">
  <CardHeader>
    <CardTitle class="flex items-center gap-2">
      <Building2 class="size-4" />
      <span>Organization</span>
    </CardTitle>
    <CardDescription>Create or update the active workspace boundary.</CardDescription>
  </CardHeader>

  <CardContent class="space-y-5">
    <form
      class="space-y-3"
      onsubmit={(event) => {
        event.preventDefault()
        void onCreate?.()
      }}
    >
      <input
        class={inputClass}
        bind:value={createForm.name}
        placeholder="Acme Control"
        onblur={() => {
          if (!createForm.slug) createForm.slug = slugify(createForm.name)
        }}
      />
      <input class={inputClass} bind:value={createForm.slug} placeholder="acme-control" />
      <Button class="w-full" type="submit" disabled={busy}>
        <Plus class="mr-2 size-4" />
        Create organization
      </Button>
    </form>

    <div class="border-border/70 border-t pt-5">
      {#if selectedOrg}
        <form
          class="space-y-3"
          onsubmit={(event) => {
            event.preventDefault()
            void onUpdate?.()
          }}
        >
          <input
            class={inputClass}
            bind:value={editForm.name}
            onblur={() => {
              if (!editForm.slug) editForm.slug = slugify(editForm.name)
            }}
          />
          <input class={inputClass} bind:value={editForm.slug} />
          <Button class="w-full" type="submit" disabled={busy}>
            <Save class="mr-2 size-4" />
            Save organization
          </Button>
        </form>
      {:else}
        <div
          class="text-muted-foreground border-border/70 bg-muted/30 rounded-3xl border border-dashed px-4 py-5 text-sm"
        >
          Select or create an organization to continue.
        </div>
      {/if}
    </div>
  </CardContent>
</Card>
