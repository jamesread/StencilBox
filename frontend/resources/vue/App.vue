<template>
	<Navigation ref="navigation">
		<Header
			:logoUrl="logo"
			breadcrumbs
			title="StencilBox"
			:username="currentUsername"
			:sidebarEnabled="false"
			@logoClick="goHome"
			@userClick="goAccount"
			>
			<template #toolbar>
				<QuickSearch
					v-if="isLoggedIn"
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
					<p v-if="statusError" class="inline-notification bad">Could not reach the StencilBox server: {{ statusError }}</p>
					<template v-else-if="statusLoaded">
						<LoginForm v-if="!isLoggedIn && appStatus?.databaseEnabled" @login-success="onLogin" />
						<LocalAccessOnly v-else-if="!isLoggedIn" />
						<template v-else>
							<p v-if="appStatus?.adminUsesDefaultPassword" class="inline-notification warning">
								The <strong>admin</strong> account still uses the default password.
								<router-link :to="{ name: 'myAccount' }">Change it now</router-link>.
							</p>
							<router-view />
						</template>
					</template>
				</main>

				<footer>
					<span><a href = "https://jamesread.github.io/StencilBox/">Documentation</a></span>
					<span><a href = "https://github.com/jamesread/StencilBox">GitHub</a></span>
					<span v-if="currentVersion">{{ currentVersion }}</span>
				</footer>
			</div>
		</div>
	</Navigation>
</template>

<script setup>
import { ref, computed, onMounted, nextTick } from 'vue';
import { useRouter } from 'vue-router';

import logo from '../images/logo.png';

import Navigation from 'picocrank/vue/components/Navigation.vue';
import Header from 'picocrank/vue/components/Header.vue';
import QuickSearch from 'picocrank/vue/components/QuickSearch.vue';
import LoginForm from './LoginForm.vue';
import LocalAccessOnly from './LocalAccessOnly.vue';
import { Configuration01Icon } from '@hugeicons/core-free-icons';
import { fetchAppStatus, useAppStatus, errorMessage } from '../../scripts/appStatus.js';
import { canAccessControlPanelFromStatus } from '../../scripts/rbacAccess.js';
import { routeAllowed } from '../../scripts/router.js';

const router = useRouter();
const navigation = ref(null);
const quickSearch = ref(null);
const statusLoaded = ref(false);
const statusError = ref('');
const currentVersion = ref('');

const { appStatus } = useAppStatus();
const isLoggedIn = computed(() => Boolean(appStatus.value?.isLoggedIn));
const currentUsername = computed(() => appStatus.value?.username || '');

function goHome() {
	router.push({ name: 'welcome' });
}

function goAccount() {
	router.push({ name: 'myAccount' });
}

function buildNavigation() {
	const nav = navigation.value;
	if (!nav) {
		return;
	}

	nav.clearNavigationLinks();
	nav.addRouterLink('welcome');
	nav.addRouterLink('buildConfigList');
	nav.addRouterLink('templateList');
	nav.addRouterLink('dataFileList');
	nav.addRouterLink('systemDetails');

	if (canAccessControlPanelFromStatus(appStatus.value)) {
		nav.addRouterLink('controlPanel');
	}
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

// startSession runs once the user is logged in: Init and everything after it need app.access.
async function startSession() {
	buildNavigation();
	await nextTick();
	importNavigationIntoSearch();

	try {
		const init = await window.client.init({});
		currentVersion.value = 'Version: ' + init.version;
		importSearchHints(init.searchHints);
	} catch (e) {
		console.error('Init failed:', e);
	}
}

async function onLogin() {
	await fetchAppStatus({ force: true });
	await startSession();
	// The route was entered before permissions were known; leave it if it is not allowed.
	if (!routeAllowed(router.currentRoute.value, appStatus.value)) {
		await router.replace({ name: 'welcome' });
	}
}

onMounted(async () => {
	try {
		await fetchAppStatus();
	} catch (e) {
		statusError.value = errorMessage(e);
		return;
	} finally {
		statusLoaded.value = true;
	}

	if (isLoggedIn.value) {
		await startSession();
	}
});

</script>
