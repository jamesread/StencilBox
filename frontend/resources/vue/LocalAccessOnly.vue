<template>
	<Section title="Local access only" :icon="ComputerIcon" classes="small">
		<p>
			StencilBox is running without a database, so there are no user accounts to log in with.
			It only accepts requests made directly from the computer it runs on.
		</p>
		<p>To use StencilBox from here, either:</p>
		<ul>
			<li>open it from the computer it runs on, at <code>http://localhost:&lt;port&gt;/webui/</code>, or</li>
			<li>give it a database: remove <code>database.enabled: false</code> from <code>config.yaml</code>
				(or <code>STENCILBOX_DB_DISABLED</code>), make sure the config directory is writable, and restart.</li>
		</ul>
		<p class="subtle">
			Requests through a reverse proxy are never treated as local, even when the proxy runs on the same computer.
		</p>
		<a class="button inline-icon" href="https://jamesread.github.io/StencilBox/config/authentication.html">
			<HugeiconsIcon :icon="LinkSquare01Icon" width="1em" height="1em" :strokeWidth="2.5" aria-hidden="true" />
			<span>Authentication docs</span>
		</a>
	</Section>
</template>

<script setup>
import { HugeiconsIcon } from '@hugeicons/vue'
import { ComputerIcon, LinkSquare01Icon } from '@hugeicons/core-free-icons'
import Section from 'picocrank/vue/components/Section.vue'
</script>
