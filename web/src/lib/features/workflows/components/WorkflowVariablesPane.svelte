<script lang="ts">
  import { Badge } from '$lib/components/ui/badge'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'

  const variables = [
    { token: '{{ ticket.identifier }}', description: 'Stable human ticket key.' },
    { token: '{{ ticket.title }}', description: 'Ticket title injected into the harness.' },
    { token: '{{ project.name }}', description: 'Selected project display name.' },
    { token: '{{ agent.name }}', description: 'Assigned agent name at execution time.' },
    { token: '{{ attempt }}', description: 'Current retry / execution attempt number.' },
  ]
</script>

<Card class="border-border/80 bg-background/80 backdrop-blur">
  <CardHeader>
    <CardTitle>Variables dictionary</CardTitle>
    <CardDescription>
      Common harness tokens stay documented next to the editor so future features can extend the
      same pattern.
    </CardDescription>
  </CardHeader>

  <CardContent class="space-y-3">
    {#each variables as variable}
      <div class="border-border/70 bg-background/60 rounded-2xl border px-4 py-3">
        <Badge variant="outline">{variable.token}</Badge>
        <p class="text-muted-foreground mt-2 text-sm">{variable.description}</p>
      </div>
    {/each}
  </CardContent>
</Card>
