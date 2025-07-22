<script lang="ts">
	import { pushState } from '$app/navigation';
	import Viewer from '$lib/Viewer.svelte';
	import { onMount } from 'svelte';
	import type { Resource } from '$lib/types';

	interface Query {
		prepared: boolean;
		resources: Resource[];
		intags: string;
		extags: string;
		number: number;
	}

	interface View {
		resource: Resource | null;
	}

	let query: Query = $state({
		prepared: false,
		resources: [],
		intags: '',
		extags: '',
		number: 1
	});

	let view: View = $state({
		resource: null
	});

	async function updateView() {
		if (!query.prepared) {
			return;
		}

		let url = new URL(`${location.origin}/api/query`);
		url.searchParams.append('intags', query.intags);
		url.searchParams.append('extags', query.extags);
		url.searchParams.append('number', '' + query.number);

		let res = await fetch(url);
		let rsrc = await res.json();

		query.resources = rsrc;
		view.resource = query.resources[0];
	}

	onMount(async () => {
		let url = new URL(location.href);
		query.prepared =
			url.searchParams.has('intags') ||
			url.searchParams.has('extags') ||
			url.searchParams.has('number');
		query.intags = url.searchParams.get('intags') || '';
		query.extags = url.searchParams.get('extags') || '';
		query.number = Number(url.searchParams.get('number')) || 1;
		await updateView();
	});

	async function onquery() {
		let loc = new URL(location.origin + location.pathname);
		loc.searchParams.append('intags', query.intags);
		loc.searchParams.append('extags', query.extags);
		loc.searchParams.append('number', '' + query.number);
		pushState(loc, '');

		query.prepared = true;
		await updateView();
	}
</script>

{#if view.resource !== null}
	<div class="flex h-screen flex-col">
		<div class="flex flex-row justify-around gap-x-4 bg-gray-500">
			<button
				onclick={() => {
					query.number--;
					onquery();
				}}
				class="">Prev</button
			>
			<button
				onclick={() => {
					query.number++;
					onquery();
				}}
				class="">Next</button
			>
		</div>
		<Viewer resource={view.resource} />
	</div>
{/if}
<div class="mt-6 flex w-full justify-center">
	<form method="get" class="w-full max-w-md rounded-2xl bg-gray-600 p-6 text-gray-50">
		<input hidden name="number" min="1" max="10000" bind:value={query.number} />
		<div class="mb-6 items-center sm:flex">
			<div class="sm:w-1/4">
				<label for="form-intags" class="mb-1 block pr-4 font-bold sm:mb-0 sm:text-right"
					>Includes</label
				>
			</div>
			<div class="sm:w-3/4">
				<input
					type="text"
					name="intags"
					id="form-intags"
					bind:value={query.intags}
					class="w-full rounded border-2 border-gray-300 bg-gray-300 px-2 py-1 text-gray-950 focus:border-purple-500 focus:bg-gray-50 focus:outline-none"
				/>
			</div>
		</div>
		<div class="mb-6 items-center sm:flex">
			<div class="sm:w-1/4">
				<label for="form-extags" class="mb-1 block pr-4 font-bold sm:mb-0 sm:text-right"
					>Excludes</label
				>
			</div>
			<div class="sm:w-3/4">
				<input
					type="text"
					name="extags"
					id="form-extags"
					bind:value={query.extags}
					class="w-full rounded border-2 border-gray-300 bg-gray-300 px-2 py-1 text-gray-950 focus:border-purple-500 focus:bg-gray-50 focus:outline-none"
				/>
			</div>
		</div>
		<div class="items-center sm:flex">
			<div class="sm:w-1/4"></div>
			<div class="sm:w-3/4">
				<input
					type="submit"
					value="Search"
					class="rounded bg-purple-500 px-4 py-2 font-bold hover:bg-purple-400"
					onclick={(e) => {
						e.preventDefault();
						query.number = 1;
						onquery();
						return false;
					}}
				/>
			</div>
		</div>
	</form>
</div>
