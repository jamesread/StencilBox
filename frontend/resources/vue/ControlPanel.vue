<template>
	<Section
		title="Control Panel"
		subtitle="System administration for StencilBox."
		:icon="DashboardSquare02Icon"
	>
		<Navigation ref="localNavigation">
			<NavigationGrid />
		</Navigation>
	</Section>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import Section from 'picocrank/vue/components/Section.vue'
import Navigation from 'picocrank/vue/components/Navigation.vue'
import NavigationGrid from 'picocrank/vue/components/NavigationGrid.vue'
import { DashboardSquare02Icon, WebSecurityIcon } from '@hugeicons/core-free-icons'
import { useAppStatus } from '../../scripts/appStatus.js'
import { canAccessIamFromStatus } from '../../scripts/rbacAccess.js'

const { appStatus } = useAppStatus()
const localNavigation = ref(null)

onMounted(async () => {
	await nextTick()
	const nav = localNavigation.value
	if (!nav) {
		return
	}

	if (canAccessIamFromStatus(appStatus.value)) {
		nav.addNavigationLink({
			id: 'iam',
			name: 'iamHub',
			title: 'Identity & Access',
			path: '/control-panel/iam',
			icon: WebSecurityIcon,
			type: 'route',
			description: 'Users, groups, roles and permissions',
		})
	}
})
</script>
