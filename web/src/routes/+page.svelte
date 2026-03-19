<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import {
		Card,
		CardContent,
		CardDescription,
		CardFooter,
		CardHeader,
		CardTitle
	} from '$lib/components/ui/card';

	const pillars = [
		{
			title: 'All-Go Monolith',
			body: 'A single binary serves the UI shell and the control plane surface, keeping deployment aligned with the PRD.'
		},
		{
			title: 'Binary-First Delivery',
			body: 'The SvelteKit build lands in an embedded file system so the shipped artifact does not depend on Node.js at runtime.'
		},
		{
			title: 'Ticket-Driven Orchestration',
			body: 'This scaffold is intentionally narrow: one usable vertical slice that future issue-driven work can extend in place.'
		}
	];

	const stack = [
		'SvelteKit 2 static output',
		'Tailwind CSS 4 theme tokens',
		'shadcn-svelte component source',
		'Go go:embed asset serving'
	];
</script>

<svelte:head>
	<title>OpenASE Scaffold</title>
	<meta
		name="description"
		content="OpenASE vertical slice scaffold with SvelteKit, Tailwind, shadcn-svelte, and Go embedded static serving."
	/>
</svelte:head>

<div class="relative overflow-hidden">
	<div class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-foreground/20 to-transparent"></div>

	<section class="mx-auto flex min-h-screen w-full max-w-6xl flex-col px-6 py-10 sm:px-8 lg:px-10">
		<div class="mb-14 flex flex-wrap items-center justify-between gap-4">
			<div class="flex items-center gap-3">
				<div class="bg-foreground text-background flex size-10 items-center justify-center rounded-2xl text-sm font-semibold tracking-[0.24em]">
					OA
				</div>
				<div>
					<p class="text-xs uppercase tracking-[0.32em] text-muted-foreground">OpenASE</p>
					<p class="text-sm text-muted-foreground">F03 foundation slice</p>
				</div>
			</div>

			<div class="flex flex-wrap items-center gap-2">
				<Badge variant="outline">SvelteKit</Badge>
				<Badge variant="outline">Tailwind</Badge>
				<Badge variant="outline">shadcn-svelte</Badge>
				<Badge variant="outline">go:embed</Badge>
			</div>
		</div>

		<div class="grid gap-8 lg:grid-cols-[minmax(0,1.2fr)_minmax(22rem,0.8fr)] lg:items-end">
			<div class="space-y-8">
				<div class="space-y-5">
					<p class="text-sm font-medium uppercase tracking-[0.32em] text-muted-foreground">
						Issue-driven automation platform
					</p>
					<h1 class="max-w-4xl text-5xl leading-none font-semibold tracking-[-0.06em] text-balance sm:text-6xl lg:text-7xl">
						The web shell is now wired into the Go binary.
					</h1>
					<p class="max-w-2xl text-lg leading-8 text-muted-foreground">
						This scaffold establishes the frontend baseline for OpenASE: a static SvelteKit app,
						Tailwind theme tokens, shadcn-svelte components, and embedded delivery from the main
						service process.
					</p>
				</div>

				<div class="flex flex-wrap gap-3">
					<Button href="#stack">Inspect the stack</Button>
					<Button href="#slice" variant="outline">View the shipped slice</Button>
				</div>

				<div class="grid gap-4 sm:grid-cols-3">
					<div class="rounded-3xl border border-border/70 bg-background/70 p-4 backdrop-blur">
						<p class="text-xs uppercase tracking-[0.3em] text-muted-foreground">runtime</p>
						<p class="mt-3 text-3xl font-semibold tracking-[-0.05em]">1 binary</p>
						<p class="mt-2 text-sm text-muted-foreground">Static UI assets ship inside the Go server.</p>
					</div>
					<div class="rounded-3xl border border-border/70 bg-background/70 p-4 backdrop-blur">
						<p class="text-xs uppercase tracking-[0.3em] text-muted-foreground">front-end</p>
						<p class="mt-3 text-3xl font-semibold tracking-[-0.05em]">0 runtime UI deps</p>
						<p class="mt-2 text-sm text-muted-foreground">shadcn-svelte copies component source into the repo.</p>
					</div>
					<div class="rounded-3xl border border-border/70 bg-background/70 p-4 backdrop-blur">
						<p class="text-xs uppercase tracking-[0.3em] text-muted-foreground">delivery</p>
						<p class="mt-3 text-3xl font-semibold tracking-[-0.05em]">1 slice</p>
						<p class="mt-2 text-sm text-muted-foreground">Optimized for fast progress on the mainline.</p>
					</div>
				</div>
			</div>

			<Card class="bg-card/85 border-border/80 shadow-2xl shadow-black/5">
				<CardHeader>
					<CardTitle>Embedded UI contract</CardTitle>
					<CardDescription>
						The browser-facing shell is prepared for later CRUD, SSE, and ticket workflow pages.
					</CardDescription>
				</CardHeader>
				<CardContent class="space-y-4">
					<div class="rounded-2xl border border-dashed border-border/80 bg-muted/40 p-4">
						<p class="text-xs uppercase tracking-[0.3em] text-muted-foreground">build target</p>
						<p class="mt-2 font-mono text-sm">internal/webui/static/</p>
					</div>
					<div class="space-y-3">
						{#each stack as item}
							<div class="flex items-center justify-between rounded-2xl border border-border/70 px-4 py-3">
								<span class="text-sm font-medium">{item}</span>
								<Badge>{'ready'}</Badge>
							</div>
						{/each}
					</div>
				</CardContent>
				<CardFooter class="justify-between text-sm text-muted-foreground">
					<span>Prepared for binary-first deployment</span>
					<span class="font-mono">main</span>
				</CardFooter>
			</Card>
		</div>

		<section id="slice" class="mt-20 grid gap-5 lg:grid-cols-3">
			{#each pillars as pillar}
				<Card class="bg-background/80 backdrop-blur">
					<CardHeader>
						<CardTitle>{pillar.title}</CardTitle>
						<CardDescription>{pillar.body}</CardDescription>
					</CardHeader>
				</Card>
			{/each}
		</section>

		<section
			id="stack"
			class="mt-20 rounded-[2rem] border border-border/80 bg-background/80 px-6 py-8 backdrop-blur sm:px-8"
		>
			<div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
				<div class="max-w-2xl space-y-3">
					<p class="text-xs uppercase tracking-[0.32em] text-muted-foreground">Implementation notes</p>
					<h2 class="text-3xl leading-tight font-semibold tracking-[-0.05em]">
						A narrow base layer that future GitHub issues can build on directly.
					</h2>
					<p class="text-base leading-7 text-muted-foreground">
						The current page is static by design. The important part is that the asset pipeline,
						component system, and Go embedding boundary are already in place for subsequent ticket,
						workflow, and SSE work.
					</p>
				</div>

				<div class="grid gap-2 text-sm text-muted-foreground">
					<p>Output is generated by SvelteKit static adapter.</p>
					<p>Assets are embedded with the Go standard library.</p>
					<p>Runtime serving falls back to the prerendered shell for app routes.</p>
				</div>
			</div>
		</section>
	</section>
</div>
