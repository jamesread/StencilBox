<template>
	<Section
		title="Identity &amp; Access Management"
		subtitle="Users belong to groups, groups have roles, and roles grant permissions."
		:icon="WebSecurityIcon"
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
import {
	UserMultiple02Icon,
	UserGroupIcon,
	WebSecurityIcon,
	ShieldKeyIcon,
} from '@hugeicons/core-free-icons'
import { useAppStatus } from '../../../scripts/appStatus.js'

const { hasPermission } = useAppStatus()
const localNavigation = ref(null)

onMounted(async () => {
	await nextTick()
	const nav = localNavigation.value
	if (!nav) {
		return
	}

	if (hasPermission('users.view')) {
		nav.addNavigationLink({
			id: 'iam-users',
			name: 'iamUsers',
			title: 'Users',
			path: '/control-panel/iam/users',
			icon: UserMultiple02Icon,
			type: 'route',
			description: 'Create and manage user accounts',
		})
	}

	if (hasPermission('usergroups.view')) {
		nav.addNavigationLink({
			id: 'iam-groups',
			name: 'iamGroups',
			title: 'User Groups',
			path: '/control-panel/iam/groups',
			icon: UserGroupIcon,
			type: 'route',
			description: 'Organise users into groups, and give groups roles',
		})
	}

	if (hasPermission('rbac.view')) {
		nav.addNavigationLink({
			id: 'iam-roles',
			name: 'iamRoles',
			title: 'Roles',
			path: '/control-panel/iam/roles',
			icon: WebSecurityIcon,
			type: 'route',
			description: 'Choose which permissions each role grants',
		})
		nav.addNavigationLink({
			id: 'iam-permissions',
			name: 'iamPermissions',
			title: 'Permission Catalog',
			path: '/control-panel/iam/permissions',
			icon: ShieldKeyIcon,
			type: 'route',
			description: 'Every permission, and the roles that grant it',
		})
	}
})
</script>
