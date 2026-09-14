<template id = "build-config-template">
	<Section
		:title = "'Build Config: ' + config?.name"
		:icon = "Configuration01Icon"
		subtitle = "This shows a build configuration."
		>

			<template #toolbar>
				<a href = "https://jamesread.github.io/StencilBox/buildconfigs/index.html" class = "button inline-icon">
					<HugeiconsIcon :icon = "LinkSquare01Icon" width = "1em" height = "1em" :strokeWidth = "2.5" aria-hidden = "true" />
					<span>Open docs</span>
				</a>
			</template>

		<dl v-if="config">
			<dt>File name</dt>
			<dd>{{ config.filename }}</dd>

			<dt>File path</dt>
			<dd>
				{{ config.path }}
				{{ config.inContainer ? '(container volume)' : '(on host)' }}
			</dd>

			<dt>Template</dt>
			<dd>
				<router-link :to ="'/template/' +config.template" class = "link">
					{{ config.template }}
				</router-link>
			</dd>

			<dt>Output directory</dt>
			<dd>
				<span v-if = "!config.outputDir" class = "subtle">N/A</span>
				<span v-else>{{ config.outputDir }}</span>
			</dd>

			<dt>Repos</dt>
			<dd>
				<span v-if = "config.repos.length == 0" class = "subtle">No repos defined</span>
				<ul v-else>
					<li v-for="repo in config.repos" :key="repo">
						<a :href="repo" target="_blank">{{ repo }}</a>
					</li>
				</ul>
			</dd>

			<dt>
				<abbr title = "YAML files that provide data to the template during the build process.">
				   Data Files
				</abbr>
			</dt>
			<dd>
				<p>
					{{ config.datafilesPath }}
					<span v-if = "config.datafilesPathInContainer">(container volume)</span>
					<span v-else>(on host)</span>
				</p>
				<ul v-if = "Object.keys(config.datafiles).length > 0">
					<li v-for="(path, name) in config.datafiles" :key="name">
						<router-link :to="{ name: 'dataFileView', params: { buildConfigName: config.name, datafileName: name } }" class="link">
							{{ name }}
						</router-link>
						<span class="subtle"> ({{ path }})</span>
					</li>
				</ul>
				<span v-else class = "subtle">No datafiles defined</span>
			</dd>
		</dl>

		<p>
			All this information comes from your build config file.
		</p>
	</Section>

	<Section title = "Build" id = "build" :padding = "false" :icon = "Rocket01Icon">
		<div class = "padding">
			<p v-if="config">Click the button below to build the project.</p>

			<div v-if="config" class="build-actions">
				<button class="inline-icon good" type="button" @click="startBuild">
					<HugeiconsIcon :icon="Rocket01Icon" width="1em" height="1em" :strokeWidth="2.5" aria-hidden="true" />
					<span>Start Build</span>
				</button>
				<button class="inline-icon neutral" type="button" :disabled="isClearingCache" @click="clearCache">
					<HugeiconsIcon :icon="Delete02Icon" width="1em" height="1em" :strokeWidth="2.5" aria-hidden="true" />
					<span>{{ isClearingCache ? 'Clearing…' : 'Clear cache' }}</span>
				</button>
			</div>
		</div>

		<div v-if="config" class="section-subheader">
			<h3>Build output</h3>
		</div>

		<div v-if="config" class="padding">
			<ReadOnlyTextArea
				ref="buildLogArea"
				v-model="buildLogText"
				placeholder="No build output yet. Start a build to stream status here."
				:rows="16"
			>
				<template #actions>
					<button type="button" :disabled="!buildLogText" @click="clearBuildLog">
						Clear log
					</button>
				</template>
			</ReadOnlyTextArea>

			<dl v-if = "lastBuildUpdate && lastBuildUpdate.isComplete">
				<dt>Output directory</dt>
				<dd>
					<span v-if = "lastBuildUpdate.baseOutputDir">
						{{ lastBuildUpdate.baseOutputDir }}
						<span v-if = "lastBuildUpdate.inContainer">(container volume)</span>
						<span v-else>(on host)</span>
					</span>
					<span v-else class = "subtle">Not available</span>
				</dd>

				<dt>Output size</dt>
				<dd>
					{{ lastBuildUpdate.outputSizeHumanReadable }}
				</dd>

				<dt>Build URL</dt>
				<dd>
					<span v-if = "lastBuildUpdate.buildUrl">
						<a :href = "lastBuildUpdate.buildUrl">{{ lastBuildUpdate.buildUrl }}</a>
						(<a href = "https://jamesread.github.io/StencilBox/config/build_urls.html">Docs</a>)
					</span>
					<span v-else class = "subtle">
						Not available
					</span>
				</dd>
			</dl>
		</div>
	</Section>

	<BuildHistory
		v-if="config"
		:configName="config.name"
		:outputDir="config.outputDir"
		ref="historyComponent"
	/>
</template>

<script setup>
	import { ref, onMounted, nextTick, watch } from 'vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { Configuration01Icon, Delete02Icon, LinkSquare01Icon, Rocket01Icon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import ReadOnlyTextArea from 'picocrank/vue/components/ReadOnlyTextArea.vue';
	import BuildHistory from './BuildHistory.vue';

	const props = defineProps({
		name: {
			type: String,
			required: true
		}
	});

	const MAX_BUILD_LOG_LINES = 2000;
	const BUILD_LOG_STORAGE_PREFIX = 'StencilBox.buildLog.';

	const config = ref(null);
	const historyComponent = ref(null);
	const buildLogArea = ref(null);
	const buildLogText = ref('');
	let buildSessionStartMs = 0;

	const lastBuildUpdate = ref(null);
	const isClearingCache = ref(false);

	function buildLogStorageKey(configName) {
		return BUILD_LOG_STORAGE_PREFIX + configName;
	}

	function pad2(n) {
		return String(n).padStart(2, '0');
	}

	function formatClock(d) {
		return `${pad2(d.getHours())}:${pad2(d.getMinutes())}:${pad2(d.getSeconds())}.${String(d.getMilliseconds()).padStart(3, '0')}`;
	}

	function formatElapsed(ms) {
		if (!Number.isFinite(ms) || ms < 0) {
			return '+0.000s';
		}
		const sec = ms / 1000;
		if (sec < 60) {
			return `+${sec.toFixed(3)}s`;
		}
		const m = Math.floor(sec / 60);
		const s = sec - m * 60;
		return `+${m}m ${s.toFixed(1)}s`;
	}

	function scrollBuildLogToBottom() {
		nextTick(() => {
			const el = buildLogArea.value?.$el?.querySelector?.('textarea');
			if (el) {
				el.scrollTop = el.scrollHeight;
			}
		});
	}

	function persistBuildLog() {
		const name = config.value?.name;
		if (!name || typeof sessionStorage === 'undefined') {
			return;
		}
		try {
			sessionStorage.setItem(buildLogStorageKey(name), JSON.stringify(buildLogText.value));
		} catch (e) {
			console.warn('Could not persist build log:', e);
		}
	}

	function storedLogToText(parsed) {
		if (typeof parsed === 'string') {
			return parsed;
		}
		if (!Array.isArray(parsed)) {
			return '';
		}
		return parsed
			.map((row) => `[${row.clock ?? ''}] ${row.elapsed ?? '+0.000s'} ${row.text ?? ''}`)
			.join('\n');
	}

	function loadBuildLogFromStorage(configName) {
		buildLogText.value = '';
		if (!configName || typeof sessionStorage === 'undefined') {
			return;
		}
		try {
			const raw = sessionStorage.getItem(buildLogStorageKey(configName));
			if (!raw) {
				return;
			}
			buildLogText.value = storedLogToText(JSON.parse(raw));
			scrollBuildLogToBottom();
		} catch (e) {
			console.warn('Could not load build log:', e);
			buildLogText.value = '';
		}
	}

	function trimBuildLog() {
		const lines = buildLogText.value.split('\n');
		if (lines.length <= MAX_BUILD_LOG_LINES) {
			return;
		}
		buildLogText.value = lines.slice(lines.length - MAX_BUILD_LOG_LINES).join('\n');
	}

	function appendBuildLogLine(text) {
		const now = new Date();
		const elapsedMs = buildSessionStartMs ? performance.now() - buildSessionStartMs : 0;
		const line = `[${formatClock(now)}] ${formatElapsed(elapsedMs)} ${text}`;
		buildLogText.value = buildLogText.value ? `${buildLogText.value}\n${line}` : line;
		trimBuildLog();
		persistBuildLog();
		scrollBuildLogToBottom();
	}

	function clearBuildLog() {
		buildLogText.value = '';
		buildLogArea.value?.clear?.();
		const name = config.value?.name;
		if (name && typeof sessionStorage !== 'undefined') {
			try {
				sessionStorage.removeItem(buildLogStorageKey(name));
			} catch (e) {
				console.warn('Could not clear persisted build log:', e);
			}
		}
	}

	async function loadConfig() {
		try {
			const response = await window.client.getBuildConfig({
				configName: props.name
			});
			config.value = response.buildConfig;
			loadBuildLogFromStorage(config.value.name);
		} catch (error) {
			console.error('Error loading build config:', error);
		}
	}

	onMounted(() => {
		loadConfig();
	});

	watch(
		() => props.name,
		() => {
			loadConfig();
		}
	);

	async function clearCache() {
		if (!config.value || isClearingCache.value) {
			return;
		}
		isClearingCache.value = true;
		try {
			const response = await window.client.clearBuildCache({
				configName: config.value.name
			});
			appendBuildLogLine(response.message + (response.cachePath ? ` (${response.cachePath})` : ''));
		} catch (error) {
			appendBuildLogLine('Error clearing cache: ' + (error && error.message ? error.message : String(error)));
		} finally {
			isClearingCache.value = false;
		}
	}

	async function startBuild() {
		if (!config.value) {
			return;
		}
		clearBuildLog();
		buildSessionStartMs = performance.now();
		appendBuildLogLine(`── Build started: ${config.value.name} ──`);
		try {
			for await (const update of window.client.startBuild({ configName: config.value.name })) {
				onBuildUpdate(update);
			}
			appendBuildLogLine('── Build stream finished ──');
		} catch (error) {
			const msg = 'Error starting build: ' + (error && error.message ? error.message : String(error));
			appendBuildLogLine(msg);
			lastBuildUpdate.value = {
				status: msg,
				isError: true,
				isComplete: false,
				cssClass: 'critical'
			};
		}
	}

	function onBuildUpdate(update) {
		update.cssClass = update.isError ? 'critical' : 'good';

		lastBuildUpdate.value = update;

		appendBuildLogLine(update.status);

		console.log('Build update:', update);
		updateBuildUrl();

		if (update.isComplete && historyComponent.value) {
			historyComponent.value.refresh();
		}
	}

	function updateBuildUrl() {
		if (!lastBuildUpdate.value) {
			return;
		}
		if (lastBuildUpdate.value.buildUrlBase === '') {
			const l = window.location;
			lastBuildUpdate.value.buildUrl = l.origin + '/' + lastBuildUpdate.value.relativePath;
		} else {
			lastBuildUpdate.value.buildUrl = lastBuildUpdate.value.buildUrlBase + '/' + lastBuildUpdate.value.relativePath;
		}
	}
</script>

<style scoped>
.build-actions {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: 0.75rem;
}
</style>
