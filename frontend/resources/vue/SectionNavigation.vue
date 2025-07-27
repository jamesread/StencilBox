<template>
		<ul>
			<li v-for = "link in links" :key = "link.name" :class = "$route.path === link.href ? 'active' : ''">
				<router-link :to = "link.href">
					<HugeiconsIcon :icon="link.icon" :size="24" />
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
	import { HugeiconsIcon } from '@hugeicons/vue';

	const $route = useRouter();
	const links = ref([]);

	function addLink(title, href, icon) {
		links.value.push({
			title: title,
			name: href,
			href: href,
			icon: icon
		});
	}

	defineExpose({
		addLink
	});
</script>
