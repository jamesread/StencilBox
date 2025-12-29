<template>
	<Navigation ref="navigation">
		<Header
			:logoUrl="logo"
			breadcrumbs
			title="StencilBox"
			:username="currentUsername"
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
	</Navigation>
</template>

<script setup>
import { ref, onMounted } from 'vue';

import logo from '../images/logo.png';

import Breadcrumbs from 'picocrank/vue/components/Breadcrumbs.vue';
import Navigation from 'picocrank/vue/components/Navigation.vue';
import Sidebar from 'picocrank/vue/components/Sidebar.vue';
import Header from 'picocrank/vue/components/Header.vue';

const navigation = ref(null);
const sidebar = ref(null);
const currentUsername = ref('');

async function loadCurrentUser() {
	try {
		const response = await window.client.getCurrentUser({});
		if (response.isAuthenticated && response.username) {
			currentUsername.value = response.username;
		}
	} catch (error) {
		// If auth is not enabled or user is not authenticated, username will remain empty
		console.debug('Could not load current user:', error);
	}
}

onMounted(() => {
	// Add links to Navigation component instead of Sidebar
	navigation.value.addRouterLink('welcome');
	navigation.value.addRouterLink('buildConfigList');
	navigation.value.addRouterLink('templateList');
	navigation.value.addRouterLink('dataFileList');
	navigation.value.addRouterLink('systemDetails');

	// Sidebar will automatically read from Navigation component
	sidebar.value.toggle();
	sidebar.value.stick();

	// Load current user info
	loadCurrentUser();
});

</script>
