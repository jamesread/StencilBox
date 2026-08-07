<template>
	<Navigation ref="navigation">
		<Header
			:logoUrl="logo"
			breadcrumbs
			title="StencilBox"
			:username="currentUsername"
			:sidebarEnabled="false"
			@logoClick="goHome"
			>
			<template #toolbar>
				<QuickSearch
					ref="quickSearch"
					:auto-import-routes="false"
					:search-fields="['title', 'description', 'category']"
					:max-results="15"
					placeholder="Search pages and build configs…"
				/>
			</template>
		</Header>

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
import QuickSearch from 'picocrank/vue/components/QuickSearch.vue';
import { Configuration01Icon } from '@hugeicons/core-free-icons';

const router = useRouter();
const navigation = ref(null);
const quickSearch = ref(null);
const currentUsername = ref('');

function goHome() {
	router.push({ name: 'welcome' });
}

function importNavigationIntoSearch() {
	if (!quickSearch.value || !navigation.value) {
		return;
	}

	for (const link of navigation.value.getNavigationLinks()) {
		if (link.type !== 'route') {
			continue;
		}

		quickSearch.value.addItem({
			id: `nav-${link.name}`,
			title: link.title,
			description: link.description || '',
			category: 'Navigation',
			type: 'route',
			path: link.path,
			icon: link.icon,
		});
	}
}

function importSearchHints(searchHints) {
	if (!quickSearch.value || !searchHints?.buildConfigs) {
		return;
	}

	for (const name of searchHints.buildConfigs) {
		quickSearch.value.addItem({
			id: `build-config-${name}`,
			title: name,
			description: 'Build configuration',
			category: 'Build Configs',
			type: 'route',
			path: `/build-config/${encodeURIComponent(name)}`,
			icon: Configuration01Icon,
		});
	}
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

onMounted(async () => {
	navigation.value.addRouterLink('welcome');
	navigation.value.addRouterLink('buildConfigList');
	navigation.value.addRouterLink('templateList');
	navigation.value.addRouterLink('dataFileList');
	navigation.value.addRouterLink('systemDetails');

	importNavigationIntoSearch();

	const status = await window.client.init();
	document.getElementById('current-version').innerText = 'Version: ' + status.version;
	importSearchHints(status.searchHints);

	loadCurrentUser();
});

</script>
