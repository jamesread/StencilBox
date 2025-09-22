<template>
	<Header
		:logoUrl="logo"
		breadcrumbs
		title="StencilBox"
		@toggleSidebar="sidebar.toggle()"
		/>

	<div id="layout">
		<Sidebar ref="sidebar" />

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

import logo from '../images/logo.png';

import Breadcrumbs from 'picocrank/vue/components/Breadcrumbs.vue';
import Sidebar from 'picocrank/vue/components/Sidebar.vue';
import Header from 'picocrank/vue/components/Header.vue';

const sidebar = ref(null);

onMounted(() => {
    sidebar.value.addRouterLink('welcome');
	sidebar.value.addRouterLink('buildConfigList');
	sidebar.value.addRouterLink('templateList');
	sidebar.value.addRouterLink('systemDetails');
	sidebar.value.toggle();
	sidebar.value.stick();
});

</script>
