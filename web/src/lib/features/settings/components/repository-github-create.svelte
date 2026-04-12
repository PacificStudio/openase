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
  import { i18nStore } from '$lib/i18n/store.svelte'

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
      <Label class="text-xs">{i18nStore.t('settings.repositoryGitHubCreate.labels.namespace')}</Label>
      <Select.Root
        type="single"
        value={draft.owner}
        onValueChange={(value) => onDraftChange?.('owner', value || '')}
      >
        <Select.Trigger class="w-full">
          {draft.owner ||
            (loadingNamespaces
              ? i18nStore.t('settings.repositoryGitHubCreate.messages.loading')
              : i18nStore.t('settings.repositoryGitHubCreate.placeholder.selectNamespace'))}
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
      <Label class="text-xs">{i18nStore.t('settings.repositoryGitHubCreate.labels.visibility')}</Label>
      <Select.Root
        type="single"
        value={draft.visibility}
        onValueChange={(value) => onDraftChange?.('visibility', value || 'private')}
      >
        <Select.Trigger class="w-full capitalize">
          {i18nStore.t(
            `settings.repositoryGitHubCreate.visibility.${draft.visibility || 'private'}`,
          )}
        </Select.Trigger>
        <Select.Content>
          <Select.Item value="private">
            {i18nStore.t('settings.repositoryGitHubCreate.visibility.private')}
          </Select.Item>
          <Select.Item value="public">
            {i18nStore.t('settings.repositoryGitHubCreate.visibility.public')}
          </Select.Item>
        </Select.Content>
      </Select.Root>
    </div>
  </div>

  <div class="space-y-1.5">
    <Label for="github-create-repo-name" class="text-xs">
      {i18nStore.t('settings.repositoryGitHubCreate.labels.repositoryName')}
    </Label>
    <Input
      id="github-create-repo-name"
      value={draft.name}
      placeholder={i18nStore.t('settings.repositoryGitHubCreate.placeholders.name')}
      class="h-9 text-sm"
      oninput={(event) => onDraftChange?.('name', (event.currentTarget as HTMLInputElement).value)}
    />
  </div>

  <div class="space-y-1.5">
    <Label for="github-create-repo-description" class="text-xs">
      {i18nStore.t('settings.repositoryGitHubCreate.labels.description')}
    </Label>
    <Textarea
      id="github-create-repo-description"
      rows={2}
      value={draft.description}
      placeholder={i18nStore.t('settings.repositoryGitHubCreate.placeholders.description')}
      class="text-sm"
      oninput={(event) =>
        onDraftChange?.('description', (event.currentTarget as HTMLTextAreaElement).value)}
    />
  </div>

  <Button onclick={onCreate} disabled={creating || loadingNamespaces || namespaces.length === 0}>
    {creating
      ? i18nStore.t('settings.repositoryGitHubCreate.actions.creating')
      : i18nStore.t('settings.repositoryGitHubCreate.actions.createAndBind')}
  </Button>
</div>
