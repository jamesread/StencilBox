<template>
	<Navigation ref="navigation">
		<Header
			:logoUrl="logo"
			breadcrumbs
			title="StencilBox"
			:username="currentUsername"
			:sidebarEnabled="false"
			@logoClick="goHome"
			/>

		<div id="layout">
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
import { useRouter } from 'vue-router';

import logo from '../images/logo.png';

import Navigation from 'picocrank/vue/components/Navigation.vue';
import Header from 'picocrank/vue/components/Header.vue';

const router = useRouter();
const navigation = ref(null);
const currentUsername = ref('');

function goHome() {
	router.push({ name: 'welcome' });
}

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
	navigation.value.addRouterLink('welcome');
	navigation.value.addRouterLink('buildConfigList');
	navigation.value.addRouterLink('templateList');
	navigation.value.addRouterLink('dataFileList');
	navigation.value.addRouterLink('systemDetails');

	loadCurrentUser();
});

</script>
