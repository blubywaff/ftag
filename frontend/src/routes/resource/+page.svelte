<script lang="ts">
	import Viewer from '$lib/Viewer.svelte';
	import { onMount } from 'svelte';
	import type { Resource } from '$lib/types';

	interface View {
		resource: Resource | null;
	}

	let view: View = $state({
		resource: null
	});

	onMount(async () => {
		let loc = new URL(location.href);
		let id = loc.searchParams.get('id') || '';

		let url = new URL(`${location.origin}/api/resource`);
		url.searchParams.append('id', id);

		let res = await fetch(url);
		if (res.status !== 200) {
			view.resource = null;
			return;
		}

		let rsrc = await res.json();
		view.resource = rsrc;
	});
</script>

{#if view.resource !== null}
	<div class="flex h-screen flex-col">
		<Viewer bind:resource={view.resource} />
	</div>
{:else}
	<p>Resource not found.</p>
{/if}
