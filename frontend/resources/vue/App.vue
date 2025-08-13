<template>
	<header>
		<div class="flex-row" id = "sidebar-button">
			<img src="../images/logo.png" class="logo" />
			<h1>StencilBox</h1>
		</div>

		<Breadcrumbs />
	</header>

	<div id="layout">
		<aside class="shown stuck" id = "sidebar">
			<SectionNavigation ref="sectionNavigation" />
		</aside>

		<div id = "content">
			<main>
				<router-view />
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
import { ref, onMounted } from 'vue';
import {
	Notification03Icon,
	CodeSquareIcon,
	Settings01Icon,
	Configuration01Icon
} from '@hugeicons/core-free-icons';

import Breadcrumbs from 'picocrank/vue/components/Breadcrumbs.vue';

const sectionNavigation = ref(null);

onMounted(() => {
	sectionNavigation.value.addLink('Welcome', '/', Notification03Icon);
	sectionNavigation.value.addLink('Build Configs', '/build-configs', Configuration01Icon);
	sectionNavigation.value.addLink('Templates', '/templates', CodeSquareIcon);
	sectionNavigation.value.addLink('System Details', '/system', Settings01Icon);
});

</script>
