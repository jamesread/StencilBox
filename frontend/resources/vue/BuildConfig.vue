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

		<table v-if="config">
			<tbody>
				<tr>
					<th>Template</th>
					<td>{{ config.template }}</td>
				</tr>
				<tr>
					<th>Status</th>
					<td :class = "buildClass">{{ buildStatus }}</td>
				</tr>
				<tr>
					<th>URL</th>
					<td>
						<span v-if = "buildUrl">
							<a :href = "buildUrl">LINK</a>
						</span>
						<span v-else>
							Not available
						</span>
					</td>
				</tr>
			</tbody>
		</table>

		<p v-if="config">Click the button below to build the project.</p>

		<button v-if="config" class = "start-build-button" type = "submit" @click = "startBuild(config)">Build</button>
	</section>
</template>

<script setup>
	import { ref, defineProps } from 'vue';

	const props = defineProps({
		config: {
			type: Object,
			required: false,
			default: null
		}
	});

	const config = ref(props.config);

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
