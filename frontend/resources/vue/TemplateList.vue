<template>
	<section>
		<SectionHeader title="Templates" subtitle="This is a list of templates, both builtin, and from your config directory.">
			<template #actions>
				<button class="good" @click="addTemplate" disabled>
					Add Template
				</button>
			</template>
		</SectionHeader>

		<table>
			<thead>
				<tr>
					<th>Name</th>
					<th>Source</th>
					<th>Used by</th>
					<th class = "small">Status</th>
				</tr>
			</thead>

			<tbody>
				<tr v-for="template in templates" :key="template.name">
					<td><a href = "#" @click.prevent = "openTemplate(template)">{{ template.name }}</a></td>
					<td>{{ template.source }}</td>
					<td>{{ template.buildConfigs.length }}</td>
					<td :class = "'small ' + template.statusClass ">{{ template.status }}</td>
				</tr>
			</tbody>
		</table>
	</section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { useRouter } from 'vue-router';

	const templates = ref([]);
	const router = useRouter();

	function openTemplate(template) {
		router.push({ name: 'templateView', params: { name: template.name } });
	}

	function addTemplate() {
		router.push({ name: 'templateAdd' });

	}

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
