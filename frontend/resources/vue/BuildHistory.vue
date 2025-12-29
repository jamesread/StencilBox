<template>
	<Section title="Build History" subtitle="Recent rebuilds of this build configuration" :padding="false">
		<p v-if="history.length === 0" class="subtle padding">No build history available yet.</p>

		<Table v-else :headers="headers" :data="history">
			<template #cell-timestamp="{ row, value }">
				{{ formatTimestamp(row.timestamp) }}
			</template>

			<template #cell-status="{ row, value }">
				<span :class="row.isError ? 'bad' : 'good'">
					{{ row.status }}
				</span>
			</template>

			<template #cell-type="{ row, value }">
				<span v-if="row.isAutoRebuild" class="annotation">Auto-rebuild</span>
				<span v-else class="annotation">Manual</span>
			</template>

			<template #cell-outputSize="{ row, value }">
				<span v-if="row.outputSizeHumanReadable">{{ row.outputSizeHumanReadable }}</span>
				<span v-else class="subtle">N/A</span>
			</template>

			<template #cell-buildUrl="{ row, value }">
				<span v-if="row.buildUrl">
					<a :href="row.buildUrl" target="_blank">View</a>
				</span>
				<span v-else class="subtle">N/A</span>
			</template>
		</Table>
	</Section>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue';
import Section from 'picocrank/vue/components/Section.vue';
import Table from 'picocrank/vue/components/Table.vue';

const props = defineProps({
	configName: {
		type: String,
		required: true
	},
	outputDir: {
		type: String,
		default: ''
	}
});

const history = ref([]);

const headers = [
	{ label: 'Time', key: 'timestamp', sortable: true },
	{ label: 'Status', key: 'status', sortable: false },
	{ label: 'Type', key: 'type', sortable: false },
	{ label: 'Output Size', key: 'outputSize', sortable: false },
	{ label: 'Build URL', key: 'buildUrl', sortable: false }
];

function formatTimestamp(timestamp) {
	if (!timestamp) return 'N/A';

	// Handle bigint (from proto) or number
	const ts = typeof timestamp === 'bigint' ? Number(timestamp) : timestamp;
	const date = new Date(ts * 1000); // Convert from Unix timestamp
	const now = new Date();
	const diffMs = now - date;
	const diffMins = Math.floor(diffMs / 60000);
	const diffHours = Math.floor(diffMs / 3600000);
	const diffDays = Math.floor(diffMs / 86400000);

	if (diffMins < 1) {
		return 'Just now';
	} else if (diffMins < 60) {
		return `${diffMins} minute${diffMins === 1 ? '' : 's'} ago`;
	} else if (diffHours < 24) {
		return `${diffHours} hour${diffHours === 1 ? '' : 's'} ago`;
	} else if (diffDays < 7) {
		return `${diffDays} day${diffDays === 1 ? '' : 's'} ago`;
	} else {
		return date.toLocaleString();
	}
}

async function loadHistory() {
	try {
		const response = await window.client.getBuildHistory({
			configName: props.configName
		});

		// Process entries and construct build URLs if needed
		history.value = response.entries.map(entry => {
			const processed = { ...entry };

			// If buildUrl is empty but we have outputDir, construct it
			if (!processed.buildUrl && props.outputDir) {
				const l = window.location;
				processed.buildUrl = l.origin + '/' + props.outputDir;
			}

			return processed;
		});
	} catch (error) {
		console.error('Error loading build history:', error);
		history.value = [];
	}
}

onMounted(() => {
	loadHistory();
});

// Reload history when configName changes
watch(() => props.configName, () => {
	loadHistory();
});

// Expose refresh function for parent component
defineExpose({
	refresh: loadHistory
});
</script>
