<template id = "build-config-template">
	<Section
		:title = "'Build Config: ' + config?.name"
		subtitle = "This shows a build configuration."
		>

			<template #toolbar>
				<a href = "https://jamesread.github.io/StencilBox/buildconfigs/index.html" class = "button">
					Open docs
					<HugeiconsIcon :icon = "LinkSquare01Icon" size = "24" />
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

	<Section title = "Build" id = "build">
		<p v-if="config">Click the button below to build the project.</p>

		<button v-if="config" class = "start-build-button" type = "submit" @click = "startBuild(config)">
			Start Build
			<HugeiconsIcon :icon = "Rocket01Icon" size = "24" />
		</button>

		<div v-if="config" class="build-log-panel">
			<div class="build-log-toolbar">
				<span class="subtle">Build output — newest at bottom; scroll for full history (saved for this browser session).</span>
				<button type="button" class="button build-log-clear" @click="clearBuildLog">
					Clear log
				</button>
			</div>
			<div
				ref="buildLogEl"
				class="build-log"
				role="log"
				aria-relevant="additions"
				aria-live="polite"
			>
				<p v-if="buildLogLines.length === 0" class="subtle build-log-empty">No build output yet. Start a build to stream status here.</p>
				<div
					v-for="line in buildLogLines"
					:key="line.id"
					class="build-log-line"
					:class="{
						'build-log-line--banner': line.kind === 'banner',
						'build-log-line--error': line.kind === 'error',
						'build-log-line--ok': line.kind === 'ok'
					}"
				>
					<span class="build-log-meta">[{{ line.clock }}] {{ line.elapsed }}</span>
					<span class="build-log-text">{{ line.text }}</span>
				</div>
			</div>
		</div>

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
	import { LinkSquare01Icon, Rocket01Icon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
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
	const buildLogEl = ref(null);
	const buildLogLines = ref([]);
	let buildSessionStartMs = 0;
	let logIdSeq = 0;

	const lastBuildUpdate = ref(null);

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
			const el = buildLogEl.value;
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
			sessionStorage.setItem(buildLogStorageKey(name), JSON.stringify(buildLogLines.value));
		} catch (e) {
			console.warn('Could not persist build log:', e);
		}
	}

	function loadBuildLogFromStorage(configName) {
		buildLogLines.value = [];
		if (!configName || typeof sessionStorage === 'undefined') {
			return;
		}
		try {
			const raw = sessionStorage.getItem(buildLogStorageKey(configName));
			if (!raw) {
				return;
			}
			const parsed = JSON.parse(raw);
			if (!Array.isArray(parsed)) {
				return;
			}
			logIdSeq = 0;
			buildLogLines.value = parsed.map((row) => ({
				id: row.id ?? ++logIdSeq,
				clock: row.clock ?? '',
				elapsed: row.elapsed ?? '+0.000s',
				text: row.text ?? '',
				kind: row.kind === 'banner' || row.kind === 'error' || row.kind === 'ok' ? row.kind : 'ok'
			}));
			logIdSeq = buildLogLines.value.reduce((m, l) => (typeof l.id === 'number' ? Math.max(m, l.id) : m), 0);
			scrollBuildLogToBottom();
		} catch (e) {
			console.warn('Could not load build log:', e);
			buildLogLines.value = [];
			logIdSeq = 0;
		}
	}

	function appendBuildLogLine(text, kind) {
		const now = new Date();
		const elapsedMs = buildSessionStartMs ? performance.now() - buildSessionStartMs : 0;
		const line = {
			id: ++logIdSeq,
			clock: formatClock(now),
			elapsed: formatElapsed(elapsedMs),
			text,
			kind
		};
		buildLogLines.value.push(line);
		if (buildLogLines.value.length > MAX_BUILD_LOG_LINES) {
			const overflow = buildLogLines.value.length - MAX_BUILD_LOG_LINES;
			buildLogLines.value.splice(0, overflow);
		}
		persistBuildLog();
		scrollBuildLogToBottom();
	}

	function clearBuildLog() {
		buildLogLines.value = [];
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

	async function startBuild() {
		if (!config.value) {
			return;
		}
		buildSessionStartMs = performance.now();
		appendBuildLogLine(`── Build started: ${config.value.name} ──`, 'banner');
		try {
			for await (const update of window.client.startBuild({ configName: config.value.name })) {
				onBuildUpdate(update);
			}
			appendBuildLogLine('── Build stream finished ──', 'banner');
		} catch (error) {
			const msg = 'Error starting build: ' + (error && error.message ? error.message : String(error));
			appendBuildLogLine(msg, 'error');
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

		appendBuildLogLine(update.status, update.isError ? 'error' : 'ok');

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
.build-log-panel {
	margin-top: 1rem;
}

.build-log-toolbar {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	justify-content: space-between;
	gap: 0.5rem 1rem;
	margin-bottom: 0.35rem;
}

.build-log-clear {
	font-size: 0.9em;
}

.build-log {
	max-height: 22rem;
	overflow-y: auto;
	padding: 0.65rem 0.75rem;
	border-radius: 6px;
	font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
	font-size: 0.85rem;
	line-height: 1.45;
	white-space: pre-wrap;
	word-break: break-word;
	background: color-mix(in srgb, Canvas 92%, CanvasText 8%);
	border: 1px solid color-mix(in srgb, CanvasText 12%, transparent);
}

.build-log-empty {
	margin: 0;
	font-family: inherit;
	font-size: inherit;
}

.build-log-line {
	margin: 0;
	padding: 0.1rem 0;
}

.build-log-meta {
	display: inline-block;
	min-width: 13.5rem;
	margin-right: 0.5rem;
	opacity: 0.85;
	user-select: none;
}

.build-log-line--banner .build-log-text {
	opacity: 0.75;
	font-style: italic;
}

.build-log-line--error .build-log-text {
	color: var(--color-critical, #c62828);
}

.build-log-line--ok .build-log-text {
	color: inherit;
}
</style>
