<script lang="ts">
  import { page } from '$app/state'
  import { ApiError } from '$lib/api/client'
  import { listAgents, listProviders, updateProject } from '$lib/api/openase'
  import {
    getSettingsSectionCapability,
    capabilityStateClasses,
    capabilityStateLabel,
  } from '$lib/features/capabilities'
  import { appStore } from '$lib/stores/app.svelte'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Separator } from '$ui/separator'
  import { Bot, Cpu } from '@lucide/svelte'
  import AgentSettingsBoundaries from './agent-settings-boundaries.svelte'
  import AgentSettingsInventory from './agent-settings-inventory.svelte'
  import AgentSettingsOverview from './agent-settings-overview.svelte'
  import {
    buildGovernanceAgents,
    buildProviderOptions,
    parseDefaultProviderSelection,
    type GovernanceAgent,
    type ProviderOption,
  } from './agent-settings-model'

  const agentsCapability = getSettingsSectionCapability('agents')
  const inheritProviderValue = '__org_default__'
  const agentsConsoleHref = $derived(`/agents${page.url.search}`)

  const adapterIcons: Record<string, typeof Bot> = {
    claude: Bot,
    codex: Cpu,
  }

  let providers = $state<ProviderOption[]>([])
  let agents = $state<GovernanceAgent[]>([])
  let loading = $state(false)
  let loadError = $state('')
  let saving = $state(false)
  let saveError = $state('')
  let feedback = $state('')
  let selectedDefaultProviderId = $state('')

  const selectedDefaultProvider = $derived(
    providers.find((provider) => provider.id === selectedDefaultProviderId) ?? null,
  )
  const orgDefaultProvider = $derived(
    providers.find((provider) => provider.id === appStore.currentOrg?.default_agent_provider_id) ??
      null,
  )
  const runningAgents = $derived(
    agents.filter((agent) => agent.status === 'running' || agent.status === 'claimed').length,
  )

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id

    if (!projectId || !orgId) {
      providers = []
      agents = []
      loading = false
      loadError = ''
      selectedDefaultProviderId = ''
      return
    }

    selectedDefaultProviderId = appStore.currentProject?.default_agent_provider_id ?? ''

    let cancelled = false

    const load = async () => {
      loading = true
      loadError = ''

      try {
        const [providerPayload, agentPayload] = await Promise.all([
          listProviders(orgId),
          listAgents(projectId),
        ])

        if (cancelled) return

        providers = buildProviderOptions(providerPayload.providers, agentPayload.agents)
        agents = buildGovernanceAgents(agentPayload.agents, providerPayload.providers)
      } catch (caughtError) {
        if (cancelled) return
        loadError =
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load agent settings.'
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  async function handleSaveDefaultProvider() {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      saveError = 'Project context is unavailable.'
      return
    }

    const parsed = parseDefaultProviderSelection(selectedDefaultProviderId, providers)
    if (!parsed.ok) {
      saveError = parsed.error
      return
    }

    saving = true
    saveError = ''
    feedback = ''

    try {
      const payload = await updateProject(projectId, {
        default_agent_provider_id: parsed.value,
      })
      appStore.currentProject = payload.project
      feedback = parsed.value
        ? `Default agent provider set to ${selectedDefaultProvider?.name ?? 'the selected provider'}.`
        : 'Project now inherits the organization default provider.'
    } catch (caughtError) {
      saveError =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to save default provider.'
    } finally {
      saving = false
    }
  }
</script>

<div class="space-y-6">
  <div>
    <div class="flex items-center gap-2">
      <h2 class="text-foreground text-base font-semibold">Agents</h2>
      <span
        class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${capabilityStateClasses(agentsCapability.state)}`}
      >
        {capabilityStateLabel(agentsCapability.state)}
      </span>
    </div>
    <p class="text-muted-foreground mt-1 max-w-3xl text-sm">{agentsCapability.summary}</p>
  </div>

  <Separator />

  {#if loading}
    <div class="text-muted-foreground text-sm">Loading agent governance settings…</div>
  {:else if loadError}
    <div class="text-destructive text-sm">{loadError}</div>
  {:else}
    <AgentSettingsOverview
      selectedDefaultProviderName={selectedDefaultProvider?.name ?? null}
      orgDefaultProviderName={orgDefaultProvider?.name ?? null}
      agentCount={agents.length}
      runningAgentCount={runningAgents}
    />

    <div class="grid gap-6 xl:grid-cols-[minmax(0,22rem),minmax(0,1fr)]">
      <div class="space-y-6">
        <Card.Root>
          <Card.Header>
            <Card.Title>Defaults</Card.Title>
            <Card.Description>
              Set project-level routing defaults without duplicating runtime controls.
            </Card.Description>
          </Card.Header>
          <Card.Content class="space-y-4">
            <div class="space-y-2">
              <Label>Default agent provider</Label>
              <Select.Root
                type="single"
                value={selectedDefaultProviderId || inheritProviderValue}
                onValueChange={(value) => {
                  selectedDefaultProviderId = value === inheritProviderValue ? '' : value || ''
                  feedback = ''
                  saveError = ''
                }}
              >
                <Select.Trigger class="w-full">
                  {selectedDefaultProvider?.name ??
                    (selectedDefaultProviderId
                      ? 'Unavailable provider'
                      : 'Use organization default')}
                </Select.Trigger>
                <Select.Content>
                  <Select.Item value={inheritProviderValue}>
                    Use organization default
                    {#if orgDefaultProvider}
                      · {orgDefaultProvider.name}
                    {/if}
                  </Select.Item>
                  {#each providers as provider (provider.id)}
                    <Select.Item value={provider.id}>
                      {provider.name}
                      {#if !provider.available}
                        {' '}· unavailable
                      {/if}
                      {' '}· {provider.adapterType} · {provider.modelName}
                    </Select.Item>
                  {/each}
                </Select.Content>
              </Select.Root>
            </div>

            <div
              class="border-border bg-muted/20 text-muted-foreground rounded-md border px-3 py-3 text-xs"
            >
              Max concurrent agents remains in `Settings / General` and is currently set to
              {` ${appStore.currentProject?.max_concurrent_agents ?? 0}`}.
            </div>

            <div class="space-y-2">
              <div class="text-foreground text-sm font-medium">Providers in scope</div>
              {#if providers.length === 0}
                <div class="text-muted-foreground text-xs">
                  No providers are registered for this organization yet.
                </div>
              {:else}
                <div class="space-y-2">
                  {#each providers as provider (provider.id)}
                    {@const Icon = adapterIcons[provider.adapterType] ?? Bot}
                    <div
                      class="border-border flex items-start justify-between gap-3 rounded-md border px-3 py-2"
                    >
                      <div class="flex items-start gap-2">
                        <div
                          class="bg-muted mt-0.5 flex size-7 items-center justify-center rounded-md"
                        >
                          <Icon class="text-muted-foreground size-3.5" />
                        </div>
                        <div class="min-w-0">
                          <div class="flex items-center gap-2">
                            <span class="text-foreground truncate text-sm font-medium">
                              {provider.name}
                            </span>
                            <Badge
                              variant={provider.available ? 'secondary' : 'outline'}
                              class="text-[10px]"
                            >
                              {provider.available ? 'Available' : 'Unavailable'}
                            </Badge>
                            {#if selectedDefaultProviderId === provider.id}
                              <Badge variant="outline" class="text-[10px]">Project default</Badge>
                            {:else if appStore.currentOrg?.default_agent_provider_id === provider.id}
                              <Badge variant="secondary" class="text-[10px]">Org default</Badge>
                            {/if}
                          </div>
                          <div class="text-muted-foreground text-xs">
                            {provider.adapterType} · {provider.modelName}
                          </div>
                        </div>
                      </div>
                      <div class="text-muted-foreground text-right text-xs">
                        {provider.agentCount} agents
                      </div>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          </Card.Content>
          <Card.Footer class="flex flex-col items-start gap-3">
            <Button onclick={handleSaveDefaultProvider} disabled={saving || providers.length === 0}>
              {saving ? 'Saving…' : 'Save default provider'}
            </Button>
            {#if feedback}
              <div class="text-sm text-emerald-400">{feedback}</div>
            {/if}
            {#if saveError}
              <div class="text-destructive text-sm">{saveError}</div>
            {/if}
          </Card.Footer>
        </Card.Root>

        <AgentSettingsBoundaries />
      </div>

      <AgentSettingsInventory {agents} {agentsConsoleHref} />
    </div>
  {/if}
</div>
