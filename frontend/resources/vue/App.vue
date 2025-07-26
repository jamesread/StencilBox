<template>
	<header>
		<div class="flex-row">
			<img src="../images/logo.png" class="logo" />
			<h1>StencilBox</h1>

			<Breadcrumbs />
		</div>

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

const sectionNavigation = ref(null);

onMounted(() => {
	sectionNavigation.value.addLink('Welcome', '/');
	sectionNavigation.value.addLink('Templates', '/templates');
	sectionNavigation.value.addLink('Build Configs', '/build-configs');
	sectionNavigation.value.addLink('System Details', '/system');
});

</script>
