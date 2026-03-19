<script lang="ts">
	type ValidationIssue = {
		level: 'error' | 'warning' | string;
		message: string;
		line?: number;
		column?: number;
	};

	let {
		value = $bindable(''),
		issues = [],
		placeholder = ''
	}: {
		value: string;
		issues?: ValidationIssue[];
		placeholder?: string;
	} = $props();

	let overlay: HTMLPreElement | null = null;

	const highlightedMarkup = $derived(highlightHarness(value, placeholder));
	const hasErrors = $derived(issues.some((issue) => issue.level === 'error'));

	function syncScroll(event: Event) {
		if (!overlay) {
			return;
		}

		const target = event.currentTarget as HTMLTextAreaElement;
		overlay.scrollTop = target.scrollTop;
		overlay.scrollLeft = target.scrollLeft;
	}

	function highlightHarness(content: string, fallback: string) {
		const source = content || fallback;
		if (!source) {
			return '';
		}

		const normalized = source.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
		const parts = splitHarness(normalized);
		if (!parts) {
			return renderBody(normalized);
		}

		const sections = [
			'<span class="text-slate-500">---</span>',
			renderFrontmatter(parts.frontmatter),
			'<span class="text-slate-500">---</span>'
		];
		if (parts.body) {
			sections.push(renderBody(parts.body));
		}

		return sections.join('\n');
	}

	function splitHarness(content: string) {
		const lines = content.split('\n');
		if (lines[0] !== '---') {
			return null;
		}

		for (let index = 1; index < lines.length; index += 1) {
			if (lines[index] !== '---') {
				continue;
			}

			return {
				frontmatter: lines.slice(1, index).join('\n'),
				body: lines.slice(index + 1).join('\n')
			};
		}

		return null;
	}

	function renderFrontmatter(content: string) {
		let rendered = escapeHTML(content);
		rendered = rendered.replace(
			/^(\s*)([A-Za-z0-9_-]+)(\s*:)/gm,
			'$1<span class="text-amber-300">$2</span>$3'
		);
		rendered = rendered.replace(
			/("[^"]*"|'[^']*')/g,
			'<span class="text-emerald-300">$1</span>'
		);
		rendered = rendered.replace(
			/\b(true|false|null|\d+)\b/g,
			'<span class="text-fuchsia-300">$1</span>'
		);
		return rendered;
	}

	function renderBody(content: string) {
		let rendered = escapeHTML(content);
		rendered = rendered.replace(
			/\{\{[\s\S]*?\}\}/g,
			'<span class="text-sky-300">$&</span>'
		);
		rendered = rendered.replace(
			/\{%[\s\S]*?%\}/g,
			'<span class="text-emerald-300">$&</span>'
		);
		rendered = rendered.replace(
			/\{#[\s\S]*?#\}/g,
			'<span class="text-slate-500 italic">$&</span>'
		);
		return rendered;
	}

	function escapeHTML(content: string) {
		return content
			.replaceAll('&', '&amp;')
			.replaceAll('<', '&lt;')
			.replaceAll('>', '&gt;');
	}
</script>

<div
	class={`rounded-[1.75rem] border bg-slate-950 shadow-[0_24px_80px_rgba(15,23,42,0.32)] ${
		hasErrors ? 'border-rose-500/45' : 'border-slate-800/90'
	}`}
>
	<div class="flex items-center justify-between gap-4 border-b border-slate-800/90 px-5 py-3">
		<div class="flex items-center gap-2 text-[11px] font-medium uppercase tracking-[0.32em] text-slate-400">
			<span class="inline-flex size-2 rounded-full bg-emerald-400"></span>
			Harness editor
		</div>
		<div class="text-xs text-slate-500">{issues.length} validation marker(s)</div>
	</div>

	<div class="relative">
		<pre
			bind:this={overlay}
			class="pointer-events-none absolute inset-0 overflow-auto p-5 font-mono text-[13px] leading-6 whitespace-pre-wrap break-words"
			aria-hidden="true"
		>{@html `${highlightedMarkup}<br />`}</pre>
		<textarea
			bind:value
			class="relative z-10 h-[30rem] w-full resize-none overflow-auto bg-transparent p-5 font-mono text-[13px] leading-6 text-transparent caret-white outline-none selection:bg-sky-300/30"
			spellcheck="false"
			placeholder={placeholder}
			onscroll={syncScroll}
		></textarea>
	</div>
</div>
