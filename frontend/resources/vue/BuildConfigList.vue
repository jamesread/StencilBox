<template>
	<section>
		<SectionHeader title="Build Configurations" subtitle="This is a list of build configurations, both builtin, and from your config directory." />

		<table>
			<thead>
				<tr>
					<th>Name</th>
					<th>Template</th>
					<th class = "small">Actions</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="config in buildConfigs" :key="config.name">
					<td>{{ config.name }}</td>
					<td>{{ config.template }}</td>
					<td>
						<span :class="'small ' + config.statusClass">{{ config.status }}</span>
						<a href = "#" @click.prevent = "openBuild(config)">Open</a>
					</td>
				</tr>
			</tbody>
		</table>
	</section>
</template>

<script setup>
	import { ref, onMounted, inject } from 'vue';

	const buildConfigs = ref([]);
	const changeSection = inject('changeSection');

	function openBuild(config) {
		changeSection('buildConfig', config);
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

	onMounted(() => {
		getBuildConfigs();
	});
</script>
