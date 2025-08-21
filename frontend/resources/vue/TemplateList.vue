<template>
	<Section title = "Templates" :padding = false>
			<template #toolbar>
				<a href = "https://jamesread.github.io/StencilBox/templates/index.html" class = "button">
					Docs
					<HugeiconsIcon :icon = "LinkSquare01Icon" size = "24" />
				</a>

				<button class="good" @click="addTemplate" disabled>
					Add Template
				</button>
			</template>

		<Table :headers = "headers" :data = "templates">
			<template #cell-name="{ row, value }">
				<span v-if="!row.name" class="subtle">N/A</span>
				<router-link v-else :to="{ name: 'templateView', params: { name: row.name } }">
					{{ row.name }}
				</router-link>
			</template>
			<template #cell-buildConfigs="{ row, value }">
				{{ value.length }}
			</template>
		</Table>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { useRouter } from 'vue-router';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { LinkSquare01Icon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import Table from 'picocrank/vue/components/Table.vue';

	const headers = [
		{ label: 'Name', key: 'name', hidden: false, sortable: true },
		{ label: 'Source', key: 'source', hidden: false, sortable: false, linkFunc: null },
		{ label: 'Used by', key: 'buildConfigs', hidden: false, sortable: false, linkFunc: null },
		{ label: 'Status', key: 'status', hidden: false, sortable: false, linkFunc: null }
	];

	const templates = ref([]);
	const router = useRouter();

	onMounted(() => {
		getTemplates();
	});

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
		} catch (error) {
			console.error('Error loading templates:', error);
		}
	}
</script>
