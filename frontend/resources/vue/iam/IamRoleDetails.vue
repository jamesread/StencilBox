<template>
	<Section :title="role?.name || 'Role'" subtitle="Which permissions this role grants." :icon="WebSecurityIcon">
		<template #toolbar>
			<router-link :to="{ name: 'iamRoles' }" class="button inline-icon neutral">
				<HugeiconsIcon :icon="ArrowLeft01Icon" width="1em" height="1em" :strokeWidth="2.5" />
				<span>Roles</span>
			</router-link>
			<button type="button" class="inline-icon neutral" aria-label="Refresh" :disabled="loading" @click="load">
				<HugeiconsIcon :icon="RefreshIcon" width="1em" height="1em" :strokeWidth="2.5" />
			</button>
		</template>

		<NotificationBlock v-if="errorText" type="error" :message="errorText" />
		<NotificationBlock v-if="saveMessage" type="success" :message="saveMessage" />
		<p v-if="loading">Loading…</p>
		<p v-else-if="!role" class="inline-notification note">Role not found.</p>
		<template v-else>
			<p class="inline-tags">
				<span v-if="isSystemRole(role.name)" class="tag note">system role</span>
				<span v-if="isSuperuserRole" class="tag good">all access</span>
			</p>
			<p v-if="isSuperuserRole" class="inline-notification note">
				The <strong>superuser</strong> role grants every permission, including ones added later. It cannot be edited.
			</p>

			<FormLayout @submit.prevent="save">
				<FormField label="Name" for="role-name">
					<input id="role-name" v-model="form.name" type="text" :disabled="readOnly || isSystemRole(role.name) || saving" />
				</FormField>
				<FormField label="Description" for="role-desc">
					<input id="role-desc" v-model="form.description" type="text" :disabled="readOnly || saving" />
				</FormField>
				<FormField v-if="!isSuperuserRole" label="Permissions" component-has-label>
					<PermissionCheckTable v-model="form.permissionIds" :permissions="permissions" :disabled="readOnly || saving" />
				</FormField>
				<p v-if="!canManage" class="inline-notification note">Editing roles needs the rbac.manage permission.</p>
				<template v-if="!readOnly" #actions>
					<button type="submit" class="good" :disabled="saving || !form.name.trim()">{{ saving ? 'Saving…' : 'Save role' }}</button>
				</template>
			</FormLayout>
		</template>
	</Section>

	<Section v-if="role" title="Groups with this role" :icon="UserGroupIcon">
		<p v-if="!groupNames.length" class="inline-notification note">No group has this role yet. Give it to a group from User Groups.</p>
		<p v-else class="inline-tags">
			<span v-for="name in groupNames" :key="name" class="tag note">{{ name }}</span>
		</p>
	</Section>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { HugeiconsIcon } from '@hugeicons/vue'
import { ArrowLeft01Icon, RefreshIcon, UserGroupIcon, WebSecurityIcon } from '@hugeicons/core-free-icons'
import Section from 'picocrank/vue/components/Section.vue'
import FormLayout from 'picocrank/vue/components/FormLayout.vue'
import FormField from 'picocrank/vue/components/FormField.vue'
import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue'
import PermissionCheckTable from './PermissionCheckTable.vue'
import { useAppStatus, errorMessage } from '../../../scripts/appStatus.js'
import { isSystemRole } from '../../../scripts/rbacAccess.js'

const props = defineProps({ id: { type: String, required: true } })
const { hasPermission } = useAppStatus()

const role = ref(null)
const permissions = ref([])
const groupNames = ref([])
const loading = ref(true)
const saving = ref(false)
const errorText = ref('')
const saveMessage = ref('')
const form = ref({ name: '', description: '', permissionIds: [] })

const roleId = computed(() => Number(props.id))
const canManage = computed(() => hasPermission('rbac.manage'))
const isSuperuserRole = computed(() => role.value?.name === 'superuser')
const readOnly = computed(() => !canManage.value || isSuperuserRole.value)

async function load() {
	loading.value = true
	errorText.value = ''
	try {
		const [permsRes, rolesRes, groupsRes] = await Promise.all([
			window.client.listRbacPermissions({}),
			window.client.listRbacRoles({}),
			window.client.getRbacRoleGroups({ roleId: roleId.value }),
		])
		permissions.value = permsRes.permissions || []
		role.value = (rolesRes.roles || []).find((r) => r.id === roleId.value) || null
		groupNames.value = groupsRes.groupNames || []
		if (role.value) {
			form.value = {
				name: role.value.name,
				description: role.value.description || '',
				permissionIds: [...role.value.permissionIds],
			}
		}
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to load role')
		role.value = null
	} finally {
		loading.value = false
	}
}

async function save() {
	saving.value = true
	errorText.value = ''
	saveMessage.value = ''
	try {
		await window.client.updateRbacRole({
			roleId: roleId.value,
			name: form.value.name.trim(),
			description: form.value.description.trim(),
			permissionIds: form.value.permissionIds,
		})
		await load()
		saveMessage.value = 'Role saved.'
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to save role')
	} finally {
		saving.value = false
	}
}

watch(() => props.id, load)
onMounted(load)
</script>
