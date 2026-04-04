<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import type { TicketRepoOption } from '../types'

  let {
    repos,
    creating = false,
    onCreate,
    onCancel,
  }: {
    repos: TicketRepoOption[]
    creating?: boolean
    onCreate?: (draft: {
      repoId: string
      branchName: string
      pullRequestUrl: string
    }) => Promise<boolean> | boolean
    onCancel?: () => void
  } = $props()

  let createRepoId = $state('')
  let createBranchName = $state('')
  let createPullRequestUrl = $state('')

  $effect(() => {
    if (!repos.length) {
      createRepoId = ''
      createBranchName = ''
      return
    }

    if (!repos.some((repo) => repo.id === createRepoId)) {
      createRepoId = repos[0]?.id ?? ''
      createBranchName = ''
    }
  })

  function handleRepoChange(value: string) {
    createRepoId = value
    createBranchName = ''
  }

  async function handleCreateScope() {
    const accepted =
      (await onCreate?.({
        repoId: createRepoId,
        branchName: createBranchName,
        pullRequestUrl: createPullRequestUrl,
      })) ?? false

    if (accepted) {
      createBranchName = ''
      createPullRequestUrl = ''
    }
  }
</script>

<div class="grid gap-3">
  <div class="space-y-2">
    <Label>Repository</Label>
    <Select.Root
      type="single"
      value={createRepoId}
      onValueChange={(value) => {
        handleRepoChange(value || '')
      }}
    >
      <Select.Trigger class="w-full">
        {repos.find((repo) => repo.id === createRepoId)?.name ?? 'Select repository'}
      </Select.Trigger>
      <Select.Content>
        {#each repos as repo (repo.id)}
          <Select.Item value={repo.id}>{repo.name}</Select.Item>
        {/each}
      </Select.Content>
    </Select.Root>
  </div>

  <div class="space-y-2">
    <Label for="new-scope-branch">Work branch override</Label>
    <Input
      id="new-scope-branch"
      bind:value={createBranchName}
      disabled={!repos.length}
      placeholder="Leave blank to use the generated ticket branch"
    />
    <p class="text-muted-foreground text-xs">
      Base branch: {repos.find((repo) => repo.id === createRepoId)?.defaultBranch || 'main'}
    </p>
  </div>

  <div class="space-y-2">
    <Label for="new-scope-pr-url">Pull request URL</Label>
    <Input id="new-scope-pr-url" bind:value={createPullRequestUrl} placeholder="https://..." />
  </div>
  <div class="flex justify-end gap-2">
    {#if onCancel}
      <Button size="sm" variant="outline" onclick={onCancel} disabled={creating}>Cancel</Button>
    {/if}
    <Button size="sm" onclick={handleCreateScope} disabled={!repos.length || creating}>
      {creating ? 'Adding…' : 'Add repo scope'}
    </Button>
  </div>
</div>
