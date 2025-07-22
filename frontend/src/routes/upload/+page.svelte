<script lang="ts">
	let form: HTMLFormElement | undefined = $state();

	async function onsave() {
		await fetch(`/api/upload`, {
			method: 'POST',
			body: new FormData(form)
		});
		location.pathname = '/public/query';
	}
</script>

<div class="mt-6 flex w-full justify-center">
	<form
		bind:this={form}
		method="post"
		enctype="multipart/form-data"
		class="w-full max-w-md rounded-2xl bg-gray-600 p-6"
	>
		<input
			type="file"
			name="uploadfile"
			multiple
			class="mb-6 file:rounded file:border-2 file:border-gray-300 file:bg-gray-300 file:hover:border-gray-200 file:hover:bg-gray-200 focus:outline-none file:focus:border-purple-500"
		/>
		<div class="mb-6">
			<label for="tags" class="mb-1 block pr-4 font-bold">Tags</label>
			<input
				type="text"
				name="tags"
				id="tags"
				class="w-full rounded border-2 border-gray-300 bg-gray-300 px-2 py-1 text-gray-950 focus:border-purple-500 focus:bg-gray-50 focus:outline-none"
			/>
		</div>
		<input
			type="submit"
			value="Upload"
			onclick={async (e) => {
				e.preventDefault();
				await onsave();
				return false;
			}}
			class="rounded bg-purple-500 px-4 py-2 font-bold hover:bg-purple-400"
		/>
	</form>
</div>
