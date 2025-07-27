<template>
	<div class = "flex-row">
		<div class = "fg1">
			<h2>{{ title }}</h2>

			<p>
				{{ subtitle }}
			</p>
		</div>

		<div>
			<slot name = "actions" />
		</div>
	</div>
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
