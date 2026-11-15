<template>
	<dialog ref="createDialog" class="iam-dialog" @close="resetCreate">
		<h2>Create user</h2>
		<p>The password is optional. Without one, the user can only sign in through SSO or an API key.</p>
		<FormLayout @submit.prevent="createUser">
			<FormField label="Username" for="new-user-name">
				<input id="new-user-name" ref="nameInput" v-model="form.username" type="text" autocomplete="off" required :disabled="saving" />
			</FormField>
			<FormField label="Password" for="new-user-password">
				<input id="new-user-password" v-model="form.password" type="password" autocomplete="new-password" placeholder="Optional (at least 8 characters)" :disabled="saving" />
			</FormField>
			<NotificationBlock v-if="createError" type="error" :message="createError" />
			<template #actions>
				<button type="button" class="neutral" @click="createDialog.close()">Cancel</button>
				<button type="submit" class="good" :disabled="saving || !form.username.trim()">{{ saving ? 'Creating…' : 'Create user' }}</button>
			</template>
		</FormLayout>
	</dialog>

	<Section
		title="Users"
		subtitle="User accounts. Access comes from the groups a user belongs to."
		:icon="UserMultiple02Icon"
		:padding="false"
	>
		<template #toolbar>
			<button type="button" class="inline-icon neutral" aria-label="Refresh" :disabled="loading" @click="load">
				<HugeiconsIcon :icon="RefreshIcon" width="1em" height="1em" :strokeWidth="2.5" />
			</button>
			<button v-if="canCreate" type="button" class="inline-icon good" aria-label="Create user" :disabled="loading" @click="openCreate">
				<HugeiconsIcon :icon="Add01Icon" width="1em" height="1em" :strokeWidth="2.5" />
			</button>
		</template>

		<div v-if="errorText" class="padded">
			<NotificationBlock type="error" :message="errorText" />
		</div>
		<p v-if="loading && !users.length" class="padded">Loading…</p>
		<Table
			v-else
			:data="users"
			:headers="headers"
			row-key="id"
			:filterable="false"
			:column-options="false"
			@row-click="openUser"
		>
			<template #cell-username="{ row }">
				<strong>{{ row.username }}</strong>
				<span v-if="row.username === appStatus?.username" class="tag note">you</span>
			</template>
			<template #cell-actions="{ row }">
				<button
					type="button"
					class="bad small"
					data-row-click-ignore
					:disabled="saving || row.username === appStatus?.username"
					:title="row.username === appStatus?.username ? 'You cannot delete your own account' : 'Delete user'"
					@click.stop="deleteUser(row)"
				>Delete</button>
			</template>
		</Table>
	</Section>
</template>

<script setup>
import { ref, reactive, computed, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { HugeiconsIcon } from '@hugeicons/vue'
import { Add01Icon, RefreshIcon, UserMultiple02Icon } from '@hugeicons/core-free-icons'
import Section from 'picocrank/vue/components/Section.vue'
import Table from 'picocrank/vue/components/Table.vue'
import FormLayout from 'picocrank/vue/components/FormLayout.vue'
import FormField from 'picocrank/vue/components/FormField.vue'
import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue'
import { useAppStatus, errorMessage } from '../../../scripts/appStatus.js'

const router = useRouter()
const { appStatus, hasPermission } = useAppStatus()

const users = ref([])
const loading = ref(true)
const saving = ref(false)
const errorText = ref('')
const createDialog = ref(null)
const nameInput = ref(null)
const createError = ref('')
const form = reactive({ username: '', password: '' })

const canCreate = computed(() => hasPermission('users.create'))
const canDelete = computed(() => hasPermission('users.delete'))

const headers = computed(() => {
	const h = [{ key: 'username', label: 'Username', sortable: true }]
	if (canDelete.value) {
		h.push({ key: 'actions', label: '', width: '6rem' })
	}
	return h
})

async function load() {
	loading.value = true
	errorText.value = ''
	try {
		const res = await window.client.getUsers({})
		users.value = (res.users || []).slice().sort((a, b) => a.username.localeCompare(b.username))
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to load users')
	} finally {
		loading.value = false
	}
}

function openUser({ row }) {
	router.push({ name: 'iamUserDetails', params: { id: String(row.id) } })
}

function resetCreate() {
	form.username = ''
	form.password = ''
	createError.value = ''
}

function openCreate() {
	resetCreate()
	createDialog.value.showModal()
	nextTick(() => nameInput.value?.focus())
}

async function createUser() {
	if (form.password && form.password.length < 8) {
		createError.value = 'Password must be at least 8 characters.'
		return
	}
	saving.value = true
	createError.value = ''
	try {
		const res = await window.client.createUser({ username: form.username.trim(), password: form.password })
		createDialog.value.close()
		router.push({ name: 'iamUserDetails', params: { id: String(res.user.id) } })
	} catch (e) {
		createError.value = errorMessage(e, 'Failed to create user')
	} finally {
		saving.value = false
	}
}

async function deleteUser(user) {
	if (!confirm(`Delete user "${user.username}"? This cannot be undone.`)) {
		return
	}
	saving.value = true
	errorText.value = ''
	try {
		await window.client.deleteUser({ userId: user.id })
		await load()
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to delete user')
	} finally {
		saving.value = false
	}
}

onMounted(load)
</script>
