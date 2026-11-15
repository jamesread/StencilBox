<template>
	<Section :title="group?.name || 'User group'" subtitle="Members inherit every role the group has." :icon="UserGroupIcon">
		<template #toolbar>
			<router-link :to="{ name: 'iamGroups' }" class="button inline-icon neutral">
				<HugeiconsIcon :icon="ArrowLeft01Icon" width="1em" height="1em" :strokeWidth="2.5" />
				<span>User Groups</span>
			</router-link>
			<button type="button" class="inline-icon neutral" aria-label="Refresh" :disabled="loading" @click="load">
				<HugeiconsIcon :icon="RefreshIcon" width="1em" height="1em" :strokeWidth="2.5" />
			</button>
			<button v-if="canManageMembers && group && !isSystem" type="button" class="bad" :disabled="saving" @click="deleteGroup">Delete group</button>
		</template>

		<NotificationBlock v-if="errorText" type="error" :message="errorText" />
		<NotificationBlock v-if="saveMessage" type="success" :message="saveMessage" />
		<p v-if="loading">Loading…</p>
		<p v-else-if="!group" class="inline-notification note">User group not found.</p>
		<template v-else>
			<dl class="meta-grid">
				<dt>Group ID</dt>
				<dd>{{ group.id }}</dd>
				<dt>Members</dt>
				<dd>{{ memberIds.length }}</dd>
			</dl>
			<p v-if="group.name === 'Everyone'" class="inline-notification warning">
				<strong>Everyone</strong> is a system group: every user is added to it, so its roles apply to all users.
			</p>
			<p v-else-if="group.name === 'Administrators'" class="inline-notification warning">
				<strong>Administrators</strong> is a system group with the superuser role. At least one user must keep superuser.
			</p>
		</template>
	</Section>

	<Section v-if="group" title="Members" :icon="UserMultiple02Icon">
		<FormLayout @submit.prevent="saveMembers">
			<FormField label="Users in this group" component-has-label>
				<CheckGroup v-model="memberIds" name="group-members" :options="userOptions" :disabled="!canManageMembers || saving" />
			</FormField>
			<p v-if="!canManageMembers" class="inline-notification note">Changing members needs the usergroups.manage permission.</p>
			<template v-if="canManageMembers" #actions>
				<button type="submit" class="good" :disabled="saving">{{ saving ? 'Saving…' : 'Save members' }}</button>
			</template>
		</FormLayout>
	</Section>

	<Section v-if="group && canViewRoles" title="Roles" subtitle="Every member of this group gets these roles." :icon="WebSecurityIcon">
		<FormLayout @submit.prevent="saveRoles">
			<FormField label="Roles for this group" component-has-label>
				<CheckGroup v-model="roleIds" name="group-roles" :options="roleOptions" :disabled="!canManageRoles || saving" />
			</FormField>
			<p v-if="!canManageRoles" class="inline-notification note">Changing roles needs the rbac.manage permission.</p>
			<template v-if="canManageRoles" #actions>
				<button type="submit" class="good" :disabled="saving">{{ saving ? 'Saving…' : 'Save roles' }}</button>
			</template>
		</FormLayout>
	</Section>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { HugeiconsIcon } from '@hugeicons/vue'
import { ArrowLeft01Icon, RefreshIcon, UserGroupIcon, UserMultiple02Icon, WebSecurityIcon } from '@hugeicons/core-free-icons'
import Section from 'picocrank/vue/components/Section.vue'
import FormLayout from 'picocrank/vue/components/FormLayout.vue'
import FormField from 'picocrank/vue/components/FormField.vue'
import CheckGroup from 'picocrank/vue/components/CheckGroup.vue'
import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue'
import { useAppStatus, errorMessage } from '../../../scripts/appStatus.js'
import { isSystemGroup } from '../../../scripts/rbacAccess.js'

const props = defineProps({ id: { type: String, required: true } })
const router = useRouter()
const { hasPermission } = useAppStatus()

const group = ref(null)
const users = ref([])
const roles = ref([])
const memberIds = ref([])
const roleIds = ref([])
const loading = ref(true)
const saving = ref(false)
const errorText = ref('')
const saveMessage = ref('')

const groupId = computed(() => Number(props.id))
const isSystem = computed(() => (group.value ? isSystemGroup(group.value.name) : false))
const canManageMembers = computed(() => hasPermission('usergroups.manage'))
const canViewRoles = computed(() => hasPermission('rbac.view'))
const canManageRoles = computed(() => hasPermission('rbac.manage'))

const userOptions = computed(() => users.value
	.slice()
	.sort((a, b) => a.username.localeCompare(b.username))
	.map((u) => ({ value: u.id, label: u.username })))

const roleOptions = computed(() => roles.value.map((r) => ({
	value: r.id,
	label: r.description ? `${r.name} — ${r.description}` : r.name,
})))

async function load() {
	loading.value = true
	errorText.value = ''
	try {
		const [groupsRes, usersRes, membersRes] = await Promise.all([
			window.client.listUserGroups({}),
			window.client.getUsers({}),
			window.client.getUserGroupMembers({ groupId: groupId.value }),
		])
		group.value = (groupsRes.groups || []).find((g) => g.id === groupId.value) || null
		users.value = usersRes.users || []
		memberIds.value = [...membersRes.userIds]

		if (canViewRoles.value) {
			const [rolesRes, groupRolesRes] = await Promise.all([
				window.client.listRbacRoles({}),
				window.client.getUserGroupRbacRoles({ groupId: groupId.value }),
			])
			roles.value = rolesRes.roles || []
			roleIds.value = [...groupRolesRes.roleIds]
		}
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to load group')
		group.value = null
	} finally {
		loading.value = false
	}
}

async function save(fn, message) {
	saving.value = true
	errorText.value = ''
	saveMessage.value = ''
	try {
		await fn()
		await load()
		saveMessage.value = message
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to save')
	} finally {
		saving.value = false
	}
}

function saveMembers() {
	return save(() => window.client.setUserGroupMembers({ groupId: groupId.value, userIds: memberIds.value }), 'Members saved.')
}

function saveRoles() {
	return save(() => window.client.setUserGroupRbacRoles({ groupId: groupId.value, roleIds: roleIds.value }), 'Roles saved.')
}

async function deleteGroup() {
	if (!confirm(`Delete group "${group.value.name}"? Its members lose the roles it grants.`)) {
		return
	}
	saving.value = true
	try {
		await window.client.deleteUserGroup({ groupId: groupId.value })
		router.push({ name: 'iamGroups' })
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to delete group')
	} finally {
		saving.value = false
	}
}

watch(() => props.id, load)
onMounted(load)
</script>
