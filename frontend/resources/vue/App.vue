<template>
	<header>
		<div class="flex-row" id = "sidebar-button">
			<img src="../images/logo.png" class="logo" />
			<h1>StencilBox</h1>
		</div>

		<Breadcrumbs />
	</header>

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
import {
	Notification03Icon,
	CodeSquareIcon,
	Settings01Icon,
	Configuration01Icon
} from '@hugeicons/core-free-icons';

import Breadcrumbs from 'picocrank/vue/components/Breadcrumbs.vue';
import Sidebar from 'picocrank/vue/components/Sidebar.vue';

const sidebar = ref(null);

onMounted(() => {
	sidebar.value.addNavigationLink({ title: 'Welcome', id: 'welcome', path: '/', icon: Notification03Icon });
	sidebar.value.addNavigationLink({ title: 'Build Configs', id: 'buildConfigs', path: '/build-configs', icon: Configuration01Icon });
	sidebar.value.addNavigationLink({ title: 'Templates', id: 'templates', path: '/templates', icon: CodeSquareIcon });
	sidebar.value.addNavigationLink({ title: 'System Details', id: 'system', path: '/system', icon: Settings01Icon });
	sidebar.value.toggle();
	sidebar.value.stick();
});

</script>
