<template>
		<ul>
			<li v-for = "link in links" :key = "link.name" :class = "$route.path === link.href ? 'active' : ''">
				<router-link :to = "link.href">
					{{ link.title }}
				</router-link>
			</li>
		</ul>
</template>

<style>
    .active {
		text-decoration: underline;
	}
</style>

<script setup>
	import { ref } from 'vue';
	import { useRouter } from 'vue-router';

	const $route = useRouter();
	const links = ref([]);

	function addLink(title, href) {
		links.value.push({
			title: title,
			name: href,
			href: href
		});
	}

	defineExpose({
		addLink
	});
</script>
