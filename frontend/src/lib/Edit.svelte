<script lang="ts">
	import type { Resource } from './types';

	interface Props {
		resource: Resource;
	}

	interface TagUpdate {
		AddTags: string;
		DelTags: string;
		ResourceId: string;
	}

	let { resource = $bindable() }: Props = $props();

	let update: TagUpdate = $state({
		AddTags: '',
		DelTags: '',
		ResourceId: ''
	});

	async function onsave() {
		update.ResourceId = resource.Id;

		let url = new URL(`${location.origin}/api/resource/tags`);
		let res = await fetch(url, { method: 'POST', body: JSON.stringify(update) });
		let rsrc = await res.json();
		resource = rsrc;
	}
</script>

{#if resource}
	<form
		method="post"
		enctype="multipart/form-data"
		class="mb-6 w-full max-w-md rounded-2xl bg-gray-600 p-6"
	>
		<div class="mb-6 items-center sm:flex">
			<div class="sm:w-1/4">
				<label for="addtags" class="mb-1 block pr-4 font-bold sm:mb-0 sm:text-right">Add</label>
			</div>
			<div class="sm:w-3/4">
				<input
					type="text"
					name="addtags"
					id="addtags"
					bind:value={update.AddTags}
					class="w-full rounded border-2 border-gray-300 bg-gray-300 px-2 py-1 text-gray-950 focus:border-purple-500 focus:bg-gray-50 focus:outline-none"
				/>
			</div>
		</div>
		<div class="mb-6 items-center sm:flex">
			<div class="sm:w-1/4">
				<label for="deltags" class="mb-1 block pr-4 font-bold sm:mb-0 sm:text-right">Remove</label>
			</div>
			<div class="sm:w-3/4">
				<input
					type="text"
					name="deltags"
					id="deltags"
					bind:value={update.DelTags}
					class="w-full rounded border-2 border-gray-300 bg-gray-300 px-2 py-1 text-gray-950 focus:border-purple-500 focus:bg-gray-50 focus:outline-none"
				/>
			</div>
		</div>
		<div class="items-center sm:flex">
			<div class="sm:w-1/4"></div>
			<div class="sm:w-3/4">
				<input
					type="submit"
					value="Save"
					onclick={(e) => {
						e.preventDefault();
						onsave();
						return false;
					}}
					class="rounded bg-purple-500 px-4 py-2 font-bold hover:bg-purple-400"
				/>
			</div>
		</div>
	</form>
{/if}
