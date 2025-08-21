<template>
	<Section title = "System Details">
		<dl>
			<dt>Build configs directory</dt>
			<dd>{{status?.buildConfigsDir}}
				<span v-if="status?.inContainer"> (container volume)</span>
				<span v-else> (on host)</span>
			</dd>

			<dt>Templates path</dt>
			<dd>{{status?.templatesPath}}</dd>

			<dt>Output path</dt>
			<dd>{{status?.outputPath}}</dd>

			<dt>In container</dt>
			<dd>{{status?.inContainer}}</dd>
		</dl>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import Section from 'picocrank/vue/components/Section.vue';

	const status = ref(null)

	async function getStatus() {
		const ret = await window.client.getStatus();

		status.value = ret
	}

	onMounted(() => {
		getStatus();
	});
</script>
