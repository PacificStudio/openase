<script lang="ts">
  import type { Machine } from '$lib/api/contracts'
  import { providerAdapterOptions } from '$lib/features/agents/public'
  import type { ProviderDraft } from '$lib/features/agents/public'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'

  let {
    draft,
    machines = [],
    creating = false,
    onFieldChange,
    onAdapterChange,
    onSubmit,
  }: {
    draft: ProviderDraft
    machines?: Machine[]
    creating?: boolean
    onFieldChange?: (field: keyof ProviderDraft, value: string) => void
    onAdapterChange?: (value: string) => void
    onSubmit?: () => void
  } = $props()
</script>

<Card.Root class="rounded-2xl">
  <Card.Header>
    <Card.Title>Create provider</Card.Title>
    <Card.Description>
      Add a model adapter before registering agents or selecting project defaults.
    </Card.Description>
  </Card.Header>

  <Card.Content>
    <form
      class="space-y-4"
      onsubmit={(event) => {
        event.preventDefault()
        onSubmit?.()
      }}
    >
      <div class="space-y-2">
        <Label for="provider-name">Provider name</Label>
        <Input
          id="provider-name"
          value={draft.name}
          placeholder="Codex primary"
          oninput={(event) =>
            onFieldChange?.('name', (event.currentTarget as HTMLInputElement).value)}
        />
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <div class="space-y-2">
          <Label>Execution machine</Label>
          <Select.Root
            type="single"
            value={draft.machineId}
            onValueChange={(value) => onFieldChange?.('machineId', value || '')}
          >
            <Select.Trigger class="w-full">
              {machines.find((machine) => machine.id === draft.machineId)?.name ?? 'Select machine'}
            </Select.Trigger>
            <Select.Content>
              {#each machines as machine (machine.id)}
                <Select.Item value={machine.id}
                  >{machine.name} · {machine.status} · {machine.host}</Select.Item
                >
              {/each}
            </Select.Content>
          </Select.Root>
        </div>

        <div class="space-y-2">
          <Label>Adapter</Label>
          <Select.Root
            type="single"
            value={draft.adapterType}
            onValueChange={(value) => onAdapterChange?.(value || 'custom')}
          >
            <Select.Trigger class="w-full">
              {providerAdapterOptions.find((option) => option.value === draft.adapterType)?.label ??
                'Select adapter'}
            </Select.Trigger>
            <Select.Content>
              {#each providerAdapterOptions as option (option.value)}
                <Select.Item value={option.value}>{option.label}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>

        <div class="space-y-2">
          <Label for="provider-model">Model name</Label>
          <Input
            id="provider-model"
            value={draft.modelName}
            placeholder="gpt-5.4"
            oninput={(event) =>
              onFieldChange?.('modelName', (event.currentTarget as HTMLInputElement).value)}
          />
        </div>
      </div>

      <div class="space-y-2">
        <Label for="provider-cli-command">CLI command</Label>
        <Input
          id="provider-cli-command"
          value={draft.cliCommand}
          placeholder="codex"
          oninput={(event) =>
            onFieldChange?.('cliCommand', (event.currentTarget as HTMLInputElement).value)}
        />
      </div>

      <div class="space-y-2">
        <Label for="provider-cli-args">CLI args</Label>
        <Textarea
          id="provider-cli-args"
          rows={3}
          value={draft.cliArgs}
          placeholder={`app-server\n--listen\nstdio://`}
          oninput={(event) =>
            onFieldChange?.('cliArgs', (event.currentTarget as HTMLTextAreaElement).value)}
        />
      </div>

      <div class="space-y-2">
        <Label for="provider-auth-config">Auth config</Label>
        <Textarea
          id="provider-auth-config"
          rows={5}
          value={draft.authConfig}
          placeholder={`{\n  "token": "secret"\n}`}
          oninput={(event) =>
            onFieldChange?.('authConfig', (event.currentTarget as HTMLTextAreaElement).value)}
        />
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <div class="space-y-2">
          <Label for="provider-model-temperature">Temperature</Label>
          <Input
            id="provider-model-temperature"
            type="number"
            min="0"
            step="0.01"
            value={draft.modelTemperature}
            oninput={(event) =>
              onFieldChange?.('modelTemperature', (event.currentTarget as HTMLInputElement).value)}
          />
        </div>

        <div class="space-y-2">
          <Label for="provider-model-max-tokens">Max tokens</Label>
          <Input
            id="provider-model-max-tokens"
            type="number"
            min="1"
            step="1"
            value={draft.modelMaxTokens}
            oninput={(event) =>
              onFieldChange?.('modelMaxTokens', (event.currentTarget as HTMLInputElement).value)}
          />
        </div>
      </div>

      <Button type="submit" class="w-full" disabled={creating}>
        {creating ? 'Creating…' : 'Create provider'}
      </Button>
    </form>
  </Card.Content>
</Card.Root>
