<script lang="ts">
	import Edit from './Edit.svelte';
	import type { Resource } from './types';

	interface Props {
		resource: Resource;
	}

	let { resource = $bindable() }: Props = $props();
</script>

{#if resource}
	<details open class="">
		<summary class="">
			<span class="font-bold">Tags</span>
		</summary>
		<div class="">
			{#each resource.Tags as tag (tag)}
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
					<Edit bind:resource />
				</div>
			</details>
		</div>
	</details>
	{#if resource.Mimetype.startsWith('image')}
		<div class="place-center flex h-full min-h-0 w-full flex-grow flex-col">
			{#key resource.Id}
				<img
					src="/files/{resource.Id}"
					class="m-auto min-h-0 object-scale-down"
					alt={resource.Id}
				/>
			{/key}
		</div>
	{:else if resource.Mimetype.startsWith('video')}
		<div class="place-center flex h-full min-h-0 w-full flex-grow flex-col">
			{#key resource.Id}
				<video controls loop autoplay muted class="m-auto min-h-0 object-scale-down">
					<source src="/files/{resource.Id}" />
				</video>
			{/key}
		</div>
	{:else}
		<p>Could not display resource.</p>
	{/if}
{/if}
