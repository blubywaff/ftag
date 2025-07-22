<script lang="ts">
	import * as settings from '$lib/settings.svelte';
	import '../app.css';
	let { children } = $props();
	settings.read();
</script>

{@render children()}
