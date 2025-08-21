<template>
	<Section
		:padding = "false"
		title="Build Configurations"
		subtitle="This is a list of build configurations, both builtin, and from your config directory."
		>

			<template #toolbar>
				<a href = "https://jamesread.github.io/StencilBox/buildconfigs/index.html" class = "button">
					Docs
					<HugeiconsIcon :icon = "LinkSquare01Icon" size = "24" />
				</a>

				<button class = "neutral" disabled>
					Git Pull
					<HugeiconsIcon :icon = "GitCommitIcon" size = "24" />
				</button>
			</template>

		<Table
			:headers = "headers"
			:data = "buildConfigs"
			>

			<template #cell-name="{ row, value }">
					<span v-if="!row.name" class = "subtle">
						N/A
					</span>
			        <router-link v-else :to="{ name: 'buildConfig', params: { name: row.name } }">
						{{ row.name }}
					</router-link>
			</template>

			<template #cell-template="{ row, value }">
					<span v-if="!row.template" class = "subtle">N/A</span>

					<router-link v-else :to="{ name: 'templateView', params: { name: row.template } }">
						{{ row.template }}
					</router-link>
			</template>

			<template #cell-status="{ row, value }">
					<span v-if="row.errorMessage" class = "bad">
						<button class = "bad"@click="showErrorMessage(row)">Show Error</button>
					</span>
					<span v-else class = "annotation good">
						OK
					</span>
			</template>
		</Table>
	</Section>

	<Section title = "Create build config">
		<template #toolbar>
			<a href = "https://jamesread.github.io/StencilBox/buildconfigs/index.html" class = "button">
				Docs
				<HugeiconsIcon :icon = "LinkSquare01Icon" size = "24" />
			</a>
		</template>

		<p>
		    To create a new build config, you need to create a YAML file in the build configs directory. You cannot create a build config from the UI yet.
		</p>
		<dl v-if="status">
			<dt>Build configs directory</dt>
			<dd>
				{{ status.buildConfigsDir }}
				<span v-if="status.inContainer"> (container volume)</span>
				<span v-else> (on host)</span>
			</dd>

			<dt>Git repository</dt>
			<dd class = "subtle">Not yet supported</dd>
		</dl>
	</Section>

	<dialog ref="errorDialog" class = "bad">
		<h2>Error loading build config</h2>
		<p v-if="errorMessage">{{ errorMessage }}</p>
		<p v-else>Unknown error</p>

		<button @click="errorDialog.close()">Close</button>
	</dialog>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { HugeiconsIcon } from '@hugeicons/vue';
import { LinkSquare01Icon, GitCommitIcon } from '@hugeicons/core-free-icons';
import Section from 'picocrank/vue/components/Section.vue';
import Table from 'picocrank/vue/components/Table.vue';

const headers = ref([
	{ label: 'Name', key: 'name', width: '25%', sortable: true },
	{ label: 'Template', key: 'template', width: '25%', sortable: true },
	{ label: 'Filename', key: 'filename', width: '25%' },
	{ label: 'Status', key: 'status', width: '25%' }
])

const buildConfigs = ref([]);
const router = useRouter();
const status = ref(null);
const errorDialog = ref(null);
const errorMessage = ref(null);

async function getBuildConfigs() {
	try {
		const response = await window.client.getBuildConfigs();

		let configs = response.buildConfigs;
		configs.sort((a, b) => a.name.localeCompare(b.name));
		configs = configs.map(c => {
			c.statusClass = c.status === 'OK' ? 'good' : c.status === 'error' ? 'critical' : 'unknown';
			return c;
		});

		buildConfigs.value = configs;
		console.log('Build configs loaded:', buildConfigs.value);
	} catch (error) {
		console.error('Error loading build configs:', error);
	}
}

async function getStatus() {
	const response = await window.client.getStatus();
	status.value = response;
}

function showErrorMessage(config) {
	errorMessage.value = config.errorMessage;
	errorDialog.value.showModal();
}

onMounted(() => {
	getStatus();
	getBuildConfigs();
});
</script>
