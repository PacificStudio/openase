<script lang="ts">
  import type { OrgGitHubCredentialResponse } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'

  type Credential = OrgGitHubCredentialResponse['credential']

  let {
    credential,
    saveDialogOpen,
    deleteDialogOpen,
    tokenDraft,
    anyBusy,
    actionKey,
    onSaveOpenChange,
    onDeleteOpenChange,
    onTokenDraftChange,
    onSave,
    onDelete,
  }: {
    credential: Credential | null
    saveDialogOpen: boolean
    deleteDialogOpen: boolean
    tokenDraft: string
    anyBusy: boolean
    actionKey: string
    onSaveOpenChange: (open: boolean) => void
    onDeleteOpenChange: (open: boolean) => void
    onTokenDraftChange: (value: string) => void
    onSave: () => void
    onDelete: () => void
  } = $props()
</script>

<Dialog.Root open={saveDialogOpen} onOpenChange={onSaveOpenChange}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>
        {credential?.configured ? 'Rotate GitHub token' : 'Save GitHub token'}
      </Dialog.Title>
      <Dialog.Description>
        {#if credential?.configured}
          Paste the replacement token. The previous value is immediately overwritten and cannot be
          recovered.
        {:else}
          Paste a GitHub personal access token (<code>ghu_xxx</code> or
          <code>github_pat_xxx</code>). It will be masked after saving.
        {/if}
      </Dialog.Description>
    </Dialog.Header>

    <div class="space-y-1.5">
      <Label for="credentials-token">Token</Label>
      <Input
        id="credentials-token"
        type="password"
        value={tokenDraft}
        placeholder="ghu_xxx or github_pat_xxx"
        disabled={anyBusy}
        oninput={(event) => onTokenDraftChange(event.currentTarget.value)}
        onkeydown={(event) => {
          if (event.key === 'Enter') onSave()
        }}
      />
    </div>

    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={anyBusy}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button onclick={onSave} disabled={anyBusy || !tokenDraft.trim()}>
        {actionKey === 'save' ? 'Saving…' : credential?.configured ? 'Rotate token' : 'Save token'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root open={deleteDialogOpen} onOpenChange={onDeleteOpenChange}>
  <Dialog.Content class="sm:max-w-sm">
    <Dialog.Header>
      <Dialog.Title>Remove GitHub credential?</Dialog.Title>
      <Dialog.Description>
        All projects that inherit this org credential will lose GitHub access until a new credential
        is saved or each project sets its own override.
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={anyBusy}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button variant="destructive" onclick={onDelete} disabled={anyBusy}>
        {actionKey === 'delete' ? 'Deleting…' : 'Delete credential'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
