<template>
	<Dialog ref="dialog" title="Create build config" size="medium" @close="reset">
		<p>A build config builds one site: a template, filled in with data files, written to an output directory.</p>

		<FormLayout id="create-build-config-form" @submit.prevent="submit">
			<FormField label="Name" for="new-config-name">
				<input id="new-config-name" ref="nameInput" v-model="form.name" type="text" autocomplete="off" required :disabled="saving" @input="syncOutputDir" />
			</FormField>

			<FormField label="Template" for="new-config-template">
				<select id="new-config-template" v-model="form.template" required :disabled="saving || !templates.length">
					<option value="" disabled>Choose a template</option>
					<option v-for="t in templates" :key="t.name" :value="t.name">
						{{ t.name }}{{ t.description ? ' — ' + t.description : '' }}
					</option>
				</select>
			</FormField>

			<FormField
				label="Output directory"
				for="new-config-output"
				:description="'The site is served at /' + (form.outputDir || '…') + '/ on this server.'"
			>
				<input id="new-config-output" v-model="form.outputDir" type="text" autocomplete="off" required :disabled="saving" @input="outputDirEdited = true" />
			</FormField>

			<FormField label="Build on startup" component-has-label>
				<label class="checkbox-label">
					<input v-model="form.buildOnStartup" type="checkbox" :disabled="saving" />
					Rebuild this site whenever StencilBox starts
				</label>
			</FormField>

			<NotificationBlock v-if="errorText" type="error" :message="errorText" />
		</FormLayout>

		<template #actions="{ close }">
			<button type="button" class="neutral" @click="close()">Cancel</button>
			<button type="submit" form="create-build-config-form" class="good" :disabled="!canSubmit">
				{{ saving ? 'Creating…' : 'Create build config' }}
			</button>
		</template>
	</Dialog>
</template>

<script setup>
import { computed, nextTick, reactive, ref } from 'vue'
import Dialog from 'picocrank/vue/components/Dialog.vue'
import FormLayout from 'picocrank/vue/components/FormLayout.vue'
import FormField from 'picocrank/vue/components/FormField.vue'
import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue'
import { errorMessage } from '../../scripts/appStatus.js'

const emit = defineEmits(['created'])

const dialog = ref(null)
const nameInput = ref(null)
const templates = ref([])
const saving = ref(false)
const errorText = ref('')
const outputDirEdited = ref(false)
const form = reactive({ name: '', template: '', outputDir: '', buildOnStartup: true })

const canSubmit = computed(() => !saving.value && form.name.trim() && form.template && form.outputDir.trim())

function slug(s) {
	return s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '')
}

// The output directory follows the name until it is edited by hand.
function syncOutputDir() {
	if (!outputDirEdited.value) {
		form.outputDir = slug(form.name)
	}
}

function reset() {
	form.name = ''
	form.template = ''
	form.outputDir = ''
	form.buildOnStartup = true
	outputDirEdited.value = false
	errorText.value = ''
}

async function open() {
	reset()
	dialog.value.show()
	nextTick(() => nameInput.value?.focus())
	try {
		const res = await window.client.getTemplates({})
		templates.value = (res.templates || []).slice().sort((a, b) => a.name.localeCompare(b.name))
	} catch (e) {
		errorText.value = errorMessage(e, 'Could not load templates')
	}
}

async function submit() {
	saving.value = true
	errorText.value = ''
	try {
		const res = await window.client.createBuildConfig({
			configName: form.name.trim(),
			template: form.template,
			outputDir: form.outputDir.trim(),
			buildOnStartup: form.buildOnStartup,
		})
		dialog.value.close()
		emit('created', res.buildConfig.name)
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to create the build config')
	} finally {
		saving.value = false
	}
}

defineExpose({ open })
</script>
