<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import type {
    GitHubRepositoryCreateDraft,
    GitHubRepositoryNamespace,
  } from '../repositories-model'

  let {
    namespaces = [],
    draft,
    loadingNamespaces = false,
    creating = false,
    onDraftChange,
    onCreate,
  }: {
    namespaces?: GitHubRepositoryNamespace[]
    draft: GitHubRepositoryCreateDraft
    loadingNamespaces?: boolean
    creating?: boolean
    onDraftChange?: (field: keyof GitHubRepositoryCreateDraft, value: string) => void
    onCreate?: () => void
  } = $props()
</script>

<section class="border-border bg-card/60 space-y-4 rounded-xl border p-4">
  <div class="space-y-1">
    <h3 class="text-foreground text-sm font-semibold">Create on GitHub</h3>
    <p class="text-muted-foreground text-xs">
      Create a repository from OpenASE and bind it to this project immediately.
    </p>
  </div>

  <div class="grid gap-4 md:grid-cols-2">
    <div class="space-y-2">
      <Label>Namespace</Label>
      <Select.Root
        type="single"
        value={draft.owner}
        onValueChange={(value) => onDraftChange?.('owner', value || '')}
      >
        <Select.Trigger class="w-full">
          {draft.owner || (loadingNamespaces ? 'Loading namespaces…' : 'Select namespace')}
        </Select.Trigger>
        <Select.Content>
          {#each namespaces as namespace (namespace.login)}
            <Select.Item value={namespace.login}>
              {namespace.login} · {namespace.kind}
            </Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>

    <div class="space-y-2">
      <Label>Visibility</Label>
      <Select.Root
        type="single"
        value={draft.visibility}
        onValueChange={(value) => onDraftChange?.('visibility', value || 'private')}
      >
        <Select.Trigger class="w-full capitalize">{draft.visibility}</Select.Trigger>
        <Select.Content>
          <Select.Item value="private">private</Select.Item>
          <Select.Item value="public">public</Select.Item>
        </Select.Content>
      </Select.Root>
    </div>
  </div>

  <div class="space-y-2">
    <Label for="github-create-repo-name">Repository name</Label>
    <Input
      id="github-create-repo-name"
      value={draft.name}
      placeholder="backend"
      oninput={(event) => onDraftChange?.('name', (event.currentTarget as HTMLInputElement).value)}
    />
  </div>

  <div class="space-y-2">
    <Label for="github-create-repo-description">Description</Label>
    <Textarea
      id="github-create-repo-description"
      rows={3}
      value={draft.description}
      placeholder="Optional GitHub repository description"
      oninput={(event) =>
        onDraftChange?.('description', (event.currentTarget as HTMLTextAreaElement).value)}
    />
  </div>

  <Button onclick={onCreate} disabled={creating || loadingNamespaces || namespaces.length === 0}>
    {creating ? 'Creating…' : 'Create on GitHub and bind'}
  </Button>
</section>
