<script lang="ts">
  import { ScrollArea } from '$ui/scroll-area'
  import type {
    ProjectConversationWorkspaceFilePatch,
    ProjectConversationWorkspaceFilePreview,
    ProjectConversationWorkspaceRepoMetadata,
  } from '$lib/api/chat'

  let {
    selectedRepo = null,
    selectedFilePath = '',
    currentTreePath = '',
    preview = null,
    patch = null,
    fileLoading = false,
    fileError = '',
  }: {
    selectedRepo?: ProjectConversationWorkspaceRepoMetadata | null
    selectedFilePath?: string
    currentTreePath?: string
    preview?: ProjectConversationWorkspaceFilePreview | null
    patch?: ProjectConversationWorkspaceFilePatch | null
    fileLoading?: boolean
    fileError?: string
  } = $props()
</script>

<div class="min-h-0">
  <ScrollArea class="h-full">
    <div class="space-y-4 p-4">
      {#if !selectedRepo}
        <p class="text-muted-foreground text-sm">Select a repo to browse its files.</p>
      {:else}
        <div class="space-y-1">
          <p class="text-xs font-medium tracking-wide uppercase">Selection</p>
          <p class="font-mono text-sm">
            {selectedFilePath || currentTreePath || selectedRepo.path}
          </p>
        </div>

        {#if fileError}
          <div class="border-destructive/20 bg-destructive/5 rounded-lg border p-3">
            <p class="text-destructive text-sm">{fileError}</p>
          </div>
        {:else if selectedFilePath}
          <section class="space-y-3">
            <div class="border-border bg-muted/20 rounded-lg border">
              <div class="border-border flex items-center justify-between border-b px-3 py-2">
                <div>
                  <p class="text-sm font-medium">Preview</p>
                  {#if preview}
                    <p class="text-muted-foreground text-[11px]">
                      {preview.mediaType} · {preview.sizeBytes} B
                    </p>
                  {/if}
                </div>
                {#if fileLoading}
                  <span class="text-muted-foreground text-[11px]">Loading…</span>
                {/if}
              </div>

              {#if preview?.previewKind === 'binary'}
                <p class="text-muted-foreground px-3 py-4 text-sm">
                  Binary files are not rendered inline in the read-only browser.
                </p>
              {:else if preview}
                <pre
                  class="overflow-x-auto p-3 font-mono text-[12px] leading-5 whitespace-pre-wrap">{preview.content}</pre>
              {:else if fileLoading}
                <p class="text-muted-foreground px-3 py-4 text-sm">Loading preview…</p>
              {:else}
                <p class="text-muted-foreground px-3 py-4 text-sm">
                  Choose a file to load its preview.
                </p>
              {/if}
            </div>

            <div class="border-border bg-muted/20 rounded-lg border">
              <div class="border-border flex items-center justify-between border-b px-3 py-2">
                <div>
                  <p class="text-sm font-medium">Diff</p>
                  {#if patch}
                    <p class="text-muted-foreground text-[11px]">
                      {patch.status} · {patch.diffKind}
                    </p>
                  {/if}
                </div>
              </div>

              {#if patch?.diffKind === 'text'}
                <pre
                  class="overflow-x-auto p-3 font-mono text-[12px] leading-5 whitespace-pre-wrap">{patch.diff}</pre>
              {:else if patch?.diffKind === 'binary'}
                <p class="text-muted-foreground px-3 py-4 text-sm">
                  Binary changes are detected, but the diff body is not rendered inline.
                </p>
              {:else if patch?.diffKind === 'none'}
                <p class="text-muted-foreground px-3 py-4 text-sm">
                  No diff against `HEAD` for this file.
                </p>
              {:else}
                <p class="text-muted-foreground px-3 py-4 text-sm">
                  Choose a file to load its git diff.
                </p>
              {/if}
            </div>
          </section>
        {:else}
          <div
            class="text-muted-foreground rounded-lg border border-dashed px-4 py-8 text-center text-sm"
          >
            Select a file from the tree or git status list to inspect it.
          </div>
        {/if}
      {/if}
    </div>
  </ScrollArea>
</div>
