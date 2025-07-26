<template>
	<header>
		<div class="flex-row">
			<img src="../images/logo.png" class="logo" />
			<h1>StencilBox</h1>

			<Breadcrumbs ref = "breadcrumbComp" />
		</div>

	</header>

	<div id="layout">
		<aside class="shown stuck" id = "sidebar">
			<SectionNavigation ref="sectionNavigation" />
		</aside>

		<div id = "content">
			<main>
				<KeepAlive>
					<component :is="currentSection" :config="selectedConfig" />
				</KeepAlive>
			</main>

			<footer>
				<span><a href = "https://jamesread.github.io/StencilBox/">Documentation</a></span>
				<span><a href = "https://github.com/jamesread/StencilBox">GitHub</a></span>
				<span id = "current-version">?</span>
			</footer>
		</div>
	</div>
</template>

<script setup>
import { ref, computed, onMounted, provide } from 'vue';

import Welcome from './Welcome.vue';
import BuildConfigList from './BuildConfigList.vue';
import BuildConfig from './BuildConfig.vue';
import TemplateList from './TemplateList.vue';
import SystemDetails from './SystemDetails.vue';

const breadcrumbComp = ref(null);

const sections = {
	"welcome": Welcome,
	"buildConfigList": BuildConfigList,
	"buildConfig": BuildConfig,
	"templates": TemplateList,
	"systemDetails": SystemDetails
};

const currentSectionName = ref('welcome');
const currentSection = computed(() => sections[currentSectionName.value]);
const selectedConfig = ref(null);

const sectionNavigation = ref(null);

onMounted(() => {
	sectionNavigation.value.addLink('Welcome', 'welcome');
	sectionNavigation.value.addLink('Templates', 'templates');
	sectionNavigation.value.addLink('Build Configs', 'buildConfigList');
	sectionNavigation.value.addLink('System Details', 'systemDetails');
});

provide('changeSection', (sectionName, config = null) => {
	console.log(`Changing section to: ${sectionName}`, config);
	if (sections[sectionName]) {
		currentSectionName.value = sectionName;
		if (config) {
			selectedConfig.value = config;
		}
	} else {
		console.warn(`Section "${sectionName}" does not exist.`);
	}
});
</script>
