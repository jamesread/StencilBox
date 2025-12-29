<template>
	<Section
		:title="'Data File: ' + (dataFile?.name || 'Loading...')"
		subtitle="View the contents of a data file."
	>

		<template #toolbar>
			<a href="https://jamesread.github.io/StencilBox/buildconfigs/index.html" class="button">
				Docs
				<HugeiconsIcon :icon="LinkSquare01Icon" size="24" />
			</a>
		</template>

		<dl v-if="dataFile">
			<dt>Name</dt>
			<dd>{{ dataFile.name }}</dd>

			<dt>Build Config</dt>
			<dd>
				<router-link :to="{ name: 'buildConfig', params: { name: dataFile.buildConfigName } }" class="link">
					{{ dataFile.buildConfigName }}
				</router-link>
			</dd>

			<dt>Path</dt>
			<dd>
				<code>{{ dataFile.path }}</code>
			</dd>
		</dl>

		<p v-if="!dataFile && !error" class="subtle">Loading...</p>
		<p v-if="error" class="bad">{{ error }}</p>
	</Section>

	<Section title="Content" v-if="content">
		<pre class="datafile-content"><code>{{ content }}</code></pre>
	</Section>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { HugeiconsIcon } from '@hugeicons/vue';
import { LinkSquare01Icon } from '@hugeicons/core-free-icons';
import Section from 'picocrank/vue/components/Section.vue';

const props = defineProps({
	buildConfigName: {
		type: String,
		required: true
	},
	datafileName: {
		type: String,
		required: true
	}
});

const dataFile = ref(null);
const content = ref('');
const error = ref('');

async function loadDataFile() {
	try {
		const response = await window.client.getDataFile({
			buildConfigName: props.buildConfigName,
			datafileName: props.datafileName
		});

		dataFile.value = {
			name: response.datafileName,
			path: response.path,
			buildConfigName: response.buildConfigName
		};
		content.value = response.content;
	} catch (err) {
		console.error('Error loading data file:', err);
		error.value = err.message || 'Failed to load data file';
	}
}

onMounted(() => {
	loadDataFile();
});
</script>

<style scoped>
.datafile-content {
	background-color: #f5f5f5;
	border: 1px solid #ddd;
	border-radius: 4px;
	padding: 1em;
	overflow-x: auto;
	font-family: 'Courier New', monospace;
	font-size: 0.9em;
	line-height: 1.5;
}

.datafile-content code {
	background: transparent;
	padding: 0;
	border: none;
}
</style>
