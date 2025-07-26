<template>
	<div class = "flex-row">
		<h2>{{ title }}</h2>
	</div>

	<p>
		{{ subtitle }}
	</p>
</template>

<script setup>
	defineProps({
		title: {
			type: String,
			required: true
		},
		subtitle: {
			type: String,
			default: ''
		}
	});
</script>
