<template>
	<section>
		<SectionHeader title="Add Template" subtitle="Add a new template from a Git repository." />

		<form @submit.prevent="submitForm">
				<label for="repository-url">Git Repository URL</label>
				<input 
					type="text" 
					id="repository-url" 
					v-model="repositoryUrl" 
					placeholder="https://github.com/username/repository.git"
					required
				/>
				<small>Enter the Git repository URL for the template you want to add.</small>

			<fieldset>
				<button type="submit" :disabled="isSubmitting">
					{{ isSubmitting ? 'Adding...' : 'Add Template' }}
				</button>
				<button type="button" @click="cancel" :disabled="isSubmitting">
					Cancel
				</button>
			</fieldset>
		</form>
	</section>
</template>

<script setup>
	import { ref } from 'vue';
	import { useRouter } from 'vue-router';

	const router = useRouter();
	const repositoryUrl = ref('');
	const isSubmitting = ref(false);

	async function submitForm() {
		if (!repositoryUrl.value.trim()) {
			return;
		}

		isSubmitting.value = true;
		
		try {
			// TODO: Implement API call to add template
			console.log('Adding template from:', repositoryUrl.value);
			
			// Simulate API call
			await new Promise(resolve => setTimeout(resolve, 1000));
			
			// Navigate back to templates list
			router.push({ name: 'templateList' });
		} catch (error) {
			console.error('Error adding template:', error);
			// TODO: Show error message to user
		} finally {
			isSubmitting.value = false;
		}
	}

	function cancel() {
		router.push({ name: 'templateList' });
	}
</script>