<div class="mt-6 flex w-full justify-center">
	<form method="get" class="w-full max-w-md rounded-2xl bg-gray-600 p-6 text-gray-50">
		<input hidden name="number" min="1" max="10000" value="1" />
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
				/>
			</div>
		</div>
	</form>
</div>
