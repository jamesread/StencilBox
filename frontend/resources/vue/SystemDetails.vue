<template>
	<section>
		<SectionHeader title = "System Details" subtitle = "This is a list of the system details." />

		<table>
			<tbody>
				<tr>
					<th>Templates path</th>
					<td>{{templatesPath}}</td>
				</tr>
				<tr>
					<th>Output path</th>
					<td>{{outputPath}}</td>
				</tr>
			</tbody>
		</table>
	</section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';

	const templatesPath = ref('...');
	const outputPath = ref('...');

	async function getStatus() {
		const status = await window.client.getStatus();

		templatesPath.value = status.templatesPath;
		outputPath.value = status.outputPath;
	}

	onMounted(() => {
		getStatus();
	});
</script>
