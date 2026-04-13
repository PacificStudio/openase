<script lang="ts">
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import type { ScopeGroup, WorkflowAgentOption } from '../types'
  import type { WorkflowLifecycleDraft } from '../workflow-lifecycle'
  import {
    ensureWorkflowRequiredPlatformScopes,
    REQUIRED_WORKFLOW_PLATFORM_SCOPE,
  } from '../workflow-requirements'
  import WorkflowAgentBindingCard from './workflow-agent-binding-card.svelte'
  import WorkflowAgentSelectOption from './workflow-agent-select-option.svelte'
  import WorkflowAgentSelectTrigger from './workflow-agent-select-trigger.svelte'
  import ScopeGroupPicker from './scope-group-picker.svelte'
  import type { TranslationKey, TranslationParams } from '$lib/i18n/index'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    draft,
    saving = false,
    deleting = false,
    agentOptions = [],
    selectedAgent = null,
    scopeGroups = [],
    onFieldChange,
  }: {
    draft: WorkflowLifecycleDraft
    saving?: boolean
    deleting?: boolean
    agentOptions?: WorkflowAgentOption[]
    selectedAgent?: WorkflowAgentOption | null
    scopeGroups?: ScopeGroup[]
    onFieldChange: (field: keyof WorkflowLifecycleDraft, value: string) => void
  } = $props()

  const selectedScopes = $derived(
    ensureWorkflowRequiredPlatformScopes(
      (draft.platformAccessAllowed ?? '')
        .split('\n')
        .map((s) => s.trim())
        .filter(Boolean),
    ),
  )

  function t(key: TranslationKey, params?: TranslationParams) {
    return i18nStore.t(key, params)
  }

  function handleScopeChange(scopes: string[]) {
    onFieldChange('platformAccessAllowed', ensureWorkflowRequiredPlatformScopes(scopes).join('\n'))
  }
</script>

<div class="space-y-6">
  <div class="grid gap-4 sm:grid-cols-2">
    <div class="space-y-1.5">
      <Label
        for="workflow-name"
        class="text-muted-foreground text-xs font-medium tracking-wide uppercase"
      >
        {t('workflows.detail.identity.labels.workflowName')}
      </Label>
      <Input
        id="workflow-name"
        value={draft.name}
        disabled={saving || deleting}
        oninput={(event) => onFieldChange('name', (event.currentTarget as HTMLInputElement).value)}
      />
    </div>

    <div class="space-y-1.5">
      <Label
        for="workflow-type-label"
        class="text-muted-foreground text-xs font-medium tracking-wide uppercase"
      >
        {t('workflows.detail.identity.labels.typeLabel')}
      </Label>
      <Input
        id="workflow-type-label"
        value={draft.typeLabel}
        disabled={saving || deleting}
        oninput={(event) =>
          onFieldChange('typeLabel', (event.currentTarget as HTMLInputElement).value)}
      />
    </div>
  </div>

  <div class="grid gap-4 sm:grid-cols-2">
    <div class="space-y-1.5">
      <Label
        for="workflow-role-name"
        class="text-muted-foreground text-xs font-medium tracking-wide uppercase"
      >
        {t('workflows.detail.identity.labels.roleName')}
      </Label>
      <Input
        id="workflow-role-name"
        value={draft.roleName}
        disabled={saving || deleting}
        oninput={(event) =>
          onFieldChange('roleName', (event.currentTarget as HTMLInputElement).value)}
      />
    </div>
  </div>

  <div class="space-y-1.5">
    <Label
      for="workflow-role-description"
      class="text-muted-foreground text-xs font-medium tracking-wide uppercase"
    >
      {t('workflows.detail.identity.labels.roleDescription')}
    </Label>
    <textarea
      id="workflow-role-description"
      class="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring min-h-24 w-full rounded-md border px-3 py-2 text-sm focus-visible:ring-2 focus-visible:outline-none"
      value={draft.roleDescription}
      disabled={saving || deleting}
      oninput={(event) =>
        onFieldChange('roleDescription', (event.currentTarget as HTMLTextAreaElement).value)}
    ></textarea>
  </div>

  <div class="space-y-1.5">
    <Label class="text-muted-foreground text-xs font-medium tracking-wide uppercase">
      {t('workflows.detail.identity.labels.platformAccessAllowed')}
    </Label>
    <div class="bg-muted/40 rounded-md border px-3 py-2 text-xs leading-relaxed">
      <span class="font-medium">
        {t('workflows.detail.identity.platformAccess.systemRequired')}
      </span>
      {t('workflows.detail.identity.platformAccess.description.prefix')}
      <code class="bg-background rounded px-1 py-0.5 font-mono"
        >{REQUIRED_WORKFLOW_PLATFORM_SCOPE}</code
      >
      {t('workflows.detail.identity.platformAccess.description.suffix')}
    </div>
    {#if scopeGroups.length > 0}
      <ScopeGroupPicker
        groups={scopeGroups}
        selected={selectedScopes}
        lockedScopes={[REQUIRED_WORKFLOW_PLATFORM_SCOPE]}
        disabled={saving || deleting}
        onchange={handleScopeChange}
      />
    {:else}
      <textarea
        id="workflow-platform-access"
        class="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring min-h-24 w-full rounded-md border px-3 py-2 font-mono text-sm focus-visible:ring-2 focus-visible:outline-none"
        value={draft.platformAccessAllowed}
        disabled={saving || deleting}
        placeholder={t('workflows.detail.identity.placeholders.platformScope')}
        oninput={(event) =>
          onFieldChange(
            'platformAccessAllowed',
            (event.currentTarget as HTMLTextAreaElement).value,
          )}
      ></textarea>
      <p class="text-muted-foreground text-xs">
        <code class="bg-background rounded px-1 py-0.5 font-mono"
          >{REQUIRED_WORKFLOW_PLATFORM_SCOPE}</code
        >
        {t('workflows.detail.identity.platformAccess.note')}
      </p>
    {/if}
  </div>

  <div class="space-y-1.5">
    <Label class="text-muted-foreground text-xs font-medium tracking-wide uppercase">
      {t('workflows.detail.identity.labels.boundAgent')}
    </Label>
    <Select.Root
      type="single"
      value={draft.agentId}
      disabled={saving || deleting || agentOptions.length === 0}
      onValueChange={(value) => onFieldChange('agentId', value || '')}
    >
      <Select.Trigger class="h-auto w-full py-2">
        <WorkflowAgentSelectTrigger {selectedAgent} />
      </Select.Trigger>
      <Select.Content>
        {#each agentOptions as option (option.id)}
          <Select.Item value={option.id}>
            <WorkflowAgentSelectOption {option} />
          </Select.Item>
        {/each}
      </Select.Content>
    </Select.Root>
  </div>

  <WorkflowAgentBindingCard {selectedAgent} />
</div>
