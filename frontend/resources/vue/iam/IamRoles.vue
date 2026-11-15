<template>
	<dialog ref="createDialog" class="iam-dialog wide" @close="resetCreate">
		<h2>Create role</h2>
		<p>Choose which permissions the role grants, then give the role to a group.</p>
		<FormLayout @submit.prevent="createRole">
			<FormField label="Name" for="new-role-name">
				<input id="new-role-name" ref="nameInput" v-model="form.name" type="text" autocomplete="off" required :disabled="saving" />
			</FormField>
			<FormField label="Description" for="new-role-desc">
				<input id="new-role-desc" v-model="form.description" type="text" :disabled="saving" />
			</FormField>
			<FormField label="Permissions" component-has-label>
				<PermissionCheckTable v-model="form.permissionIds" :permissions="permissions" :disabled="saving" />
			</FormField>
			<NotificationBlock v-if="createError" type="error" :message="createError" />
			<template #actions>
				<button type="button" class="neutral" @click="createDialog.close()">Cancel</button>
				<button type="submit" class="good" :disabled="saving || !form.name.trim()">{{ saving ? 'Creating…' : 'Create role' }}</button>
			</template>
		</FormLayout>
	</dialog>

	<Section
		title="Roles"
		subtitle="Roles grant permissions. Give roles to groups, never directly to users."
		:icon="WebSecurityIcon"
		:padding="false"
	>
		<template #toolbar>
			<router-link :to="{ name: 'iamPermissions' }" class="button inline-icon neutral">
				<HugeiconsIcon :icon="ShieldKeyIcon" width="1em" height="1em" :strokeWidth="2.5" />
				<span>Permission Catalog</span>
			</router-link>
			<button type="button" class="inline-icon neutral" aria-label="Refresh" :disabled="loading" @click="load">
				<HugeiconsIcon :icon="RefreshIcon" width="1em" height="1em" :strokeWidth="2.5" />
			</button>
			<button v-if="canManage" type="button" class="inline-icon good" aria-label="Create role" :disabled="loading" @click="openCreate">
				<HugeiconsIcon :icon="Add01Icon" width="1em" height="1em" :strokeWidth="2.5" />
			</button>
		</template>

		<div v-if="errorText" class="padded">
			<NotificationBlock type="error" :message="errorText" />
		</div>
		<p v-if="loading && !roles.length" class="padded">Loading…</p>
		<Table
			v-else
			:data="rows"
			:headers="headers"
			row-key="id"
			:filterable="false"
			:column-options="false"
			@row-click="openRole"
		>
			<template #cell-name="{ row }">
				<strong>{{ row.name }}</strong>
				<span class="inline-tags">
					<span v-if="isSystemRole(row.name)" class="tag note">system role</span>
					<span v-if="row.name === 'superuser'" class="tag good">all access</span>
				</span>
			</template>
			<template #cell-actions="{ row }">
				<button
					v-if="!isSystemRole(row.name)"
					type="button"
					class="bad small"
					data-row-click-ignore
					:disabled="saving"
					@click.stop="deleteRole(row)"
				>Delete</button>
			</template>
		</Table>
	</Section>
</template>

<script setup>
import { ref, reactive, computed, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { HugeiconsIcon } from '@hugeicons/vue'
import { Add01Icon, RefreshIcon, ShieldKeyIcon, WebSecurityIcon } from '@hugeicons/core-free-icons'
import Section from 'picocrank/vue/components/Section.vue'
import Table from 'picocrank/vue/components/Table.vue'
import FormLayout from 'picocrank/vue/components/FormLayout.vue'
import FormField from 'picocrank/vue/components/FormField.vue'
import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue'
import PermissionCheckTable from './PermissionCheckTable.vue'
import { useAppStatus, errorMessage } from '../../../scripts/appStatus.js'
import { isSystemRole } from '../../../scripts/rbacAccess.js'

const router = useRouter()
const { hasPermission } = useAppStatus()

const roles = ref([])
const permissions = ref([])
const loading = ref(true)
const saving = ref(false)
const errorText = ref('')
const createDialog = ref(null)
const nameInput = ref(null)
const createError = ref('')
const form = reactive({ name: '', description: '', permissionIds: [] })

const canManage = computed(() => hasPermission('rbac.manage'))
const headers = computed(() => {
	const h = [
		{ key: 'name', label: 'Name', sortable: true },
		{ key: 'description', label: 'Description' },
		{ key: 'permissionsLabel', label: 'Permissions' },
		{ key: 'groupCount', label: 'Groups', sortable: true, width: '6rem' },
		{ key: 'userCount', label: 'Users', sortable: true, width: '6rem' },
	]
	if (canManage.value) {
		h.push({ key: 'actions', label: '', width: '6rem' })
	}
	return h
})

const rows = computed(() => {
	const names = new Map(permissions.value.map((p) => [p.id, p.name]))
	return roles.value.map((r) => ({
		...r,
		description: r.description || '—',
		permissionsLabel: r.name === 'superuser'
			? 'all permissions'
			: r.permissionIds.map((id) => names.get(id) || String(id)).sort().join(', ') || '—',
	}))
})

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
		errorText.value = errorMessage(e, 'Failed to load roles')
	} finally {
		loading.value = false
	}
}

function openRole({ row }) {
	router.push({ name: 'iamRoleDetails', params: { id: String(row.id) } })
}

function resetCreate() {
	form.name = ''
	form.description = ''
	form.permissionIds = []
	createError.value = ''
}

function openCreate() {
	resetCreate()
	createDialog.value.showModal()
	nextTick(() => nameInput.value?.focus())
}

async function createRole() {
	saving.value = true
	createError.value = ''
	try {
		const res = await window.client.createRbacRole({
			name: form.name.trim(),
			description: form.description.trim(),
			permissionIds: form.permissionIds,
		})
		createDialog.value.close()
		router.push({ name: 'iamRoleDetails', params: { id: String(res.role.id) } })
	} catch (e) {
		createError.value = errorMessage(e, 'Failed to create role')
	} finally {
		saving.value = false
	}
}

async function deleteRole(role) {
	if (!confirm(`Delete role "${role.name}"? Groups lose it, and their members lose its permissions.`)) {
		return
	}
	saving.value = true
	errorText.value = ''
	try {
		await window.client.deleteRbacRole({ roleId: role.id })
		await load()
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to delete role')
	} finally {
		saving.value = false
	}
}

onMounted(load)
</script>
