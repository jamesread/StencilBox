<template>
    <div id = "breadcrumbs">
        <a v-for = "link in links" :key = "link.name" :href = "link.href" @click.prevent = "goTo(link.name)">{{ link.name }}</a>
    </div>
</template>

<script setup>
    import { ref } from 'vue';

    const links = ref([]);

    function addLink(name, href) {
        links.value.push({ name, href });
    }

    defineExpose({
        addLink
    });
</script>
