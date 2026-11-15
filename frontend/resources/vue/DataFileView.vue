<template>
	<Section
		:title="'Data File: ' + (dataFile?.name || 'Loading...')"
		:icon="File01Icon"
		:subtitle="canEdit ? 'Edit the YAML that this build config reads. Saving rebuilds every site that uses the file.' : 'View the contents of a data file.'"
	>
		<template #toolbar>
			<a href="https://jamesread.github.io/StencilBox/buildconfigs/index.html" class="button inline-icon">
				<HugeiconsIcon :icon="LinkSquare01Icon" width="1em" height="1em" :strokeWidth="2.5" aria-hidden="true" />
				<span>Docs</span>
			</a>
		</template>

		<dl v-if="dataFile">
			<dt>Name</dt>
			<dd><code>{{ dataFile.name }}</code></dd>

			<dt>Build Config</dt>
			<dd>
				<router-link :to="{ name: 'buildConfig', params: { name: dataFile.buildConfigName } }" class="link">
					{{ dataFile.buildConfigName }}
				</router-link>
			</dd>

			<dt>Path</dt>
			<dd><code>{{ dataFile.path }}</code></dd>
		</dl>

		<p v-if="!dataFile && !error" class="subtle">Loading...</p>
		<NotificationBlock v-if="error" type="error" :message="error" />
	</Section>

	<Section v-if="dataFile" title="Content" :icon="SourceCodeIcon">
		<template v-if="canEdit" #toolbar>
			<span v-if="dirty" class="tag warning">unsaved changes</span>
			<button type="button" class="neutral" :disabled="!dirty || saving" @click="revert">Revert</button>
			<button type="button" class="good" :disabled="!dirty || !valid || saving" @click="save">
				{{ saving ? 'Saving…' : 'Save' }}
			</button>
		</template>

		<YamlEditor
			v-model="content"
			:readonly="!canEdit"
			:rows="24"
			aria-label="Data file content"
			@validity="valid = $event"
		/>

		<NotificationBlock v-if="saveError" type="error" :message="saveError" />
		<NotificationBlock v-if="saveMessage" type="success" :message="saveMessage" />
		<p v-if="conflict" class="inline-notification warning">
			Someone else changed this file after you opened it.
			<button type="button" class="neutral small" @click="reloadDiscarding">Load their version</button>
			(your edits are discarded), or copy your edits somewhere first.
		</p>
	</Section>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue';
import { onBeforeRouteLeave } from 'vue-router';
import { Code } from '@connectrpc/connect';
import { HugeiconsIcon } from '@hugeicons/vue';
import { File01Icon, LinkSquare01Icon, SourceCodeIcon } from '@hugeicons/core-free-icons';
import Section from 'picocrank/vue/components/Section.vue';
import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
import YamlEditor from './YamlEditor.vue';
import { useAppStatus, errorMessage } from '../../scripts/appStatus.js';

const props = defineProps({
	buildConfigName: { type: String, required: true },
	datafileName: { type: String, required: true },
});

const { hasPermission } = useAppStatus();
const canEdit = computed(() => hasPermission('datafiles.edit'));

const dataFile = ref(null);
const content = ref('');
const savedContent = ref('');
const sha256 = ref('');
const error = ref('');
const valid = ref(true);
const saving = ref(false);
const saveError = ref('');
const saveMessage = ref('');
const conflict = ref(false);

const dirty = computed(() => content.value !== savedContent.value);

async function loadDataFile() {
	error.value = '';
	try {
		const response = await window.client.getDataFile({
			buildConfigName: props.buildConfigName,
			datafileName: props.datafileName,
		});
		dataFile.value = {
			name: response.datafileName,
			path: response.path,
			buildConfigName: response.buildConfigName,
		};
		content.value = response.content;
		savedContent.value = response.content;
		sha256.value = response.sha256;
	} catch (err) {
		console.error('Error loading data file:', err);
		error.value = errorMessage(err, 'Failed to load data file');
	}
}

async function save() {
	saving.value = true;
	saveError.value = '';
	saveMessage.value = '';
	conflict.value = false;
	try {
		const res = await window.client.updateDataFile({
			buildConfigName: props.buildConfigName,
			datafileName: props.datafileName,
			content: content.value,
			expectedSha256: sha256.value,
		});
		savedContent.value = content.value;
		sha256.value = res.sha256;
		const sites = res.rebuildingBuildConfigs || [];
		saveMessage.value = sites.length ? `Saved. Rebuilding: ${sites.join(', ')}.` : 'Saved.';
	} catch (err) {
		if (err?.code === Code.FailedPrecondition) { // the file changed since it was loaded
			conflict.value = true;
		} else {
			saveError.value = errorMessage(err, 'Failed to save the data file');
		}
	} finally {
		saving.value = false;
	}
}

function revert() {
	content.value = savedContent.value;
	saveError.value = '';
	saveMessage.value = '';
}

async function reloadDiscarding() {
	conflict.value = false;
	await loadDataFile();
}

function confirmLeave() {
	return !dirty.value || confirm('You have unsaved changes to this data file. Leave anyway?');
}

function beforeUnload(event) {
	if (dirty.value) {
		event.preventDefault();
	}
}

onBeforeRouteLeave(() => confirmLeave());
onMounted(() => {
	loadDataFile();
	window.addEventListener('beforeunload', beforeUnload);
});
onBeforeUnmount(() => window.removeEventListener('beforeunload', beforeUnload));
</script>
