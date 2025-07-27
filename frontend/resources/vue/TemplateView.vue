<template>
    <section>
        <SectionHeader :title = "'View Template: ' + template?.name" />

        <dl>
            <dt>Name</dt>
            <dd>{{ template?.name }}</dd>

            <dt>Source</dt>
            <dd>{{ template?.source }}</dd>

            <dt>Documentation</dt>
            <dd>
                <a :href="template?.documentationUrl" target="_blank">{{ template?.documentationUrl }}</a>
            </dd>
        </dl>

        <h2>Build Configs</h2>
        <p>The following build configs are associated with this template:</p>

        <p v-if="template?.buildConfigs && template.buildConfigs.length === 0">
            No build configs are associated with this template.
        </p>
        <ul v-else>
            <li v-for="buildConfig in template?.buildConfigs" :key="buildConfig">
                <a :href="`/build-config/${buildConfig}`">{{ buildConfig }}</a>
            </li>
        </ul>
    </section>
</template>

<style scoped>
    </style>

<script setup>
	import { ref, onMounted } from 'vue';

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
