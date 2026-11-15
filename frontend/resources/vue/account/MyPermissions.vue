<template>
	<Section title="My Permissions" subtitle="Your group membership and effective permissions." :icon="ShieldKeyIcon">
		<template #toolbar>
			<button type="button" class="inline-icon neutral" aria-label="Refresh" :disabled="loading" @click="load">
				<HugeiconsIcon :icon="RefreshIcon" width="1em" height="1em" :strokeWidth="2.5" />
			</button>
		</template>

		<NotificationBlock v-if="errorText" type="error" :message="errorText" />
		<p v-else-if="loading">Loading…</p>
		<template v-else-if="audit">
			<h3>Groups</h3>
			<p v-if="!audit.groupNames.length" class="inline-notification note">You are not in any group.</p>
			<p v-else class="inline-tags">
				<span v-for="name in audit.groupNames" :key="name" class="tag note">{{ name }}</span>
			</p>

			<h3>Roles</h3>
			<p v-if="!audit.roleNames.length" class="inline-notification note">No roles through group membership.</p>
			<p v-else class="inline-tags">
				<span v-for="name in audit.roleNames" :key="name" class="tag">{{ name }}</span>
				<span v-if="audit.isSuperuser" class="tag good">all access</span>
			</p>

			<h3>Permissions</h3>
			<p v-if="audit.isSuperuser" class="inline-notification note">
				You have the <strong>superuser</strong> role, so every permission is granted.
			</p>
			<PermissionAuditTable :rows="audit.permissions" :is-superuser="audit.isSuperuser" />
		</template>
	</Section>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { HugeiconsIcon } from '@hugeicons/vue'
import { RefreshIcon, ShieldKeyIcon } from '@hugeicons/core-free-icons'
import Section from 'picocrank/vue/components/Section.vue'
import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue'
import PermissionAuditTable from '../iam/PermissionAuditTable.vue'
import { errorMessage } from '../../../scripts/appStatus.js'

const audit = ref(null)
const loading = ref(true)
const errorText = ref('')

async function load() {
	loading.value = true
	errorText.value = ''
	try {
		audit.value = await window.client.getMyPermissionsAudit({})
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to load your permissions')
	} finally {
		loading.value = false
	}
}

onMounted(load)
</script>
