<template>
    <Section :title = "'View Template: ' + template?.name">
        <dl>
            <dt>Name</dt>
            <dd>{{ template?.name }}</dd>

			<dt>Description</dt>
			<dd>{{ template?.description }}</dd>

            <dt>Source</dt>
            <dd>{{ template?.source }}</dd>

            <dt>Documentation</dt>
            <dd>
                <a :href="template?.documentationUrl" target="_blank">{{ template?.documentationUrl }}</a>
            </dd>
        </dl>

	</Section>
	<Section title="Associated Build Configs">
        <p>The following build configs are associated with this template:</p>

        <p v-if="template?.buildConfigs && template.buildConfigs.length === 0">
            No build configs are associated with this template.
        </p>
        <ul v-else>
            <li v-for="buildConfig in template?.buildConfigs" :key="buildConfig">
				<router-link :to="{ name: 'buildConfig', params: { name: buildConfig } }">
					{{ buildConfig }}
				</router-link>
            </li>
        </ul>
    </Section>
</template>

<style scoped>
    </style>

<script setup>
	import { ref, onMounted } from 'vue';
	import Section from 'picocrank/vue/components/Section.vue';

    const props = defineProps({
        name: {
            type: String,
            required: true
        }
    });

    const template = ref(null);

    onMounted(() => {
        loadTemplate();
    });

    async function loadTemplate() {
        try {
            const response = await window.client.getTemplate({
                templateName: props.name
            });
            template.value = response.template;
        } catch (error) {
            console.error('Error loading template:', error);
        }
    }
</script>
