<template>
	<Section title="My Account" :subtitle="'Signed in as ' + (appStatus?.username || '')" :icon="UserIcon">
		<p v-if="!databaseEnabled" class="inline-notification note">
			StencilBox is running without a database, so you are signed in as <strong>localhost</strong> because you
			are using it from this computer. Accounts, passwords and API keys need a database.
		</p>

		<template v-if="databaseEnabled" #toolbar>
			<button type="button" class="inline-icon neutral" :disabled="loggingOut" @click="logout">
				<HugeiconsIcon :icon="Logout03Icon" width="1em" height="1em" :strokeWidth="2.5" />
				<span>{{ loggingOut ? 'Logging out…' : 'Log out' }}</span>
			</button>
		</template>

		<Navigation ref="localNavigation">
			<NavigationGrid />
		</Navigation>
	</Section>

	<Section v-if="databaseEnabled" title="Change password" :icon="LockPasswordIcon">
		<FormLayout @submit.prevent="changePassword">
			<FormField label="Current password" for="current-password">
				<input id="current-password" v-model="current" type="password" autocomplete="current-password" :disabled="saving" />
			</FormField>
			<FormField label="New password" for="new-password" description="At least 8 characters.">
				<input id="new-password" v-model="next" type="password" autocomplete="new-password" :disabled="saving" />
			</FormField>
			<FormField label="Confirm new password" for="confirm-password">
				<input id="confirm-password" v-model="confirmNext" type="password" autocomplete="new-password" :disabled="saving" />
			</FormField>
			<NotificationBlock v-if="passwordError" type="error" :message="passwordError" />
			<NotificationBlock v-if="passwordStatus" type="success" :message="passwordStatus" />
			<template #actions>
				<button type="submit" class="good" :disabled="saving || !next">{{ saving ? 'Saving…' : 'Change password' }}</button>
			</template>
		</FormLayout>
	</Section>
</template>

<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import { HugeiconsIcon } from '@hugeicons/vue'
import { Key01Icon, LockPasswordIcon, Logout03Icon, ShieldKeyIcon, UserIcon } from '@hugeicons/core-free-icons'
import Section from 'picocrank/vue/components/Section.vue'
import Navigation from 'picocrank/vue/components/Navigation.vue'
import NavigationGrid from 'picocrank/vue/components/NavigationGrid.vue'
import FormLayout from 'picocrank/vue/components/FormLayout.vue'
import FormField from 'picocrank/vue/components/FormField.vue'
import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue'
import { fetchAppStatus, useAppStatus, errorMessage } from '../../../scripts/appStatus.js'

const { appStatus } = useAppStatus()
const databaseEnabled = computed(() => Boolean(appStatus.value?.databaseEnabled))
const localNavigation = ref(null)
const loggingOut = ref(false)
const saving = ref(false)
const current = ref('')
const next = ref('')
const confirmNext = ref('')
const passwordError = ref('')
const passwordStatus = ref('')

async function logout() {
	loggingOut.value = true
	try {
		await window.client.logout({})
	} catch (e) {
		console.error('Logout failed:', e)
	}
	// A full reload resets navigation and search back to the logged-out state.
	window.location.href = '/webui/'
}

async function changePassword() {
	passwordError.value = ''
	passwordStatus.value = ''
	if (next.value.length < 8) {
		passwordError.value = 'The new password must be at least 8 characters.'
		return
	}
	if (next.value !== confirmNext.value) {
		passwordError.value = 'The new passwords do not match.'
		return
	}
	saving.value = true
	try {
		const res = await window.client.changePassword({ currentPassword: current.value, newPassword: next.value })
		if (res.success) {
			passwordStatus.value = 'Password changed.'
			current.value = ''
			next.value = ''
			confirmNext.value = ''
			await fetchAppStatus({ force: true })
		} else {
			passwordError.value = res.message || 'Failed to change password'
		}
	} catch (e) {
		passwordError.value = errorMessage(e, 'Failed to change password')
	} finally {
		saving.value = false
	}
}

onMounted(async () => {
	await nextTick()
	const nav = localNavigation.value
	if (!nav) {
		return
	}
	nav.addNavigationLink({
		id: 'my-permissions',
		name: 'myPermissions',
		title: 'My Permissions',
		path: '/account/permissions',
		icon: ShieldKeyIcon,
		type: 'route',
		description: 'Your groups, roles and effective permissions',
	})
	if (!databaseEnabled.value) {
		return
	}
	nav.addNavigationLink({
		id: 'my-api-keys',
		name: 'myApiKeys',
		title: 'API Keys',
		path: '/account/api-keys',
		icon: Key01Icon,
		type: 'route',
		description: 'Bearer keys for scripts and automation',
	})
})
</script>
