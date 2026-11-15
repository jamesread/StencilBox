<template>
	<Dialog ref="keyDialog" :title="dialogTitle" size="large" class="api-key-dialog" @close="resetDialog">
		<template v-if="createdKey">
			<p>Copy this key now. It will not be shown again.</p>
			<p><code class="api-key">{{ createdKey }}</code></p>
			<p>Send it with each request as <code>Authorization: Bearer &lt;key&gt;</code>.</p>
		</template>
		<FormLayout v-else id="api-key-form" @submit.prevent="submit">
			<FormField v-if="!editing" label="Name" for="new-key-name" description="What the key is for, such as the script that uses it.">
				<input id="new-key-name" ref="nameInput" v-model="form.name" type="text" autocomplete="off" required :disabled="saving" />
			</FormField>
			<ApiKeyScopeFields v-model="form.scope" :disabled="saving" />
			<NotificationBlock v-if="dialogError" type="error" :message="dialogError" />
		</FormLayout>

		<template #actions="{ close }">
			<template v-if="createdKey">
				<button type="button" class="neutral" @click="copyKey">{{ copied ? 'Copied' : 'Copy' }}</button>
				<button type="button" class="good" @click="close()">Done</button>
			</template>
			<template v-else>
				<button type="button" class="neutral" @click="close()">Cancel</button>
				<button type="submit" form="api-key-form" class="good" :disabled="saving || !canSubmit">
					{{ saving ? 'Saving…' : (editing ? 'Save access' : 'Create key') }}
				</button>
			</template>
		</template>
	</Dialog>

	<Section title="API Keys" subtitle="Bearer keys for scripts and automation. Click a key to change what it can do." :icon="Key01Icon" :padding="false">
		<template #toolbar>
			<button type="button" class="inline-icon neutral" aria-label="Refresh" :disabled="loading" @click="load">
				<HugeiconsIcon :icon="RefreshIcon" width="1em" height="1em" :strokeWidth="2.5" />
			</button>
			<button type="button" class="inline-icon good" aria-label="Create API key" :disabled="loading" @click="openCreate">
				<HugeiconsIcon :icon="Add01Icon" width="1em" height="1em" :strokeWidth="2.5" />
			</button>
		</template>

		<div v-if="errorText" class="padded">
			<NotificationBlock type="error" :message="errorText" />
		</div>
		<p v-if="loading && !keys.length" class="padded">Loading…</p>
		<p v-else-if="!keys.length" class="padded inline-notification note">You have no API keys.</p>
		<Table
			v-else
			:data="rows"
			:headers="headers"
			row-key="id"
			:filterable="false"
			:column-options="false"
			@row-click="openEdit"
		>
			<template #cell-keyHint="{ value }">
				<code>{{ value }}</code>
			</template>
			<template #cell-access="{ row }">
				<span class="inline-tags">
					<span v-if="!row.scope.restrictPermissions" class="tag note">all my permissions</span>
					<span v-for="p in row.scope.restrictPermissions ? row.scope.permissions : []" :key="p" class="tag">{{ p }}</span>
					<span v-if="!row.scope.restrictBuildConfigs" class="tag note">all build configs</span>
					<span v-else-if="!row.scope.buildConfigs.length" class="tag warning">no build configs</span>
					<span v-for="bc in row.scope.restrictBuildConfigs ? row.scope.buildConfigs : []" :key="'bc-' + bc" class="tag good">{{ bc }}</span>
				</span>
			</template>
			<template #cell-actions="{ row }">
				<button type="button" class="bad small" data-row-click-ignore :disabled="saving" @click.stop="deleteKey(row)">Delete</button>
			</template>
		</Table>
	</Section>
</template>

<script setup>
import { ref, reactive, computed, onMounted, nextTick } from 'vue'
import { HugeiconsIcon } from '@hugeicons/vue'
import { Add01Icon, Key01Icon, RefreshIcon } from '@hugeicons/core-free-icons'
import Section from 'picocrank/vue/components/Section.vue'
import Table from 'picocrank/vue/components/Table.vue'
import Dialog from 'picocrank/vue/components/Dialog.vue'
import FormLayout from 'picocrank/vue/components/FormLayout.vue'
import FormField from 'picocrank/vue/components/FormField.vue'
import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue'
import ApiKeyScopeFields from './ApiKeyScopeFields.vue'
import { errorMessage } from '../../../scripts/appStatus.js'

const headers = [
	{ key: 'name', label: 'Name', sortable: true },
	{ key: 'keyHint', label: 'Key' },
	{ key: 'access', label: 'Access' },
	{ key: 'createdAt', label: 'Created', sortable: true },
	{ key: 'lastUsedAt', label: 'Last used', sortable: true },
	{ key: 'actions', label: '', width: '6rem' },
]

function emptyScope() {
	return { restrictPermissions: false, permissions: [], restrictBuildConfigs: false, buildConfigs: [] }
}

const keys = ref([])
const loading = ref(true)
const saving = ref(false)
const errorText = ref('')
const keyDialog = ref(null)
const nameInput = ref(null)
const editing = ref(null) // the key being edited, or null when creating
const form = reactive({ name: '', scope: emptyScope() })
const dialogError = ref('')
const createdKey = ref('')
const copied = ref(false)

const rows = computed(() => keys.value.map((k) => ({
	...k,
	scope: k.scope || emptyScope(),
	lastUsedAt: k.lastUsedAt || 'never',
})))

const dialogTitle = computed(() => {
	if (createdKey.value) {
		return 'API key created'
	}
	return editing.value ? `Access for "${editing.value.name}"` : 'Create API key'
})

const canSubmit = computed(() => {
	if (!editing.value && !form.name.trim()) {
		return false
	}
	return !form.scope.restrictPermissions || form.scope.permissions.length > 0
})

async function load() {
	loading.value = true
	errorText.value = ''
	try {
		const res = await window.client.listApiKeys({})
		keys.value = res.apiKeys || []
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to load API keys')
	} finally {
		loading.value = false
	}
}

function resetDialog() {
	editing.value = null
	form.name = ''
	form.scope = emptyScope()
	dialogError.value = ''
	createdKey.value = ''
	copied.value = false
}

function openCreate() {
	resetDialog()
	keyDialog.value.show()
	nextTick(() => nameInput.value?.focus())
}

function openEdit({ row }) {
	resetDialog()
	editing.value = row
	form.scope = {
		restrictPermissions: row.scope.restrictPermissions,
		permissions: [...row.scope.permissions],
		restrictBuildConfigs: row.scope.restrictBuildConfigs,
		buildConfigs: [...row.scope.buildConfigs],
	}
	keyDialog.value.show()
}

async function submit() {
	saving.value = true
	dialogError.value = ''
	try {
		if (editing.value) {
			await window.client.updateApiKey({ apiKeyId: editing.value.id, scope: form.scope })
			keyDialog.value.close()
		} else {
			const res = await window.client.createApiKey({ name: form.name.trim(), scope: form.scope })
			createdKey.value = res.key
		}
		await load()
	} catch (e) {
		dialogError.value = errorMessage(e, 'Failed to save the API key')
	} finally {
		saving.value = false
	}
}

async function copyKey() {
	try {
		await navigator.clipboard.writeText(createdKey.value)
		copied.value = true
	} catch (e) {
		console.error('Copy failed:', e)
	}
}

async function deleteKey(key) {
	if (!confirm(`Delete API key "${key.name}"? Anything using it will stop working.`)) {
		return
	}
	saving.value = true
	errorText.value = ''
	try {
		await window.client.deleteApiKey({ apiKeyId: key.id })
		await load()
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to delete API key')
	} finally {
		saving.value = false
	}
}

onMounted(load)
</script>
