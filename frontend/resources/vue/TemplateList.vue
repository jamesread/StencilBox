<template>
	<section>
		<SectionHeader title = "Templates" subtitle = "This is a list of templates, both builtin, and from your config directory." />

		<table>
			<thead>
				<tr>
					<th>Name</th>
					<th>Source</th>
					<th class = "small">Status</th>
				</tr>
			</thead>

			<tbody>
				<tr v-for="template in templates" :key="template.name">
					<td>{{ template.name }}</td>
					<td>{{ template.source }}</td>
					<td :class = "'small ' + template.statusClass ">{{ template.status }}</td>
				</tr>
			</tbody>
		</table>
	</section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';

	const templates = ref([]);

	async function getTemplates() {
		try {
			const response = await window.client.getTemplates();

			let tpl = response.templates;
			tpl.sort((a, b) => a.name.localeCompare(b.name));
			tpl = tpl.map(t => {
			    t.statusClass = t.status === 'OK' ? 'good' : t.status === 'error' ? 'critical' : 'unknown';
				return t;
			})

			templates.value = tpl;
			console.log('Templates loaded:', templates.value);
		} catch (error) {
			console.error('Error loading templates:', error);
		}
	}

	onMounted(() => {
		getTemplates();
	});
</script>
