<script lang="ts">
	import * as set from '$lib/settings.svelte';
</script>

<div class="flex h-screen w-full flex-col justify-center">
	<div class="flex w-full flex-row justify-center">
		<form method="post" class="w-full max-w-lg rounded-2xl bg-gray-600 p-6">
			<fieldset class="">
				<legend class="mb-6 text-xl font-bold">View Page</legend>
				<label for="view-tags" class="mb-1 block font-bold">Tag Visibility</label>
				<select
					name="view-tags"
					id="view-tags"
					required
					class="mb-6 text-gray-950"
					bind:value={set.settings.defaultTagView}
				>
					{#each set.options.tagView as view (view)}
						<option>{view}</option>
					{/each}
				</select>
				<label for="def-ex" class="mb-1 block font-bold">Default Excludes</label>
				<input
					type="text"
					name="def-ex"
					id="def-ex"
					bind:value={set.settings.defaultExcludes}
					class="mb-6 w-full rounded border-2 border-gray-300 bg-gray-300 px-2 py-1 text-gray-950 focus:border-purple-500 focus:bg-gray-50 focus:outline-none"
				/>
			</fieldset>
			<input
				type="submit"
				value="Save"
				class="rounded bg-purple-500 px-4 py-2 font-bold hover:bg-purple-400"
				onclick={(e) => {
					e.preventDefault();
					console.log($state.snapshot(set.settings));
					set.write();
					return false;
				}}
			/>
		</form>
	</div>
</div>
