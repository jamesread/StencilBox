<template>
	<Section :title="user?.username || 'User'" subtitle="Account details and effective access" :icon="UserIcon">
		<template #toolbar>
			<router-link :to="{ name: 'iamUsers' }" class="button inline-icon neutral">
				<HugeiconsIcon :icon="ArrowLeft01Icon" width="1em" height="1em" :strokeWidth="2.5" />
				<span>Users</span>
			</router-link>
			<button type="button" class="inline-icon neutral" aria-label="Refresh" :disabled="loading" @click="load">
				<HugeiconsIcon :icon="RefreshIcon" width="1em" height="1em" :strokeWidth="2.5" />
			</button>
		</template>

		<NotificationBlock v-if="errorText" type="error" :message="errorText" />
		<p v-if="loading">Loading…</p>
		<template v-else-if="user">
			<dl class="meta-grid">
				<dt>User ID</dt>
				<dd>{{ user.id }}</dd>
				<dt>Created by</dt>
				<dd>{{ user.createdBy || '—' }}</dd>
				<dt>Password login</dt>
				<dd>{{ user.hasPassword ? 'Enabled' : 'No password set (SSO or API key only)' }}</dd>
			</dl>
		</template>
	</Section>

	<Section v-if="user" title="Groups" subtitle="Users get their roles, and so their permissions, from groups." :icon="UserGroupIcon">
		<FormLayout v-if="allGroups.length" @submit.prevent="saveGroups">
			<FormField label="Groups for this user" component-has-label>
				<CheckGroup v-model="groupIds" name="user-groups" :options="groupOptions" :disabled="!canManageGroups || saving" />
			</FormField>
			<NotificationBlock v-if="groupsMessage" type="success" :message="groupsMessage" />
			<template v-if="canManageGroups" #actions>
				<button type="submit" class="good" :disabled="saving">{{ saving ? 'Saving…' : 'Save groups' }}</button>
			</template>
		</FormLayout>
		<p v-else class="inline-tags">
			<span v-for="name in access?.groupNames || []" :key="name" class="tag note">{{ name }}</span>
		</p>
		<p v-if="!canManageGroups" class="inline-notification note">Changing groups needs the usergroups.manage permission.</p>
	</Section>

	<Section v-if="user && access" title="Effective access" subtitle="Read-only. Change access through group membership and group roles." :icon="ShieldKeyIcon">
		<h3>Roles</h3>
		<p v-if="!access.roleNames.length" class="inline-notification note">No roles through group membership.</p>
		<p v-else class="inline-tags">
			<span v-for="name in access.roleNames" :key="name" class="tag">{{ name }}</span>
		</p>

		<h3>Permissions</h3>
		<p v-if="access.isSuperuser" class="inline-notification note">
			This user has the <strong>superuser</strong> role, so every permission is granted.
		</p>
		<PermissionAuditTable :rows="access.permissions" :is-superuser="access.isSuperuser" />
	</Section>

	<Section v-if="user && canResetPassword" title="Reset password" :icon="LockPasswordIcon">
		<FormLayout @submit.prevent="resetPassword">
			<FormField label="New password" for="reset-password" description="At least 8 characters. The user is signed out of every session.">
				<input id="reset-password" v-model="newPassword" type="password" autocomplete="new-password" :disabled="saving" />
			</FormField>
			<NotificationBlock v-if="resetError" type="error" :message="resetError" />
			<NotificationBlock v-if="resetStatus" type="success" :message="resetStatus" />
			<template #actions>
				<button type="submit" class="good" :disabled="saving || !newPassword">{{ saving ? 'Saving…' : 'Reset password' }}</button>
			</template>
		</FormLayout>
	</Section>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { HugeiconsIcon } from '@hugeicons/vue'
import { ArrowLeft01Icon, LockPasswordIcon, RefreshIcon, ShieldKeyIcon, UserGroupIcon, UserIcon } from '@hugeicons/core-free-icons'
import Section from 'picocrank/vue/components/Section.vue'
import FormLayout from 'picocrank/vue/components/FormLayout.vue'
import FormField from 'picocrank/vue/components/FormField.vue'
import CheckGroup from 'picocrank/vue/components/CheckGroup.vue'
import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue'
import PermissionAuditTable from './PermissionAuditTable.vue'
import { useAppStatus, errorMessage } from '../../../scripts/appStatus.js'

const props = defineProps({ id: { type: String, required: true } })
const { hasPermission } = useAppStatus()

const user = ref(null)
const access = ref(null)
const allGroups = ref([])
const groupIds = ref([])
const loading = ref(true)
const saving = ref(false)
const errorText = ref('')
const groupsMessage = ref('')
const newPassword = ref('')
const resetError = ref('')
const resetStatus = ref('')

const userId = computed(() => Number(props.id))
const canManageGroups = computed(() => hasPermission('usergroups.manage'))
const canResetPassword = computed(() => hasPermission('users.reset-password'))
const groupOptions = computed(() => allGroups.value.map((g) => ({
	value: g.id,
	label: `${g.name} (${g.memberCount} ${g.memberCount === 1 ? 'member' : 'members'})`,
})))

async function load() {
	loading.value = true
	errorText.value = ''
	groupsMessage.value = ''
	try {
		const [userRes, accessRes] = await Promise.all([
			window.client.getUser({ userId: userId.value }),
			window.client.getUserAccess({ userId: userId.value }),
		])
		user.value = userRes.user
		access.value = accessRes
		groupIds.value = [...accessRes.groupIds]
		if (hasPermission('usergroups.view')) {
			const groupsRes = await window.client.listUserGroups({})
			allGroups.value = groupsRes.groups || []
		}
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to load user')
		user.value = null
	} finally {
		loading.value = false
	}
}

async function saveGroups() {
	saving.value = true
	errorText.value = ''
	groupsMessage.value = ''
	try {
		await window.client.setUserGroups({ userId: userId.value, groupIds: groupIds.value })
		await load()
		groupsMessage.value = 'Groups saved.'
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to save groups')
	} finally {
		saving.value = false
	}
}

async function resetPassword() {
	resetError.value = ''
	resetStatus.value = ''
	if (newPassword.value.length < 8) {
		resetError.value = 'Password must be at least 8 characters.'
		return
	}
	saving.value = true
	try {
		const res = await window.client.resetUserPassword({ userId: userId.value, newPassword: newPassword.value })
		if (res.success) {
			resetStatus.value = res.message || 'Password reset.'
			newPassword.value = ''
			await load()
		} else {
			resetError.value = res.message || 'Failed to reset password'
		}
	} catch (e) {
		resetError.value = errorMessage(e, 'Failed to reset password')
	} finally {
		saving.value = false
	}
}

watch(() => props.id, load)
onMounted(load)
</script>
