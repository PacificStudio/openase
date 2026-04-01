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

<div class="space-y-4">
  <div class="grid gap-4 sm:grid-cols-2">
    <div class="space-y-1.5">
      <Label class="text-xs">Namespace</Label>
      <Select.Root
        type="single"
        value={draft.owner}
        onValueChange={(value) => onDraftChange?.('owner', value || '')}
      >
        <Select.Trigger class="w-full">
          {draft.owner || (loadingNamespaces ? 'Loading…' : 'Select namespace')}
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

    <div class="space-y-1.5">
      <Label class="text-xs">Visibility</Label>
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

  <div class="space-y-1.5">
    <Label for="github-create-repo-name" class="text-xs">Repository name</Label>
    <Input
      id="github-create-repo-name"
      value={draft.name}
      placeholder="backend"
      class="h-9 text-sm"
      oninput={(event) => onDraftChange?.('name', (event.currentTarget as HTMLInputElement).value)}
    />
  </div>

  <div class="space-y-1.5">
    <Label for="github-create-repo-description" class="text-xs">Description</Label>
    <Textarea
      id="github-create-repo-description"
      rows={2}
      value={draft.description}
      placeholder="Optional"
      class="text-sm"
      oninput={(event) =>
        onDraftChange?.('description', (event.currentTarget as HTMLTextAreaElement).value)}
    />
  </div>

  <Button onclick={onCreate} disabled={creating || loadingNamespaces || namespaces.length === 0}>
    {creating ? 'Creating…' : 'Create and bind'}
  </Button>
</div>
