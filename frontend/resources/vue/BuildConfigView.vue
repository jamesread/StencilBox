<template id = "build-config-template">
	<section class = "build-config">
		<div v-if="config">
			<h2>Build Config: {{ config.name }}</h2>

			<p>This shows the last build.</p>
		</div>
		<div v-else>
			<h2>Build Config</h2>
			<p>No build configuration selected.</p>
		</div>


		<dl v-if="config">
			<dt>Template</dt>
			<dd>
				<a :href="'/template/' + config.template">{{ config.template }}</a>
			</dd>

			<dt>Build status</dt>
			<dd :class = "buildClass">{{ buildStatus }}</dd>

			<dt>Build URL</dt>
			<dd>
				<span v-if = "buildUrl">
					<a :href = "buildUrl">LINK</a>
				</span>
				<span v-else>
					Not available
				</span>
			</dd>
		</dl>

		<p v-if="config">Click the button below to build the project.</p>

		<button v-if="config" class = "start-build-button" type = "submit" @click = "startBuild(config)">Build</button>
	</section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';

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
