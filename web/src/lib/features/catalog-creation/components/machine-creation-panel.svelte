<script lang="ts">
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'

  let {
    projectCount,
    providerCount,
    href = null,
  }: {
    projectCount: number
    providerCount: number
    href?: string | null
  } = $props()
</script>

<Card.Root class="rounded-2xl">
  <Card.Header>
    <Card.Title>Machine creation</Card.Title>
    <Card.Description>
      Machines already use the live org-scoped API. Keep project creation and provider setup here,
      then move into the machine console when you need capacity.
    </Card.Description>
  </Card.Header>

  <Card.Content class="space-y-3">
    <div class="grid grid-cols-2 gap-3 text-sm">
      <div class="border-border bg-background rounded-xl border px-3 py-3">
        <div class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Projects</div>
        <div class="text-foreground mt-2 text-xl font-semibold">{projectCount}</div>
      </div>
      <div class="border-border bg-background rounded-xl border px-3 py-3">
        <div class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Providers</div>
        <div class="text-foreground mt-2 text-xl font-semibold">{providerCount}</div>
      </div>
    </div>

    <p class="text-muted-foreground text-sm leading-6">
      Machine registration lives on the Machines page and applies across the organization, so new
      projects can reuse the same worker pool.
    </p>

    <Button href={href ?? undefined} disabled={!href} class="w-full">
      {href ? 'Open machine console' : 'Create a project first'}
    </Button>
  </Card.Content>
</Card.Root>
