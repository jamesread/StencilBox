<template>
	<section>
		<SectionHeader title="Build Configurations" subtitle="This is a list of build configurations, both builtin, and from your config directory." />

		<table>
			<thead>
				<tr>
					<th>Name</th>
					<th>Template</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="config in buildConfigs" :key="config.name">
					<td>
						<a href="#" @click.prevent="openBuild(config)">{{ config.name }}</a>
					</td>
					<td>
						<a href="#" @click.prevent="openTemplate(config)">{{ config.template }}</a>
					</td>
				</tr>
			</tbody>
		</table>
	</section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { useRouter } from 'vue-router';

	const buildConfigs = ref([]);
	const router = useRouter();

	function openBuild(config) {
		router.push({ name: 'buildConfig', params: { name: config.name } });
	}

	function openTemplate(config) {
		router.push({ name: 'templateView', params: { name: config.template } });
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
