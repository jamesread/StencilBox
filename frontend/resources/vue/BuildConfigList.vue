<template>
	<section>
		<SectionHeader title="Build Configurations"
			subtitle="This is a list of build configurations, both builtin, and from your config directory.">

			<template #actions>
				<button class = "neutral" disabled>
					Git Pull
					<HugeiconsIcon :icon = "GitCommitIcon" size = "24" />
				</button>
				<a href = "https://jamesread.github.io/StencilBox/buildconfigs/index.html" class = "button">
					Open docs
					<HugeiconsIcon :icon = "LinkSquare01Icon" size = "24" />
				</a>
			</template>
		</SectionHeader>

		<table>
			<thead>
				<tr>
					<th>Name</th>
					<th>Template</th>
					<th>Filename</th>
					<th>Status</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="config in buildConfigs" :key="config.name">
					<td>
						<span v-if="!config.name" class = "subtle">
							N/A
						</span>
						<a v-else href="#" @click.prevent="openBuild(config)">{{ config.name }}</a>
					</td>
					<td>
						<span v-if="!config.template" class = "subtle">N/A</span>
						<a v-else href="#" @click.prevent="openTemplate(config)">{{ config.template }}</a>
					</td>
					<td>
						{{ config.filename }}
					</td>
					<td v-if="config.errorMessage" class = "bad">
						<button class = "bad"@click="showErrorMessage(config)">Show Error</button>
					</td>
					<td v-else class = "good">
						OK
					</td>
				</tr>
			</tbody>
		</table>

		<p v-if="status"><strong>Build configs directory:</strong> {{ status.buildConfigsDir }}
			<span v-if="status.inContainer"> (container volume)</span>
			<span v-else> (on host)</span>
		</p>
	</section>

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

const buildConfigs = ref([]);
const router = useRouter();
const status = ref(null);
const errorDialog = ref(null);
const errorMessage = ref(null);

function openBuild(config) {
	router.push({ name: 'buildConfig', params: { name: config.name } });
}

function openTemplate(config) {
	router.push({ name: 'templateView', params: { name: config.template } });
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
