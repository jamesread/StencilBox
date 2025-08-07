<template id = "build-config-template">
	<section class = "build-config">
		<SectionHeader :title = "'Build Config: ' + config?.name" subtitle = "This shows a build configuration.">
			<template #actions>
				<a href = "https://jamesread.github.io/StencilBox/buildconfigs/index.html" class = "button">
					Open docs
					<HugeiconsIcon :icon = "LinkSquare01Icon" size = "24" />
				</a>
			</template>
		</SectionHeader>

		<dl v-if="config">
			<dt>File Path</dt>
			<dd>{{ config.filename }}</dd>

			<dt>Template</dt>
			<dd>
				<a :href="'/template/' + config.template">{{ config.template }}</a>
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
				<ul v-if = "config.datafiles.length > 0">
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
	</section>

	<section>
		<h2>Build</h2>
		<p v-if="config">Click the button below to build the project.</p>

		<button v-if="config" class = "start-build-button" type = "submit" @click = "startBuild(config)">
			Start Build
			<HugeiconsIcon :icon = "Rocket01Icon" size = "24" />
		</button>

		<dl>
			<dt>Build status</dt>
			<dd :class = "buildClass">{{ buildStatus }}</dd>

			<dt>Build URL</dt>
			<dd>
				<span v-if = "buildUrl">
					<a :href = "buildUrl">{{ buildUrl }}</a>
				</span>
				<span v-else>
					Not available
				</span>
			</dd>
		</dl>
	</section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { LinkSquare01Icon, Rocket01Icon } from '@hugeicons/core-free-icons';

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

		if (response.isError) {
			buildClass.value = 'critical';
		} else {
			buildClass.value = 'good';
		}

		updateBuildUrl(response);
	}

	function updateBuildUrl(response) {
	  let l = window.location;

      buildUrl.value = l.origin + '/' + response.relativePath;
	}
</script>
