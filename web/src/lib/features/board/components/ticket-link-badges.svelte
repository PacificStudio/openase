<script lang="ts">
  import type { BoardExternalLink } from '../types'
  import { GitPullRequest, Link2, Github } from '@lucide/svelte'
  import * as Tooltip from '$ui/tooltip'
  import * as DropdownMenu from '$ui/dropdown-menu'

  let {
    links = [],
    pullRequestURLs = [],
  }: {
    links?: BoardExternalLink[]
    pullRequestURLs?: string[]
  } = $props()

  function parseGitHubPR(url: string) {
    const m = url.match(/github\.com\/([^/]+\/[^/]+)\/pull\/(\d+)/)
    return m ? { repo: m[1], number: m[2] } : null
  }

  function parseGitHubIssue(url: string) {
    const m = url.match(/github\.com\/([^/]+\/[^/]+)\/issues\/(\d+)/)
    return m ? { repo: m[1], number: m[2] } : null
  }

  function isGitHub(url: string) {
    return /github\.com/.test(url)
  }

  // Repo-scope PRs shown as first-class badges
  const scopePRs = $derived(
    pullRequestURLs
      .map((url) => ({ url, parsed: parseGitHubPR(url) }))
      .filter(
        (x): x is { url: string; parsed: NonNullable<ReturnType<typeof parseGitHubPR>> } =>
          x.parsed !== null,
      ),
  )

  // External links that are GitHub PRs
  const linkPRs = $derived(links.filter((l) => parseGitHubPR(l.url) !== null))
  // All PR objects (scope PRs + link PRs), deduped by URL
  const allPRs = $derived.by(() => {
    const seen = new Set<string>()
    const result: Array<{ url: string; number: string; repo: string }> = []
    for (const { url, parsed } of scopePRs) {
      if (!seen.has(url)) {
        seen.add(url)
        result.push({ url, ...parsed })
      }
    }
    for (const link of linkPRs) {
      const parsed = parseGitHubPR(link.url)!
      if (!seen.has(link.url)) {
        seen.add(link.url)
        result.push({ url: link.url, ...parsed })
      }
    }
    return result
  })

  // Non-PR external links
  const otherLinks = $derived(links.filter((l) => parseGitHubPR(l.url) === null))

  const hasAnything = $derived(allPRs.length > 0 || otherLinks.length > 0)

  function openExternal(url: string) {
    window.open(url, '_blank', 'noopener,noreferrer')
  }

  const triggerClass =
    'bg-transparent text-muted-foreground hover:text-foreground hover:bg-muted inline-flex cursor-pointer items-center gap-0.5 rounded px-1 py-0.5 text-[10px] transition-colors focus-visible:outline-none'
</script>

{#if hasAnything}
  <span
    class="inline-flex items-center gap-0.5"
    onclick={(e) => e.stopPropagation()}
    role="presentation"
  >
    <!-- ── Pull requests ───────────────────────────────────── -->
    {#if allPRs.length === 1}
      {@const pr = allPRs[0]}
      <Tooltip.Provider>
        <Tooltip.Root>
          <Tooltip.Trigger class={triggerClass} onclick={() => openExternal(pr.url)}>
            <GitPullRequest class="size-3" />
            #{pr.number}
          </Tooltip.Trigger>
          <Tooltip.Content side="top">
            PR #{pr.number} · {pr.repo}
          </Tooltip.Content>
        </Tooltip.Root>
      </Tooltip.Provider>
    {:else if allPRs.length > 1}
      <DropdownMenu.Root>
        <DropdownMenu.Trigger class={triggerClass}>
          <GitPullRequest class="size-3" />
          {allPRs.length}
        </DropdownMenu.Trigger>
        <DropdownMenu.Content align="start" class="min-w-48">
          {#each allPRs as pr}
            <DropdownMenu.Item class="gap-2 text-xs" onclick={() => openExternal(pr.url)}>
              <Github class="size-3 shrink-0" />
              <span class="font-mono">#{pr.number}</span>
              <span class="text-muted-foreground ml-auto max-w-36 truncate">{pr.repo}</span>
            </DropdownMenu.Item>
          {/each}
        </DropdownMenu.Content>
      </DropdownMenu.Root>
    {/if}

    <!-- ── Other external links ───────────────────────────── -->
    {#if otherLinks.length === 1}
      {@const link = otherLinks[0]}
      {@const ghIssue = parseGitHubIssue(link.url)}
      <Tooltip.Provider>
        <Tooltip.Root>
          <Tooltip.Trigger class={triggerClass} onclick={() => openExternal(link.url)}>
            {#if ghIssue}
              <Github class="size-3" />
              #{ghIssue.number}
            {:else}
              <Link2 class="size-3" />
            {/if}
          </Tooltip.Trigger>
          <Tooltip.Content side="top">
            {#if ghIssue}
              Issue #{ghIssue.number} · {ghIssue.repo}
            {:else}
              {link.title || link.externalId || link.url}
            {/if}
          </Tooltip.Content>
        </Tooltip.Root>
      </Tooltip.Provider>
    {:else if otherLinks.length > 1}
      <DropdownMenu.Root>
        <DropdownMenu.Trigger class={triggerClass}>
          <Link2 class="size-3" />
          {otherLinks.length}
        </DropdownMenu.Trigger>
        <DropdownMenu.Content align="start" class="min-w-52">
          {#each otherLinks as link}
            {@const ghIssue = parseGitHubIssue(link.url)}
            <DropdownMenu.Item class="gap-2 text-xs" onclick={() => openExternal(link.url)}>
              {#if ghIssue}
                <Github class="size-3 shrink-0" />
                <span class="font-mono">#{ghIssue.number}</span>
                <span class="text-muted-foreground ml-auto max-w-36 truncate">{ghIssue.repo}</span>
              {:else if isGitHub(link.url)}
                <Github class="size-3 shrink-0" />
                <span class="truncate">{link.title || link.externalId || link.url}</span>
              {:else}
                <Link2 class="size-3 shrink-0" />
                <span class="truncate">{link.title || link.externalId || link.url}</span>
              {/if}
            </DropdownMenu.Item>
          {/each}
        </DropdownMenu.Content>
      </DropdownMenu.Root>
    {/if}
  </span>
{/if}
