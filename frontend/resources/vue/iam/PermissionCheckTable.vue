<template>
	<table class="data-table perm-table">
		<thead>
			<tr>
				<th scope="col" class="perm-check"><span class="a11yhidden">Grant</span></th>
				<th scope="col">Permission</th>
				<th scope="col">Description</th>
			</tr>
		</thead>
		<tbody>
			<tr v-for="p in sorted" :key="p.id">
				<td class="perm-check">
					<input
						v-model="selected"
						type="checkbox"
						:value="p.id"
						:disabled="disabled"
						:aria-label="'Grant ' + p.name"
					/>
				</td>
				<td><code>{{ p.name }}</code></td>
				<td>{{ p.description || '—' }}</td>
			</tr>
		</tbody>
	</table>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
	permissions: { type: Array, default: () => [] },
	disabled: { type: Boolean, default: false },
})

// The catalog comes from ListRbacPermissions; never hard-code it here.
const selected = defineModel({ type: Array, default: () => [] })
const sorted = computed(() => props.permissions.slice().sort((a, b) => a.name.localeCompare(b.name)))
</script>
