<script lang="ts">
  import { Button } from '$ui/button'
  import * as Tabs from '$ui/tabs'
  import { Plus } from '@lucide/svelte'
  import AgentList from './agent-list.svelte'
  import ProviderList from './provider-list.svelte'
  import type { AgentInstance, ProviderConfig } from '../types'

  const now = new Date()

  const agents: AgentInstance[] = [
    {
      id: 'agent-1',
      name: 'claude-alpha',
      providerName: 'Anthropic',
      modelName: 'claude-sonnet-4',
      status: 'running',
      runtimePhase: 'ready',
      currentTicket: {
        id: 'tkt-101',
        identifier: 'ENG-342',
        title: 'Refactor auth middleware to support OAuth2 PKCE flow',
      },
      lastHeartbeat: new Date(now.getTime() - 15_000).toISOString(),
      runtimeStartedAt: new Date(now.getTime() - 180_000).toISOString(),
      sessionId: 'thread-claude-alpha',
      todayCompleted: 7,
      todayCost: 4.32,
      capabilities: ['code-generation', 'code-review'],
    },
    {
      id: 'agent-2',
      name: 'claude-beta',
      providerName: 'Anthropic',
      modelName: 'claude-sonnet-4',
      status: 'idle',
      runtimePhase: 'none',
      lastHeartbeat: null,
      todayCompleted: 12,
      todayCost: 6.18,
      capabilities: ['code-generation', 'testing'],
    },
    {
      id: 'agent-3',
      name: 'codex-primary',
      providerName: 'OpenAI',
      modelName: 'codex-1',
      status: 'claimed',
      runtimePhase: 'launching',
      currentTicket: {
        id: 'tkt-205',
        identifier: 'ENG-587',
        title: 'Add pagination to list endpoints',
      },
      lastHeartbeat: null,
      todayCompleted: 3,
      todayCost: 2.75,
      capabilities: ['code-generation'],
    },
    {
      id: 'agent-4',
      name: 'claude-gamma',
      providerName: 'Anthropic',
      modelName: 'claude-opus-4',
      status: 'failed',
      runtimePhase: 'failed',
      currentTicket: {
        id: 'tkt-189',
        identifier: 'ENG-401',
        title: 'Migrate database schema to use UUIDs',
      },
      lastHeartbeat: new Date(now.getTime() - 600_000).toISOString(),
      lastError: 'Codex launch handshake timed out',
      todayCompleted: 1,
      todayCost: 8.4,
      capabilities: ['code-generation', 'code-review', 'architecture'],
    },
    {
      id: 'agent-5',
      name: 'codex-secondary',
      providerName: 'OpenAI',
      modelName: 'codex-1',
      status: 'terminated',
      runtimePhase: 'none',
      lastHeartbeat: new Date(now.getTime() - 7_200_000).toISOString(),
      todayCompleted: 0,
      todayCost: 0,
      capabilities: ['code-generation'],
    },
  ]

  const providers: ProviderConfig[] = [
    {
      id: 'prov-1',
      name: 'Anthropic Cloud',
      adapterType: 'claude',
      modelName: 'claude-sonnet-4',
      agentCount: 3,
      isDefault: true,
    },
    {
      id: 'prov-2',
      name: 'OpenAI Codex',
      adapterType: 'codex',
      modelName: 'codex-1',
      agentCount: 2,
      isDefault: false,
    },
    {
      id: 'prov-3',
      name: 'Anthropic Premium',
      adapterType: 'claude',
      modelName: 'claude-opus-4',
      agentCount: 1,
      isDefault: false,
    },
  ]

  let activeTab = $state('instances')
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <h1 class="text-foreground text-lg font-semibold">Agents</h1>
    <Button size="sm">
      <Plus class="size-3.5" />
      Register Agent
    </Button>
  </div>

  <Tabs.Root bind:value={activeTab}>
    <Tabs.List variant="line">
      <Tabs.Trigger value="instances">Instances</Tabs.Trigger>
      <Tabs.Trigger value="providers">Providers</Tabs.Trigger>
    </Tabs.List>
    <Tabs.Content value="instances" class="pt-3">
      <AgentList {agents} />
    </Tabs.Content>
    <Tabs.Content value="providers" class="pt-3">
      <ProviderList {providers} />
    </Tabs.Content>
  </Tabs.Root>
</div>
