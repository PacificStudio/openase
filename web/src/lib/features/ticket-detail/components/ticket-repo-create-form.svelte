<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { repoScopeCiStatusOptions, repoScopePrStatusOptions } from '../mutation-shared'
  import type { TicketRepoOption } from '../types'

  let {
    repos,
    creating = false,
    onCreate,
  }: {
    repos: TicketRepoOption[]
    creating?: boolean
    onCreate?: (draft: {
      repoId: string
      branchName: string
      pullRequestUrl: string
      prStatus: string
      ciStatus: string
      isPrimaryScope: boolean
    }) => void
  } = $props()

  let createRepoId = $state('')
  let createBranchName = $state('')
  let createPullRequestUrl = $state('')
  let createPrStatus = $state('')
  let createCiStatus = $state('')
  let createIsPrimaryScope = $state(false)

  $effect(() => {
    if (!repos.length) {
      createRepoId = ''
      createBranchName = ''
      return
    }

    if (!repos.some((repo) => repo.id === createRepoId)) {
      createRepoId = repos[0]?.id ?? ''
      createBranchName = repos[0]?.defaultBranch ?? ''
    }
  })

  function handleRepoChange(value: string) {
    createRepoId = value
    createBranchName = repos.find((repo) => repo.id === value)?.defaultBranch ?? ''
  }

  function handleCreateScope() {
    onCreate?.({
      repoId: createRepoId,
      branchName: createBranchName,
      pullRequestUrl: createPullRequestUrl,
      prStatus: createPrStatus,
      ciStatus: createCiStatus,
      isPrimaryScope: createIsPrimaryScope,
    })
  }
</script>

<section class="border-border bg-muted/20 rounded-lg border p-4">
  <div>
    <h3 class="text-sm font-medium">Add Repo Scope</h3>
    <p class="text-muted-foreground mt-1 text-xs">
      Link a repository branch and its PR/CI lifecycle to this ticket.
    </p>
  </div>

  <div class="mt-4 grid gap-3">
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
      <Label for="new-scope-branch">Branch</Label>
      <Input id="new-scope-branch" bind:value={createBranchName} disabled={!repos.length} />
    </div>

    <div class="space-y-2">
      <Label for="new-scope-pr-url">Pull request URL</Label>
      <Input id="new-scope-pr-url" bind:value={createPullRequestUrl} placeholder="https://..." />
    </div>

    <div class="grid gap-3 sm:grid-cols-2">
      <div class="space-y-2">
        <Label>PR status</Label>
        <Select.Root
          type="single"
          value={createPrStatus}
          onValueChange={(value) => {
            createPrStatus = value || ''
          }}
        >
          <Select.Trigger class="w-full">
            {repoScopePrStatusOptions.find((option) => option.value === createPrStatus)?.label ??
              'Unset'}
          </Select.Trigger>
          <Select.Content>
            {#each repoScopePrStatusOptions as option (option.value)}
              <Select.Item value={option.value}>{option.label}</Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
      </div>

      <div class="space-y-2">
        <Label>CI status</Label>
        <Select.Root
          type="single"
          value={createCiStatus}
          onValueChange={(value) => {
            createCiStatus = value || ''
          }}
        >
          <Select.Trigger class="w-full">
            {repoScopeCiStatusOptions.find((option) => option.value === createCiStatus)?.label ??
              'Unset'}
          </Select.Trigger>
          <Select.Content>
            {#each repoScopeCiStatusOptions as option (option.value)}
              <Select.Item value={option.value}>{option.label}</Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
      </div>
    </div>

    <label class="flex items-center gap-2 text-xs">
      <input type="checkbox" bind:checked={createIsPrimaryScope} />
      <span>Mark as primary scope</span>
    </label>

    <div class="flex justify-end">
      <Button size="sm" onclick={handleCreateScope} disabled={!repos.length || creating}>
        {creating ? 'Adding…' : 'Add repo scope'}
      </Button>
    </div>
  </div>
</section>
