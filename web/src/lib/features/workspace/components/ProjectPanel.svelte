<script lang="ts">
  import { Archive, Plus, Rocket, Save } from '@lucide/svelte'
  import { Button } from '$lib/components/ui/button'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
  import { inputClass, projectStatuses, textAreaClass } from '$lib/features/workspace/constants'
  import { slugify } from '$lib/features/workspace/mappers'
  import type { Organization, Project, ProjectForm } from '$lib/features/workspace/types'

  let {
    selectedOrg = null,
    selectedProject = null,
    createForm,
    editForm,
    busy = false,
    onCreate,
    onUpdate,
    onArchive,
  }: {
    selectedOrg?: Organization | null
    selectedProject?: Project | null
    createForm: ProjectForm
    editForm: ProjectForm
    busy?: boolean
    onCreate?: () => void
    onUpdate?: () => void
    onArchive?: () => void
  } = $props()
</script>

<Card class="border-border/80 bg-background/70">
  <CardHeader>
    <CardTitle class="flex items-center gap-2">
      <Rocket class="size-4" />
      <span>Project</span>
    </CardTitle>
    <CardDescription>Manage the selected project and runtime limits.</CardDescription>
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
        placeholder="Control plane"
        disabled={!selectedOrg}
        onblur={() => {
          if (!createForm.slug) createForm.slug = slugify(createForm.name)
        }}
      />
      <input
        class={inputClass}
        bind:value={createForm.slug}
        placeholder="control-plane"
        disabled={!selectedOrg}
      />
      <div class="grid gap-3 sm:grid-cols-2">
        <select class={inputClass} bind:value={createForm.status} disabled={!selectedOrg}>
          {#each projectStatuses as status}
            <option value={status}>{status}</option>
          {/each}
        </select>
        <input
          class={inputClass}
          bind:value={createForm.maxConcurrentAgents}
          min="1"
          type="number"
          disabled={!selectedOrg}
        />
      </div>
      <textarea
        class={textAreaClass}
        bind:value={createForm.description}
        placeholder="What this project owns."
        disabled={!selectedOrg}
      ></textarea>
      <Button class="w-full" type="submit" disabled={!selectedOrg || busy}>
        <Plus class="mr-2 size-4" />
        Create project
      </Button>
    </form>

    <div class="border-border/70 border-t pt-5">
      {#if selectedProject}
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
          <div class="grid gap-3 sm:grid-cols-2">
            <select class={inputClass} bind:value={editForm.status}>
              {#each projectStatuses as status}
                <option value={status}>{status}</option>
              {/each}
            </select>
            <input
              class={inputClass}
              bind:value={editForm.maxConcurrentAgents}
              min="1"
              type="number"
            />
          </div>
          <textarea class={textAreaClass} bind:value={editForm.description}></textarea>
          <div class="flex gap-3">
            <Button class="flex-1" type="submit" disabled={busy}>
              <Save class="mr-2 size-4" />
              Save
            </Button>
            <Button
              class="flex-1"
              type="button"
              variant="outline"
              disabled={busy}
              onclick={() => void onArchive?.()}
            >
              <Archive class="mr-2 size-4" />
              Archive
            </Button>
          </div>
        </form>
      {:else}
        <div
          class="text-muted-foreground border-border/70 bg-muted/30 rounded-3xl border border-dashed px-4 py-5 text-sm"
        >
          Select a project to edit it.
        </div>
      {/if}
    </div>
  </CardContent>
</Card>
