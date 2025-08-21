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
				Datafiles
			</dt>
			<dd>
				<ul v-if = "Object.keys(config.datafiles).length > 0">
					<li v-for="datafile in config.datafiles" :key="datafile">
						<span>{{ datafile }}</span>
					</li>
				</ul>
				<span v-else class = "subtle">No datafiles defined</span>
			</dd>
		</dl>

		<p>
			This is defined in your build config yaml.
		</p>
	</Section>

	<Section title = "Build">
		<p v-if="config">Click the button below to build the project.</p>

		<button v-if="config" class = "start-build-button" type = "submit" @click = "startBuild(config)">
			Start Build
			<HugeiconsIcon :icon = "Rocket01Icon" size = "24" />
		</button>

		<dl>
			<dt>Build status</dt>
			<dd :class = "buildClass">{{ buildStatus }}</dd>

			<dt>Output directory</dt>
			<dd>
				<span v-if = "outputDirectory">
					{{ outputDirectory }}
					<span v-if = "inContainer">(container volume)</span>
					<span v-else>(on host)</span>
				</span>
				<span v-else class = "subtle">Not available</span>
			</dd>

			<dt>Output size</dt>
			<dd>
				<span v-if = "outputSizeHumanReadable">
					{{ outputSizeHumanReadable }}
				</span>
				<span v-else class = "subtle">Not available</span>
			</dd>

			<dt>Build URL</dt>
			<dd>
				<span v-if = "buildUrl">
					<a :href = "buildUrl">{{ buildUrl }}</a>
					(<a href = "https://jamesread.github.io/StencilBox/config/build_urls.html">Docs</a>)
				</span>
				<span v-else class = "subtle">
					Not available
				</span>
			</dd>
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
		console.log('Build config mounted:', config.value);
		loadConfig();
	});

	const buildStatus = ref('unknown');
	const buildClass = ref('unknown');
	const buildUrl = ref(null);
	const outputSizeHumanReadable = ref(null);
	const outputDirectory = ref(null);
	const inContainer = ref(false);

	async function startBuild() {
	  buildStatus.value = 'Building...';
	  buildClass.value = 'good';

	  console.log('Starting build for config:', config.value.name);

	  const result = await window.client.startBuild({
		'configName': config.value.name,
	  })

      onBuildStarted(result)
	}

	function onBuildStarted(response) {
		console.log('Build started response:', response);

		if (!response.found) {
		    buildStatus.value = 'Build config not found. Please check the configuration.';
			buildClass.value = 'critical';
			return;
		}

		buildStatus.value = response.status;
		outputSizeHumanReadable.value = response.outputSizeHumanReadable;
		outputDirectory.value = response.baseOutputDir;
		inContainer.value = response.inContainer;

		if (response.isError) {
			buildClass.value = 'critical';
		} else {
			buildClass.value = 'good';
		}

		updateBuildUrl(response);
	}

	function updateBuildUrl(response) {
        if (response.buildUrlBase == "") {
			let l = window.location;

			buildUrl.value = l.origin + '/' + response.relativePath;
		} else {
			buildUrl.value = response.buildUrlBase + '/' + response.relativePath;
		}

	}
</script>
