<script lang="ts">
	import { onMount } from 'svelte';

	interface Resource {
		Id: string;
		Mimetype: string;
		CreatedAt: string;
		Tags: string[];
	}
	interface View {
		resource: Resource | null;
		domvideo: HTMLVideoElement | undefined;
	}

	let view: View = $state({
		resource: null,
		domvideo: undefined
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
		<details open class="">
			<summary class="">
				<span class="font-bold">Tags</span>
			</summary>
			<div class="">
				{#each view.resource.Tags as tag (tag)}
					<p
						class="mr-1 mb-2 inline-block rounded-full bg-gray-300 px-2 py-1 text-center text-gray-800"
					>
						{tag}
					</p>
				{/each}
				<details class="flex justify-center" open>
					<summary>
						<span class="font-bold">Edit</span>
					</summary>
					<div class="flex w-full justify-center">
						<form
							method="post"
							enctype="multipart/form-data"
							class="mb-6 w-full max-w-md rounded-2xl bg-gray-600 p-6"
						>
							<input hidden name="resourceid" value={view.resource.Id} />
							<div class="mb-6 items-center sm:flex">
								<div class="sm:w-1/4">
									<label for="addtags" class="mb-1 block pr-4 font-bold sm:mb-0 sm:text-right"
										>Add</label
									>
								</div>
								<div class="sm:w-3/4">
									<input
										type="text"
										name="addtags"
										id="addtags"
										class="w-full rounded border-2 border-gray-300 bg-gray-300 px-2 py-1 text-gray-950 focus:border-purple-500 focus:bg-gray-50 focus:outline-none"
									/>
								</div>
							</div>
							<div class="mb-6 items-center sm:flex">
								<div class="sm:w-1/4">
									<label for="deltags" class="mb-1 block pr-4 font-bold sm:mb-0 sm:text-right"
										>Remove</label
									>
								</div>
								<div class="sm:w-3/4">
									<input
										type="text"
										name="deltags"
										id="deltags"
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
										class="rounded bg-purple-500 px-4 py-2 font-bold hover:bg-purple-400"
									/>
								</div>
							</div>
						</form>
					</div>
				</details>
			</div>
		</details>
		{#if view.resource.Mimetype.startsWith('image')}
			<div class="place-center flex h-full min-h-0 w-full flex-grow flex-col">
				<img
					src="/files/{view.resource.Id}"
					class="m-auto min-h-0 object-scale-down"
					alt={view.resource.Id}
				/>
			</div>
		{:else if view.resource.Mimetype.startsWith('video')}
			<div class="place-center flex h-full min-h-0 w-full flex-grow flex-col">
				<video
					bind:this={view.domvideo}
					controls
					loop
					autoplay
					muted
					class="m-auto min-h-0 object-scale-down"
				>
					<source src="/files/{view.resource.Id}" />
				</video>
			</div>
		{/if}
	</div>
{:else}
	<p>Resource not found.</p>
{/if}
