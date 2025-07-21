<div class="w-full flex justify-center mt-6">
    <form method="get" class="text-gray-50 max-w-md w-full p-6 bg-gray-600 rounded-2xl">
        <input hidden name="number" min=1 max=10000 value=1>
        <div class="sm:flex items-center mb-6">
            <div class="sm:w-1/4">
                <label for="form-intags" class="mb-1 sm:mb-0 pr-4 block font-bold sm:text-right">Includes</label>
            </div>
            <div class="sm:w-3/4">
                <input type="text" name="intags" id="form-intags" class="w-full text-gray-950 rounded py-1 px-2 border-2 bg-gray-300 border-gray-300 focus:bg-gray-50 focus:border-purple-500 focus:outline-none">
            </div>
        </div>
        <div class="sm:flex items-center mb-6">
            <div class="sm:w-1/4">
                <label for="form-extags" class="mb-1 sm:mb-0 pr-4 block font-bold sm:text-right">Excludes</label>
            </div>
            <div class="sm:w-3/4">
                <input type="text" name="extags" id="form-extags" class="w-full text-gray-950 rounded py-1 px-2 border-2 bg-gray-300 border-gray-300 focus:bg-gray-50 focus:border-purple-500 focus:outline-none">
            </div>
        </div>
        <div class="sm:flex items-center">
            <div class="sm:w-1/4"></div>
            <div class="sm:w-3/4">
                <input type="submit" value="Search" class="rounded bg-purple-500 hover:bg-purple-400 font-bold py-2 px-4">
            </div>
        </div>
    </form>
</div>
