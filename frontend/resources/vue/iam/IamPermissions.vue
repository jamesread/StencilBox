<template>
	<Section
		title="Permission Catalog"
		subtitle="Every permission. Users get them through roles given to their groups."
		:icon="ShieldKeyIcon"
		:padding="false"
	>
		<template #toolbar>
			<router-link :to="{ name: 'iamRoles' }" class="button inline-icon neutral">
				<HugeiconsIcon :icon="WebSecurityIcon" width="1em" height="1em" :strokeWidth="2.5" />
				<span>Roles</span>
			</router-link>
			<button type="button" class="inline-icon neutral" aria-label="Refresh" :disabled="loading" @click="load">
				<HugeiconsIcon :icon="RefreshIcon" width="1em" height="1em" :strokeWidth="2.5" />
			</button>
		</template>

		<div v-if="errorText" class="padded">
			<NotificationBlock type="error" :message="errorText" />
		</div>
		<p v-if="loading && !permissions.length" class="padded">Loading…</p>
		<Table v-else :data="rows" :headers="headers" row-key="id" :filterable="false" :column-options="false" :page-size="1000" :show-pagination="false">
			<template #cell-name="{ value }">
				<code>{{ value }}</code>
			</template>
			<template #cell-roles="{ row }">
				<span class="inline-tags">
					<span v-for="name in row.roleNames" :key="name" class="tag">{{ name }}</span>
				</span>
			</template>
		</Table>
	</Section>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { HugeiconsIcon } from '@hugeicons/vue'
import { RefreshIcon, ShieldKeyIcon, WebSecurityIcon } from '@hugeicons/core-free-icons'
import Section from 'picocrank/vue/components/Section.vue'
import Table from 'picocrank/vue/components/Table.vue'
import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue'
import { errorMessage } from '../../../scripts/appStatus.js'

const headers = [
	{ key: 'name', label: 'Permission', sortable: true },
	{ key: 'description', label: 'Description' },
	{ key: 'roles', label: 'Granted by roles' },
]

const permissions = ref([])
const roles = ref([])
const loading = ref(true)
const errorText = ref('')

const rows = computed(() => permissions.value.map((p) => ({
	...p,
	description: p.description || '—',
	// superuser grants everything, whether or not the permission is ticked on it.
	roleNames: roles.value
		.filter((r) => r.name === 'superuser' || r.permissionIds.includes(p.id))
		.map((r) => r.name)
		.sort(),
})))

async function load() {
	loading.value = true
	errorText.value = ''
	try {
		const [permsRes, rolesRes] = await Promise.all([
			window.client.listRbacPermissions({}),
			window.client.listRbacRoles({}),
		])
		permissions.value = permsRes.permissions || []
		roles.value = rolesRes.roles || []
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to load permissions')
	} finally {
		loading.value = false
	}
}

onMounted(load)
</script>
