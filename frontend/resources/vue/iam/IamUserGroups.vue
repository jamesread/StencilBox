<template>
	<dialog ref="createDialog" class="iam-dialog" @close="newName = ''; createError = ''">
		<h2>Create group</h2>
		<FormLayout @submit.prevent="createGroup">
			<FormField label="Name" for="new-group-name">
				<input id="new-group-name" ref="nameInput" v-model="newName" type="text" autocomplete="off" required :disabled="saving" />
			</FormField>
			<NotificationBlock v-if="createError" type="error" :message="createError" />
			<template #actions>
				<button type="button" class="neutral" @click="createDialog.close()">Cancel</button>
				<button type="submit" class="good" :disabled="saving || !newName.trim()">{{ saving ? 'Creating…' : 'Create group' }}</button>
			</template>
		</FormLayout>
	</dialog>

	<Section
		title="User Groups"
		subtitle="Groups carry roles to their members. Every user is in Everyone."
		:icon="UserGroupIcon"
		:padding="false"
	>
		<template #toolbar>
			<button type="button" class="inline-icon neutral" aria-label="Refresh" :disabled="loading" @click="load">
				<HugeiconsIcon :icon="RefreshIcon" width="1em" height="1em" :strokeWidth="2.5" />
			</button>
			<button v-if="canManage" type="button" class="inline-icon good" aria-label="Create group" :disabled="loading" @click="openCreate">
				<HugeiconsIcon :icon="Add01Icon" width="1em" height="1em" :strokeWidth="2.5" />
			</button>
		</template>

		<div v-if="errorText" class="padded">
			<NotificationBlock type="error" :message="errorText" />
		</div>
		<p v-if="loading && !groups.length" class="padded">Loading…</p>
		<Table
			v-else
			:data="groups"
			:headers="headers"
			row-key="id"
			:filterable="false"
			:column-options="false"
			@row-click="openGroup"
		>
			<template #cell-name="{ row }">
				<strong>{{ row.name }}</strong>
				<span v-if="isSystemGroup(row.name)" class="tag note">system group</span>
			</template>
			<template #cell-actions="{ row }">
				<button
					v-if="!isSystemGroup(row.name)"
					type="button"
					class="bad small"
					data-row-click-ignore
					:disabled="saving"
					@click.stop="deleteGroup(row)"
				>Delete</button>
			</template>
		</Table>
	</Section>
</template>

<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { HugeiconsIcon } from '@hugeicons/vue'
import { Add01Icon, RefreshIcon, UserGroupIcon } from '@hugeicons/core-free-icons'
import Section from 'picocrank/vue/components/Section.vue'
import Table from 'picocrank/vue/components/Table.vue'
import FormLayout from 'picocrank/vue/components/FormLayout.vue'
import FormField from 'picocrank/vue/components/FormField.vue'
import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue'
import { useAppStatus, errorMessage } from '../../../scripts/appStatus.js'
import { isSystemGroup } from '../../../scripts/rbacAccess.js'

const router = useRouter()
const { hasPermission } = useAppStatus()

const groups = ref([])
const loading = ref(true)
const saving = ref(false)
const errorText = ref('')
const createDialog = ref(null)
const nameInput = ref(null)
const newName = ref('')
const createError = ref('')

const canManage = computed(() => hasPermission('usergroups.manage'))
const headers = computed(() => {
	const h = [
		{ key: 'name', label: 'Name', sortable: true },
		{ key: 'memberCount', label: 'Members', sortable: true, width: '8rem' },
	]
	if (canManage.value) {
		h.push({ key: 'actions', label: '', width: '6rem' })
	}
	return h
})

async function load() {
	loading.value = true
	errorText.value = ''
	try {
		const res = await window.client.listUserGroups({})
		groups.value = res.groups || []
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to load groups')
	} finally {
		loading.value = false
	}
}

function openGroup({ row }) {
	router.push({ name: 'iamGroupDetails', params: { id: String(row.id) } })
}

function openCreate() {
	newName.value = ''
	createError.value = ''
	createDialog.value.showModal()
	nextTick(() => nameInput.value?.focus())
}

async function createGroup() {
	saving.value = true
	createError.value = ''
	try {
		const res = await window.client.createUserGroup({ name: newName.value.trim() })
		createDialog.value.close()
		router.push({ name: 'iamGroupDetails', params: { id: String(res.group.id) } })
	} catch (e) {
		createError.value = errorMessage(e, 'Failed to create group')
	} finally {
		saving.value = false
	}
}

async function deleteGroup(group) {
	if (!confirm(`Delete group "${group.name}"? Its members lose the roles it grants.`)) {
		return
	}
	saving.value = true
	errorText.value = ''
	try {
		await window.client.deleteUserGroup({ groupId: group.id })
		await load()
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to delete group')
	} finally {
		saving.value = false
	}
}

onMounted(load)
</script>
