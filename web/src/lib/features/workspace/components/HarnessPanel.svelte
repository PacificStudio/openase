<script lang="ts">
  import { Save, TriangleAlert } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import { Button } from '$lib/components/ui/button'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
  import HarnessEditor from '$lib/components/harness-editor.svelte'
  import { editorPlaceholder } from '$lib/features/workspace/constants'
  import type { HarnessValidationIssue, Workflow } from '$lib/features/workspace/types'

  let {
    selectedWorkflow = null,
    harnessPath = '',
    harnessVersion = 0,
    harnessDraft = $bindable(''),
    harnessIssues = [],
    validationBusy = false,
    harnessBusy = false,
    harnessDirty = false,
    onDraftChange,
    onValidate,
    onSave,
  }: {
    selectedWorkflow?: Workflow | null
    harnessPath?: string
    harnessVersion?: number
    harnessDraft?: string
    harnessIssues?: HarnessValidationIssue[]
    validationBusy?: boolean
    harnessBusy?: boolean
    harnessDirty?: boolean
    onDraftChange?: (value: string) => void
    onValidate?: () => void
    onSave?: () => void
  } = $props()

  const harnessErrorCount = $derived(
    harnessIssues.filter((issue) => issue.level === 'error').length,
  )
  const harnessWarningCount = $derived(
    harnessIssues.filter((issue) => issue.level !== 'error').length,
  )

  $effect(() => {
    onDraftChange?.(harnessDraft)
  })
</script>

<Card class="border-border/80 bg-background/70">
  <CardHeader>
    <div class="flex items-center justify-between gap-3">
      <div>
        <CardTitle class="flex items-center gap-2">
          <Save class="size-4" />
          <span>Harness</span>
        </CardTitle>
        <CardDescription
          >Validate YAML frontmatter and save through the workflow API.</CardDescription
        >
      </div>
      <div class="flex flex-wrap gap-2">
        {#if selectedWorkflow}
          <Badge variant="outline">v{harnessVersion}</Badge>
        {/if}
        <Badge variant={harnessErrorCount > 0 ? 'destructive' : 'secondary'}>
          {harnessErrorCount > 0 ? `${harnessErrorCount} error` : 'YAML ok'}
        </Badge>
        {#if harnessWarningCount > 0}
          <Badge variant="outline">{harnessWarningCount} warning</Badge>
        {/if}
      </div>
    </div>
  </CardHeader>

  <CardContent class="space-y-4">
    {#if selectedWorkflow}
      <div
        class="border-border/70 bg-muted/25 flex flex-wrap items-center justify-between gap-3 rounded-2xl border px-4 py-3 text-sm"
      >
        <div class="text-muted-foreground flex flex-wrap items-center gap-3">
          <span class="text-foreground font-medium">{selectedWorkflow.name}</span>
          <span class="font-mono text-xs">{harnessPath}</span>
          {#if harnessDirty}
            <Badge variant="outline">unsaved</Badge>
          {/if}
        </div>
        <div class="flex gap-2">
          <Button
            type="button"
            variant="outline"
            disabled={validationBusy}
            onclick={() => void onValidate?.()}
          >
            <TriangleAlert class="mr-2 size-4" />
            Validate
          </Button>
          <Button type="button" disabled={harnessBusy} onclick={() => void onSave?.()}>
            <Save class="mr-2 size-4" />
            Save
          </Button>
        </div>
      </div>

      <HarnessEditor
        bind:value={harnessDraft}
        issues={harnessIssues}
        placeholder={editorPlaceholder}
      />

      {#if validationBusy}
        <div
          class="text-muted-foreground border-border/70 bg-background/60 rounded-2xl border px-4 py-3 text-sm"
        >
          Checking YAML frontmatter…
        </div>
      {/if}
    {:else}
      <div
        class="text-muted-foreground border-border/70 bg-muted/30 rounded-3xl border border-dashed px-4 py-5 text-sm"
      >
        Select a workflow to edit its harness.
      </div>
    {/if}
  </CardContent>
</Card>
