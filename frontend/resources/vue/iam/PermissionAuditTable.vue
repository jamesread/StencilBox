<template>
	<table class="data-table perm-table">
		<thead>
			<tr>
				<th scope="col" class="perm-status">Status</th>
				<th scope="col">Permission</th>
				<th scope="col">Granted by groups</th>
			</tr>
		</thead>
		<tbody>
			<tr v-for="row in rows" :key="row.permission">
				<td class="perm-status">
					<span v-if="row.granted" class="tag good">granted</span>
					<span v-else class="tag bad">denied</span>
				</td>
				<td><code>{{ row.permission }}</code></td>
				<td>
					<span class="inline-tags">
						<span v-if="isSuperuser && !row.grantingGroups.length" class="tag good">superuser</span>
						<span v-for="name in row.grantingGroups" :key="name" class="tag note">{{ name }}</span>
						<span v-if="!isSuperuser && !row.grantingGroups.length">—</span>
					</span>
				</td>
			</tr>
		</tbody>
	</table>
</template>

<script setup>
defineProps({
	rows: { type: Array, default: () => [] },
	isSuperuser: { type: Boolean, default: false },
})
</script>
