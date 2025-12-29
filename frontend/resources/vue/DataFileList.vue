<template>
	<Section
		:padding="false"
		title="Data Files"
		subtitle="YAML files that provide data to templates during the build process."
	>

		<template #toolbar>
			<a href="https://jamesread.github.io/StencilBox/buildconfigs/index.html" class="button">
				Docs
				<HugeiconsIcon :icon="LinkSquare01Icon" size="24" />
			</a>

			<button
				class="neutral"
				:disabled="!canGitPull || isPulling"
				@click="handleGitPull"
			>
				{{ isPulling ? 'Pulling...' : 'Git Pull' }}
				<HugeiconsIcon :icon="GitCommitIcon" size="24" />
			</button>
		</template>

		<Table
			:headers="headers"
			:data="dataFiles"
		>

			<template #cell-name="{ row, value }">
				<router-link :to="{ name: 'dataFileView', params: { buildConfigName: row.buildConfigName, datafileName: row.name } }">
					{{ row.name }}
				</router-link>
			</template>

			<template #cell-buildConfig="{ row, value }">
				<router-link :to="{ name: 'buildConfig', params: { name: row.buildConfigName } }">
					{{ row.buildConfigName }}
				</router-link>
			</template>

			<template #cell-path="{ row, value }">
				<code>{{ row.path }}</code>
			</template>
		</Table>
	</Section>

	<dialog ref="errorDialog" class="bad">
		<h2>Error</h2>
		<p v-if="errorMessage">{{ errorMessage }}</p>
		<p v-else>Unknown error</p>

		<button @click="errorDialog.close()">Close</button>
	</dialog>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { HugeiconsIcon } from '@hugeicons/vue';
import { LinkSquare01Icon, GitCommitIcon } from '@hugeicons/core-free-icons';
import Section from 'picocrank/vue/components/Section.vue';
import Table from 'picocrank/vue/components/Table.vue';

const headers = ref([
	{ label: 'Name', key: 'name', width: '25%', sortable: true },
	{ label: 'Build Config', key: 'buildConfig', width: '25%', sortable: true },
	{ label: 'Path', key: 'path', width: '50%' }
]);

const dataFiles = ref([]);
const canGitPull = ref(false);
const isPulling = ref(false);
const errorDialog = ref(null);
const errorMessage = ref(null);

async function loadDataFiles() {
	try {
		const response = await window.client.listDataFiles({});
		dataFiles.value = response.dataFiles || [];
		dataFiles.value.sort((a, b) => {
			// Sort by build config name, then by data file name
			if (a.buildConfigName !== b.buildConfigName) {
				return a.buildConfigName.localeCompare(b.buildConfigName);
			}
			return a.name.localeCompare(b.name);
		});
		canGitPull.value = response.canGitPull;
	} catch (error) {
		console.error('Error loading data files:', error);
	}
}

async function handleGitPull() {
	if (!canGitPull.value || isPulling.value) {
		return;
	}

	isPulling.value = true;
	try {
		const response = await window.client.gitPull({});

		if (response.success) {
			// Reload data files after successful pull
			await loadDataFiles();
			console.log('Git pull successful:', response.message);
		} else {
			// Show error message
			errorMessage.value = response.message || 'Git pull failed';
			errorDialog.value.showModal();
		}
	} catch (error) {
		console.error('Error performing git pull:', error);
		errorMessage.value = 'Git pull failed: ' + error.message;
		errorDialog.value.showModal();
	} finally {
		isPulling.value = false;
	}
}

onMounted(() => {
	loadDataFiles();
});
</script>
