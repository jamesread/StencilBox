<template>
	<Section title="Login" subtitle="Sign in to StencilBox with your username and password" :icon="Login03Icon" classes="small">
		<Login
			ref="loginRef"
			:show-default-tabs="false"
			:custom-tabs="[{ id: 'local', label: 'Username & Password' }]"
			loading-message="Signing in…"
			@local-login="onLocalLogin"
		/>
	</Section>
</template>

<script setup>
import { ref } from 'vue'
import Login from 'picocrank/vue/components/Login.vue'
import Section from 'picocrank/vue/components/Section.vue'
import { Login03Icon } from '@hugeicons/core-free-icons'

const emit = defineEmits(['login-success'])
const loginRef = ref(null)

async function onLocalLogin({ username, password }) {
	try {
		const res = await window.client.loginWithUsernameAndPassword({ username, password })
		if (res.success) {
			loginRef.value?.resetLocalForm()
			emit('login-success')
			return
		}
		loginRef.value?.setLocalLoginError(res.message || 'Login failed')
	} catch (e) {
		console.error('Login failed:', e)
		loginRef.value?.setLocalLoginError('Login failed')
	}
}
</script>
