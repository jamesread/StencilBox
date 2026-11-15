<template>
	<FormField label="Permissions" component-has-label description="A key never has more access than you do. If your access shrinks, so does the key's.">
		<RadioGroup
			:model-value="scope.restrictPermissions"
			name="api-key-permission-mode"
			:options="[
				{ value: false, label: 'Everything I can do' },
				{ value: true, label: 'Only the permissions I choose' },
			]"
			:disabled="disabled"
			@update:model-value="update({ restrictPermissions: $event })"
		/>
	</FormField>
	<FormField v-if="scope.restrictPermissions" label="Allowed permissions" component-has-label>
		<CheckGroup
			:model-value="scope.permissions"
			name="api-key-permissions"
			:options="permissionOptions"
			:disabled="disabled"
			@update:model-value="update({ permissions: $event })"
		/>
	</FormField>

	<FormField label="Build configs" component-has-label description="Limits building, and viewing configs, history and data files.">
		<RadioGroup
			:model-value="scope.restrictBuildConfigs"
			name="api-key-build-config-mode"
			:options="[
				{ value: false, label: 'All build configs' },
				{ value: true, label: 'Only the build configs I choose' },
			]"
			:disabled="disabled"
			@update:model-value="update({ restrictBuildConfigs: $event })"
		/>
	</FormField>
	<FormField v-if="scope.restrictBuildConfigs" label="Allowed build configs" component-has-label>
		<CheckGroup
			v-if="buildConfigOptions.length"
			:model-value="scope.buildConfigs"
			name="api-key-build-configs"
			:options="buildConfigOptions"
			:disabled="disabled"
			@update:model-value="update({ buildConfigs: $event })"
		/>
		<p v-else class="inline-notification note">There are no build configs to choose from.</p>
	</FormField>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import FormField from 'picocrank/vue/components/FormField.vue'
import RadioGroup from 'picocrank/vue/components/RadioGroup.vue'
import CheckGroup from 'picocrank/vue/components/CheckGroup.vue'
import { useAppStatus } from '../../../scripts/appStatus.js'

// What each permission means, for the people choosing them. Unknown names show as-is.
const PERMISSION_LABELS = {
	'builds.run': 'Start builds and clear build caches',
	'buildconfigs.view': 'View build configs, history, templates and data files',
	'buildconfigs.git-pull': 'Git pull the build configs directory',
	'datafiles.edit': 'Replace data files (rebuilds the sites that use them)',
	'buildconfigs.manage': 'Create build configs and add data files to them',
	'app.access': 'Other everyday use of StencilBox',
}

defineProps({
	disabled: { type: Boolean, default: false },
})

const scope = defineModel({ type: Object, required: true })
const { appStatus } = useAppStatus()
const buildConfigNames = ref([])

function update(patch) {
	scope.value = { ...scope.value, ...patch }
}

// The caller's own permissions, plus any already on the key (eg ones the caller has since lost).
const permissionOptions = computed(() => {
	const names = new Set([...(appStatus.value?.rbacPermissions || []), ...(scope.value.permissions || [])])
	return [...names].sort().map((name) => ({
		value: name,
		label: PERMISSION_LABELS[name] ? `${name}: ${PERMISSION_LABELS[name]}` : name,
	}))
})

const buildConfigOptions = computed(() => {
	const names = new Set([...buildConfigNames.value, ...(scope.value.buildConfigs || [])])
	return [...names].sort().map((name) => ({ value: name, label: name }))
})

onMounted(async () => {
	try {
		const res = await window.client.getBuildConfigs({})
		buildConfigNames.value = (res.buildConfigs || []).map((bc) => bc.name)
	} catch (e) {
		// Without buildconfigs.view only names already on the key are offered.
		console.debug('Could not list build configs:', e)
	}
})

</script>
