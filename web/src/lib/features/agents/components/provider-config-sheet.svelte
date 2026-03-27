<script lang="ts">
  import type { Machine } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import {
    Sheet,
    SheetContent,
    SheetDescription,
    SheetFooter,
    SheetHeader,
    SheetTitle,
  } from '$ui/sheet'
  import { Textarea } from '$ui/textarea'
  import { providerAdapterOptions } from '../provider-draft'
  import type { ProviderConfig, ProviderDraft, ProviderDraftField } from '../types'

  let {
    open = $bindable(false),
    provider,
    machines,
    draft,
    saving = false,
    onDraftChange,
    onSave,
  }: {
    open?: boolean
    provider: ProviderConfig | null
    machines: Machine[]
    draft: ProviderDraft
    saving?: boolean
    onDraftChange?: (field: ProviderDraftField, value: string) => void
    onSave?: () => void
  } = $props()

  function updateField(field: ProviderDraftField, event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    onDraftChange?.(field, target.value)
  }
</script>

<Sheet bind:open>
  <SheetContent side="right" class="flex w-full flex-col gap-0 p-0 sm:max-w-xl">
    <SheetHeader class="border-border border-b px-6 py-5 text-left">
      <div class="flex items-center gap-2">
        <SheetTitle>{provider?.name ?? 'Provider configuration'}</SheetTitle>
        {#if provider?.isDefault}
          <Badge variant="outline" class="text-[10px]">Default</Badge>
        {/if}
      </div>
      <SheetDescription>
        Update adapter wiring, CLI launch settings, model tuning, and token costs.
      </SheetDescription>
    </SheetHeader>

    {#if provider}
      <div class="flex-1 overflow-y-auto px-6 py-5">
        <div class="space-y-5">
          <div class="grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <Label>Execution machine</Label>
              <Select.Root
                type="single"
                value={draft.machineId}
                onValueChange={(value) => onDraftChange?.('machineId', value || '')}
              >
                <Select.Trigger class="w-full">
                  {machines.find((machine) => machine.id === draft.machineId)?.name ??
                    'Select machine'}
                </Select.Trigger>
                <Select.Content>
                  {#each machines as machine (machine.id)}
                    <Select.Item value={machine.id}>
                      {machine.name} · {machine.status} · {machine.host}
                    </Select.Item>
                  {/each}
                </Select.Content>
              </Select.Root>
            </div>

            <div class="space-y-2">
              <Label for="provider-name">Name</Label>
              <Input
                id="provider-name"
                value={draft.name}
                oninput={(event) => updateField('name', event)}
              />
            </div>

            <div class="space-y-2">
              <Label>Adapter</Label>
              <Select.Root
                type="single"
                value={draft.adapterType}
                onValueChange={(value) => onDraftChange?.('adapterType', value || 'custom')}
              >
                <Select.Trigger class="w-full">
                  {providerAdapterOptions.find((option) => option.value === draft.adapterType)
                    ?.label ?? 'Select adapter'}
                </Select.Trigger>
                <Select.Content>
                  {#each providerAdapterOptions as option (option.value)}
                    <Select.Item value={option.value}>{option.label}</Select.Item>
                  {/each}
                </Select.Content>
              </Select.Root>
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <Label for="provider-cli-command">CLI command</Label>
              <Input
                id="provider-cli-command"
                value={draft.cliCommand}
                placeholder="codex"
                oninput={(event) => updateField('cliCommand', event)}
              />
              <p class="text-muted-foreground text-xs">
                Leave empty to let the backend resolve the adapter default command.
              </p>
            </div>

            <div class="space-y-2">
              <Label for="provider-model-name">Model name</Label>
              <Input
                id="provider-model-name"
                value={draft.modelName}
                oninput={(event) => updateField('modelName', event)}
              />
            </div>
          </div>

          <div class="space-y-2">
            <Label for="provider-cli-args">CLI args</Label>
            <Textarea
              id="provider-cli-args"
              value={draft.cliArgs}
              rows={4}
              placeholder={`app-server\n--listen\nstdio://`}
              oninput={(event) => updateField('cliArgs', event)}
            />
            <p class="text-muted-foreground text-xs">
              Enter one argument per line. Leave blank to clear.
            </p>
          </div>

          <div class="space-y-2">
            <Label for="provider-auth-config">Auth config</Label>
            <Textarea
              id="provider-auth-config"
              value={draft.authConfig}
              rows={8}
              placeholder={`{\n  "token": "secret"\n}`}
              oninput={(event) => updateField('authConfig', event)}
            />
            <p class="text-muted-foreground text-xs">
              Provide a JSON object. Leave blank to clear stored auth config.
            </p>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <Label for="provider-model-temperature">Model temperature</Label>
              <Input
                id="provider-model-temperature"
                type="number"
                min="0"
                step="0.01"
                value={draft.modelTemperature}
                oninput={(event) => updateField('modelTemperature', event)}
              />
            </div>

            <div class="space-y-2">
              <Label for="provider-model-max-tokens">Model max tokens</Label>
              <Input
                id="provider-model-max-tokens"
                type="number"
                min="1"
                step="1"
                value={draft.modelMaxTokens}
                oninput={(event) => updateField('modelMaxTokens', event)}
              />
            </div>

            <div class="space-y-2">
              <Label for="provider-cost-input">Input token cost</Label>
              <Input
                id="provider-cost-input"
                type="number"
                min="0"
                step="0.000001"
                value={draft.costPerInputToken}
                oninput={(event) => updateField('costPerInputToken', event)}
              />
            </div>

            <div class="space-y-2">
              <Label for="provider-cost-output">Output token cost</Label>
              <Input
                id="provider-cost-output"
                type="number"
                min="0"
                step="0.000001"
                value={draft.costPerOutputToken}
                oninput={(event) => updateField('costPerOutputToken', event)}
              />
            </div>
          </div>
        </div>
      </div>

      <SheetFooter class="border-border border-t px-6 py-4">
        <Button variant="outline" onclick={() => (open = false)} disabled={saving}>Cancel</Button>
        <Button onclick={onSave} disabled={saving}>
          {saving ? 'Saving…' : 'Save changes'}
        </Button>
      </SheetFooter>
    {:else}
      <div class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-sm">
        Select a provider to configure.
      </div>
    {/if}
  </SheetContent>
</Sheet>
