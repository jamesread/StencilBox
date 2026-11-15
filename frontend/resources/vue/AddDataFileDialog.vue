<template>
	<Dialog ref="dialog" title="Add data file" size="large" class="add-data-file-dialog" @close="reset">
		<FormLayout id="add-data-file-form" @submit.prevent="submit">
			<FormField label="Name" for="new-datafile-name" description="Templates read the data by this name, eg links. Letters, digits and underscores.">
				<input id="new-datafile-name" ref="nameInput" v-model="form.name" type="text" autocomplete="off" required :disabled="saving" />
			</FormField>

			<FormField label="File" component-has-label>
				<RadioGroup
					v-model="form.useExisting"
					name="datafile-source"
					:options="[
						{ value: false, label: 'Create a new file' },
						{ value: true, label: 'Use an existing file' },
					]"
					:disabled="saving"
				/>
			</FormField>

			<FormField
				label="Path"
				for="new-datafile-path"
				:description="form.useExisting
					? 'Relative to the build configs directory, eg data/sites.yml. Data files can be shared between build configs.'
					: 'Optional. Relative to the build configs directory. Leave empty for ' + defaultPath + '.'"
			>
				<input
					id="new-datafile-path"
					v-model="form.path"
					type="text"
					autocomplete="off"
					:placeholder="form.useExisting ? 'data/sites.yml' : defaultPath"
					:required="form.useExisting"
					:disabled="saving"
				/>
			</FormField>

			<FormField v-if="!form.useExisting" label="Content" component-has-label>
				<YamlEditor v-model="form.content" :rows="14" aria-label="New data file content" @validity="valid = $event" />
			</FormField>

			<NotificationBlock v-if="errorText" type="error" :message="errorText" />
		</FormLayout>

		<template #actions="{ close }">
			<button type="button" class="neutral" @click="close()">Cancel</button>
			<button type="submit" form="add-data-file-form" class="good" :disabled="!canSubmit">
				{{ saving ? 'Adding…' : 'Add data file' }}
			</button>
		</template>
	</Dialog>
</template>

<script setup>
import { computed, nextTick, reactive, ref } from 'vue'
import Dialog from 'picocrank/vue/components/Dialog.vue'
import FormLayout from 'picocrank/vue/components/FormLayout.vue'
import FormField from 'picocrank/vue/components/FormField.vue'
import RadioGroup from 'picocrank/vue/components/RadioGroup.vue'
import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue'
import YamlEditor from './YamlEditor.vue'
import { errorMessage } from '../../scripts/appStatus.js'

const props = defineProps({
	buildConfigName: { type: String, required: true },
})

const emit = defineEmits(['created'])

const STARTER = '# Data for this site. Templates read it by the name above.\n'

const dialog = ref(null)
const nameInput = ref(null)
const saving = ref(false)
const valid = ref(true)
const errorText = ref('')
const form = reactive({ name: '', useExisting: false, path: '', content: STARTER })

function slug(s) {
	return s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '') || 'data'
}

// Mirrors the server's default (buildconfigs.Slug), so the placeholder shows the real path.
const defaultPath = computed(() => `data/${slug(props.buildConfigName)}-${slug(form.name || 'name')}.yaml`)

const canSubmit = computed(() => {
	if (saving.value || !form.name.trim()) {
		return false
	}
	if (form.useExisting) {
		return Boolean(form.path.trim())
	}
	return valid.value && form.content.trim() !== ''
})

function reset() {
	form.name = ''
	form.useExisting = false
	form.path = ''
	form.content = STARTER
	errorText.value = ''
}

function open() {
	reset()
	dialog.value.show()
	nextTick(() => nameInput.value?.focus())
}

async function submit() {
	saving.value = true
	errorText.value = ''
	try {
		await window.client.createDataFile({
			buildConfigName: props.buildConfigName,
			datafileName: form.name.trim(),
			path: form.path.trim(),
			content: form.useExisting ? '' : form.content,
		})
		const name = form.name.trim()
		dialog.value.close()
		emit('created', name)
	} catch (e) {
		errorText.value = errorMessage(e, 'Failed to add the data file')
	} finally {
		saving.value = false
	}
}

defineExpose({ open })
</script>
