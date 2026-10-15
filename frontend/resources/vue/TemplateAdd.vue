<template>
	<Section title = "Add Template" :icon = "FileAddIcon" subtitle = "Custom templates are directories of files, picked up when the web UI loads.">
		<ol>
			<li>
				Create a directory for your template in the custom templates directory:
				<pre><code>{{ customTemplatesPath || '/config/templates' }}/my-template/</code></pre>
			</li>
			<li>
				Add an <code>index.html</code> to it. This is a Go template; copying a built-in template is a good starting point.
			</li>
			<li>
				Optionally, add a <code>metadata.yaml</code> with a <code>description</code> and <code>documentation_url</code>.
			</li>
			<li>
				Set <code>template: my-template</code> in a build config, then reload this page.
			</li>
		</ol>

		<p>A custom template with the same name as a built-in template replaces it.</p>

		<fieldset>
			<a class = "button inline-icon" href = "https://jamesread.github.io/StencilBox/templates/index.html">
				<HugeiconsIcon :icon = "LinkSquare01Icon" width = "1em" height = "1em" :strokeWidth = "2.5" aria-hidden = "true" />
				<span>Template docs</span>
			</a>
			<router-link class = "button" :to = "{ name: 'templateList' }">Back to templates</router-link>
		</fieldset>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { FileAddIcon, LinkSquare01Icon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';

	const customTemplatesPath = ref('');

	onMounted(async () => {
		try {
			const status = await window.client.getStatus();
			customTemplatesPath.value = status.customTemplatesPath;
		} catch (error) {
			console.error('Error loading status:', error);
		}
	});
</script>
