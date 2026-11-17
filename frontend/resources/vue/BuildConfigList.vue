<template>
	<Section
		:padding = "false"
		title="Build Configurations"
		:icon="Configuration01Icon"
		subtitle="This is a list of build configurations, both builtin, and from your config directory."
		>

			<template #toolbar>
				<a href = "https://jamesread.github.io/StencilBox/buildconfigs/index.html" class = "button inline-icon">
					<HugeiconsIcon :icon = "LinkSquare01Icon" width = "1em" height = "1em" :strokeWidth = "2.5" aria-hidden = "true" />
					<span>Docs</span>
				</a>

				<button
					type = "button"
					class = "inline-icon neutral"
					:disabled="!canGitPull || isPulling"
					@click="handleGitPull"
				>
					<HugeiconsIcon :icon = "GitCommitIcon" width = "1em" height = "1em" :strokeWidth = "2.5" aria-hidden = "true" />
					<span>{{ isPulling ? 'Pulling...' : 'Git Pull' }}</span>
				</button>

				<button v-if="canManage" type="button" class="inline-icon good" @click="createDialog.open()">
					<HugeiconsIcon :icon="FileAddIcon" width="1em" height="1em" :strokeWidth="2.5" aria-hidden="true" />
					<span>Create</span>
				</button>
			</template>

		<Table
			:headers = "headers"
			:data = "buildConfigs"
			>

			<template #cell-name="{ row }">
					<span v-if="!row.name" class = "subtle">
						N/A
					</span>
			        <router-link v-else :to="{ name: 'buildConfig', params: { name: row.name } }">
						{{ row.name }}
					</router-link>
			</template>

			<template #cell-template="{ row }">
					<span v-if="!row.template" class = "subtle">N/A</span>

					<router-link v-else :to="{ name: 'templateView', params: { name: row.template } }">
						{{ row.template }}
					</router-link>
			</template>

			<template #cell-status="{ row }">
					<button v-if="row.errorMessage" type = "button" class = "tag bad" @click="showErrorMessage(row)">Error</button>
					<span v-else class = "tag good">OK</span>
			</template>
		</Table>
	</Section>

	<Section title = "Create build config" :icon = "FileAddIcon">
		<template #toolbar>
			<a href = "https://jamesread.github.io/StencilBox/buildconfigs/index.html" class = "button inline-icon">
				<HugeiconsIcon :icon = "LinkSquare01Icon" width = "1em" height = "1em" :strokeWidth = "2.5" aria-hidden = "true" />
				<span>Docs</span>
			</a>
		</template>

		<p v-if="canManage">
			Use <strong>Create</strong> above to make a build config here, then add data files to it from its page.
			You can also write build config YAML files in the build configs directory by hand.
		</p>
		<p v-else>
			To create a new build config, create a YAML file in the build configs directory, or ask an
			administrator for the <code>buildconfigs.manage</code> permission.
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

	<CreateBuildConfigDialog ref="createDialog" @created="onCreated" />

	<dialog ref="errorDialog" class = "bad">
		<h2>Error</h2>
		<p v-if="errorMessage">{{ errorMessage }}</p>
		<p v-else>Unknown error</p>

		<button @click="errorDialog.close()">Close</button>
	</dialog>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { HugeiconsIcon } from '@hugeicons/vue';
import { Configuration01Icon, FileAddIcon, GitCommitIcon, LinkSquare01Icon } from '@hugeicons/core-free-icons';
import Section from 'picocrank/vue/components/Section.vue';
import Table from 'picocrank/vue/components/Table.vue';
import CreateBuildConfigDialog from './CreateBuildConfigDialog.vue';
import { useAppStatus } from '../../scripts/appStatus.js';

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
const canGitPull = ref(false);
const isPulling = ref(false);
const createDialog = ref(null);
const { hasPermission } = useAppStatus();
const canManage = computed(() => hasPermission('buildconfigs.manage'));

function onCreated(name) {
	router.push({ name: 'buildConfig', params: { name } });
}

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
		canGitPull.value = response.canGitPull;
		console.log('Build configs loaded:', buildConfigs.value);
	} catch (error) {
		console.error('Error loading build configs:', error);
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
			// Reload build configs after successful pull
			await getBuildConfigs();
			// Show success message (you might want to add a toast/notification here)
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
