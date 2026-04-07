<script lang="ts">
  import { FolderLock } from '@lucide/svelte'
</script>

<div class="border-border bg-card space-y-3 rounded-lg border p-4">
  <div class="flex items-center gap-2">
    <FolderLock class="text-muted-foreground size-4" />
    <div class="text-sm font-semibold">Disabled-mode project access</div>
  </div>
  <div class="text-muted-foreground space-y-2 text-sm leading-6">
    <p>
      Disabled mode keeps the local bootstrap principal in charge, so this route focuses on boundary
      clarity and migration guidance instead of pretending there is already a real multi-user
      project directory.
    </p>
    <p>
      When you switch to OIDC, return here to review effective project roles and manage
      project-scoped bindings without reopening instance auth or org membership controls.
    </p>
  </div>
</div>
