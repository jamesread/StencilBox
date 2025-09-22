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
					<li v-for="datafile in config.datafiles" :key="datafile">
						<span>{{ datafile }}</span>
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

		<dl v-if = "lastBuildUpdate">
			<dt>Current build status</dt>
			<dd :class = "lastBuildUpdate.cssClass">{{ lastBuildUpdate.status }}</dd>

			<template v-if = "lastBuildUpdate.isComplete">
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
			</template>
		</dl>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { LinkSquare01Icon, Rocket01Icon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';

	const props = defineProps({
		name: {
			type: String,
			required: true
		}
	});

	const config = ref(null);

	async function loadConfig() {
		try {
			const response = await window.client.getBuildConfig({
				configName: props.name
			});
			config.value = response.buildConfig;
		} catch (error) {
			console.error('Error loading build config:', error);
		}
	}

	onMounted(() => {
		loadConfig();
	});

    const lastBuildUpdate = ref(null);

	async function startBuild() {
	  try {
		  for await (const update of window.client.startBuild({ 'configName': config.value.name })) {
			onBuildUpdate(update);
		  }
	  }  catch (error) {
		  lastBuildUpdate.value = {
			  status: 'Error starting build: ' + error.message,
			  isError: true,
			  isComplete: false,
			  cssClass: 'critical'
		  };
	  }
	}

	function onBuildUpdate(update) {
	    update.cssClass = update.isError ? 'critical' : 'good';

	    lastBuildUpdate.value = update;

		console.log('Build update:', update);
		updateBuildUrl();
	}

	function updateBuildUrl() {
        if (lastBuildUpdate.value.buildUrlBase == "") {
			let l = window.location;

			lastBuildUpdate.value.buildUrl = l.origin + '/' + lastBuildUpdate.value.relativePath;
		} else {
			lastBuildUpdate.value.buildUrl = lastBuildUpdate.value.buildUrlBase + '/' + lastBuildUpdate.value.relativePath;
		}

	}
</script>
