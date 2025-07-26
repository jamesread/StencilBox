<template>
		<ul>
			<li v-for = "link in links" :key = "link.name" :class = "currentSectionName === link.name ? 'active' : ''">
				<a href = "#" @click.prevent="showNavSection(link.name)">
					{{ link.title }}
				</a>
			</li>
		</ul>
</template>

<style>
    .active {
		text-decoration: underline;
	}
</style>

<script setup>
	import { ref, inject } from 'vue';

	const currentSectionName = ref('welcome');
	const changeSection = inject('changeSection');
	const links = ref([
	]);

	function addLink(title, name) {
		links.value.push({
			title: title,
			name: name,
			href: '#'
		});
	}

	function showNavSection(sectionName) {
	    currentSectionName.value = sectionName;
		changeSection(sectionName);
	}

	defineExpose({
		addLink
	});
</script>
